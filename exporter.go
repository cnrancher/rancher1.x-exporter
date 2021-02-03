package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

const (
	// Used to prepand Prometheus metrics created by this exporter.
	namespace  = "rancher"
	specialTag = "__rancher__"
)

var (
	/**
	InfinityWorks
	*/

	agentStates   = []string{"activating", "active", "reconnecting", "disconnected", "disconnecting", "finishing-reconnect", "reconnected"}
	hostStates    = []string{"activating", "active", "deactivating", "error", "erroring", "inactive", "provisioned", "purged", "purging", "registering", "removed", "removing", "requested", "restoring", "updating_active", "updating_inactive"}
	stackStates   = []string{"activating", "active", "canceled_upgrade", "canceling_upgrade", "error", "erroring", "finishing_upgrade", "removed", "removing", "requested", "restarting", "rolling_back", "updating_active", "upgraded", "upgrading"}
	serviceStates = []string{"activating", "active", "canceled_upgrade", "canceling_upgrade", "deactivating", "finishing_upgrade", "inactive", "registering", "removed", "removing", "requested", "restarting", "rolling_back", "updating_active", "updating_inactive", "upgraded", "upgrading"}
	healthStates  = []string{"healthy", "unhealthy"}

	projectID, projectName string
	hc                     *httpClient
)

type buffMsg struct {
	class         string
	id            string
	parentId      string
	name          string
	state         string
	healthState   string
	transitioning string
	stackName     string
	serviceName   string
}

/**
RancherExporter
*/
type rancherExporter struct {
	mutex         *sync.Mutex
	websocketConn *websocket.Conn

	msgBuff       chan buffMsg
	stacksBuff    chan buffMsg
	servicesBuff  chan buffMsg
	instancesBuff chan buffMsg

	recreateWebsocket func() *websocket.Conn
}

func (r *rancherExporter) Describe(ch chan<- *prometheus.Desc) {
	infinityWorksStacksHealth.Describe(ch)
	infinityWorksStacksState.Describe(ch)
	infinityWorksServicesScale.Describe(ch)
	infinityWorksServicesHealth.Describe(ch)
	infinityWorksServicesState.Describe(ch)
	infinityWorksHostsState.Describe(ch)
	infinityWorksHostAgentsState.Describe(ch)

	extendingTotalStackInitializations.Describe(ch)
	extendingTotalSuccessStackInitialization.Describe(ch)
	extendingTotalErrorStackInitialization.Describe(ch)
	extendingTotalServiceInitializations.Describe(ch)
	extendingTotalSuccessServiceInitialization.Describe(ch)
	extendingTotalErrorServiceInitialization.Describe(ch)
	extendingTotalInstanceInitializations.Describe(ch)
	extendingTotalSuccessInstanceInitialization.Describe(ch)
	extendingTotalErrorInstanceInitialization.Describe(ch)

	extendingTotalStackBootstraps.Describe(ch)
	extendingTotalSuccessStackBootstrap.Describe(ch)
	extendingTotalErrorStackBootstrap.Describe(ch)
	extendingTotalServiceBootstraps.Describe(ch)
	extendingTotalSuccessServiceBootstrap.Describe(ch)
	extendingTotalErrorServiceBootstrap.Describe(ch)
	extendingTotalInstanceBootstraps.Describe(ch)
	extendingTotalSuccessInstanceBootstrap.Describe(ch)
	extendingTotalErrorInstanceBootstrap.Describe(ch)
	extendingInstanceBootstrapMsCost.Describe(ch)

	extendingInstanceHeartbeat.Describe(ch)
	extendingServiceHeartbeat.Describe(ch)
	extendingStackHeartbeat.Describe(ch)
}

func (r *rancherExporter) Collect(ch chan<- prometheus.Metric) {
	r.asyncMetrics(ch)

	r.syncMetrics(ch)
}

func (r *rancherExporter) Stop() {
	if r.websocketConn != nil {
		r.websocketConn.Close()
	}

	close(r.instancesBuff)
	close(r.servicesBuff)
	close(r.stacksBuff)
}

func (r *rancherExporter) asyncMetrics(ch chan<- prometheus.Metric) {
	// collect
	extendingTotalStackBootstraps.Collect(ch)
	extendingTotalSuccessStackBootstrap.Collect(ch)
	extendingTotalErrorStackBootstrap.Collect(ch)
	extendingTotalStackInitializations.Collect(ch)
	extendingTotalSuccessStackInitialization.Collect(ch)
	extendingTotalErrorStackInitialization.Collect(ch)

	extendingTotalServiceBootstraps.Collect(ch)
	extendingTotalSuccessServiceBootstrap.Collect(ch)
	extendingTotalErrorServiceBootstrap.Collect(ch)
	extendingTotalServiceInitializations.Collect(ch)
	extendingTotalSuccessServiceInitialization.Collect(ch)
	extendingTotalErrorServiceInitialization.Collect(ch)

	extendingTotalInstanceBootstraps.Collect(ch)
	extendingTotalSuccessInstanceBootstrap.Collect(ch)
	extendingTotalErrorInstanceBootstrap.Collect(ch)
	extendingTotalInstanceInitializations.Collect(ch)
	extendingTotalSuccessInstanceInitialization.Collect(ch)
	extendingTotalErrorInstanceInitialization.Collect(ch)

	extendingInstanceBootstrapMsCost.Collect(ch)
}

func (r *rancherExporter) syncMetrics(ch chan<- prometheus.Metric) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorln(err)
		}
	}()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	infinityWorksHostsState.Reset()
	infinityWorksHostAgentsState.Reset()
	infinityWorksStacksHealth.Reset()
	infinityWorksStacksState.Reset()
	extendingStackHeartbeat.Reset()
	infinityWorksServicesScale.Reset()
	infinityWorksServicesHealth.Reset()
	infinityWorksServicesState.Reset()
	extendingServiceHeartbeat.Reset()
	extendingInstanceHeartbeat.Reset()

	gwg := &sync.WaitGroup{}

	// collect host metrics
	gwg.Add(1)
	go func() {
		defer gwg.Done()
		if err := hc.foreachCollection(hostSubpath, nil, gwg, setHostMetrics); err != nil {
			logger.Warnf("failed to set host metrics, %v", err)
		}
	}()

	gwg.Add(1)
	go func() {
		stwg := &sync.WaitGroup{}
		swg := &sync.WaitGroup{}
		stackMap := &sync.Map{}
		serviceMap := &sync.Map{}
		defer gwg.Done()
		// collect stack metrics
		if err := hc.foreachCollection(stackSubpath, nil, stwg, func(data []byte) {
			stackID, stackName := setStackMetrics(data)
			stackMap.Store(stackID, stackName)
		}); err != nil {
			logger.Warnf("failed to set stack metrics, %v", err)
		}
		stwg.Wait()

		// collect service metrics
		if err := hc.foreachCollection(serviceSubpath, nil, swg, func(data []byte) {
			serviceID, content := setServiceMetrics(stackMap, data)
			serviceMap.Store(serviceID, content)
		}); err != nil {
			logger.Warnf("failed to set service metrics, %v", err)
		}
		swg.Wait()

		// collect instance metrics

		if err := hc.foreachCollection(instanceSubpath, nil, gwg, func(data []byte) {
			setInstanceMetrics(serviceMap, data)
		}); err != nil {
			logrus.Warnf("failed to set instance metrics, %v", err)
		}
	}()

	gwg.Wait()

	// collect
	infinityWorksHostsState.Collect(ch)
	infinityWorksHostAgentsState.Collect(ch)
	infinityWorksStacksHealth.Collect(ch)
	infinityWorksStacksState.Collect(ch)
	extendingStackHeartbeat.Collect(ch)
	infinityWorksServicesScale.Collect(ch)
	infinityWorksServicesHealth.Collect(ch)
	infinityWorksServicesState.Collect(ch)
	extendingServiceHeartbeat.Collect(ch)
	extendingInstanceHeartbeat.Collect(ch)

}

func (r *rancherExporter) collectingExtending() {
	stackMap, _ := loadAndInitAggregatedMetrics()

	// event watcher
	go func() {
		for {
		recall:
			_, messageBytes, err := r.websocketConn.ReadMessage()
			if err != nil {
				logger.Warnln("reconnect websocket")
				r.websocketConn = r.recreateWebsocket()
				goto recall
			}

			if resourceType, _ := jsonparser.GetString(messageBytes, "resourceType"); len(resourceType) != 0 {
				resourceBytes, _, _, err := jsonparser.Get(messageBytes, "data", "resource")
				if err != nil {
					logger.Warnln(err)
					continue
				}

				baseType, _ := jsonparser.GetString(resourceBytes, "baseType")
				id, _ := jsonparser.GetString(resourceBytes, "id")
				name, _ := jsonparser.GetString(resourceBytes, "name")
				state, _ := jsonparser.GetString(resourceBytes, "state")
				healthState, _ := jsonparser.GetString(resourceBytes, "healthState")
				transitioning, _ := jsonparser.GetString(resourceBytes, "transitioning")

				switch baseType {
				case "stack":
					stackMap.LoadOrStore(id, name)

					r.msgBuff <- buffMsg{
						class:         "stack",
						id:            id,
						name:          name,
						state:         state,
						healthState:   healthState,
						transitioning: transitioning,
					}
				case "service":
					stackId, _ := jsonparser.GetString(resourceBytes, "stackId")
					stackName := ""
					if val, ok := stackMap.Load(stackId); ok {
						stackName = val.(string)
					} else {
						if stackRespBytes, err := hc.getByProject(path.Join(stackSubpath, stackId), nil); err == nil {
							stackName, _ = jsonparser.GetString(stackRespBytes, "name")
							stackMap.LoadOrStore(stackId, stackName)
						}
					}

					r.msgBuff <- buffMsg{
						class:         "service",
						id:            id,
						name:          name,
						state:         state,
						healthState:   healthState,
						transitioning: transitioning,
						parentId:      stackId,
						stackName:     stackName,
					}
				case "instance":
					labelStackServiceName, _ := jsonparser.GetString(resourceBytes, "labels", "io.rancher.stack_service.name")
					labelStackServiceNameSplit := strings.Split(labelStackServiceName, "/")

					serviceId, _ := jsonparser.GetString(resourceBytes, "serviceIds", "[0]")
					var serviceName string
					stackName := labelStackServiceNameSplit[0]
					if len(labelStackServiceNameSplit) > 1 {
						serviceName = labelStackServiceNameSplit[1]
					}

					r.msgBuff <- buffMsg{
						class:         "instance",
						id:            id,
						name:          name,
						state:         state,
						healthState:   healthState,
						transitioning: transitioning,
						stackName:     stackName,
						parentId:      serviceId,
						serviceName:   serviceName,
					}
				}
			}
		}

	}()

	// msg event handler
	go func() {
		type state uint64

		const (
			stk_active_initializing state = iota
			//stk_active_degraded
			stk_active_unhealthy

			svc_activating_healthy
			svc_active_initializing
			svc_restarting
			svc_upgrading

			ins_starting
			ins_stopping
			ins_stopped
			ins_running_reinitializing
		)

		stkMap := make(map[string]state)
		svcMap := make(map[string]state)
		insMap := make(map[string]state)
		svcParentIdMap := make(map[string]state)

		stkCount := func(stackMsg *buffMsg) {
			extendingTotalStackBootstraps.WithLabelValues(projectName, specialTag).Inc()
			extendingTotalStackBootstraps.WithLabelValues(projectName, stackMsg.name).Inc()

			extendingTotalSuccessStackBootstrap.WithLabelValues(projectName, specialTag)
			extendingTotalSuccessStackBootstrap.WithLabelValues(projectName, stackMsg.name)

			extendingTotalErrorStackBootstrap.WithLabelValues(projectName, specialTag)
			extendingTotalErrorStackBootstrap.WithLabelValues(projectName, stackMsg.name)

			logger.Infof("stack [%s] be count + 1", stackMsg.name)
		}
		stkSuccess := func(stackMsg *buffMsg) {
			extendingTotalSuccessStackBootstrap.WithLabelValues(projectName, specialTag).Inc()
			extendingTotalSuccessStackBootstrap.WithLabelValues(projectName, stackMsg.name).Inc()

			logger.Infof("stack [%s] be success + 1", stackMsg.name)
			delete(stkMap, stackMsg.id)
		}
		stkFail := func(stackMsg *buffMsg) {
			extendingTotalErrorStackBootstrap.WithLabelValues(projectName, specialTag).Inc()
			extendingTotalErrorStackBootstrap.WithLabelValues(projectName, stackMsg.name).Inc()

			logger.Infof("stack [%s] be error + 1", stackMsg.name)
			delete(stkMap, stackMsg.id)
		}

		svcCount := func(serviceMsg *buffMsg) {
			extendingTotalServiceBootstraps.WithLabelValues(projectName, specialTag, specialTag).Inc()
			extendingTotalServiceBootstraps.WithLabelValues(projectName, serviceMsg.stackName, specialTag).Inc()
			extendingTotalServiceBootstraps.WithLabelValues(projectName, serviceMsg.stackName, serviceMsg.name).Inc()

			extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, specialTag, specialTag)
			extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, serviceMsg.stackName, specialTag)
			extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, serviceMsg.stackName, serviceMsg.name)

			extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, specialTag, specialTag)
			extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, serviceMsg.stackName, specialTag)
			extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, serviceMsg.stackName, serviceMsg.name)

			logger.Infof("service [%s] be count + 1", serviceMsg.name)
		}
		svcSuccess := func(serviceMsg *buffMsg) {
			extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, specialTag, specialTag).Inc()
			extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, serviceMsg.stackName, specialTag).Inc()
			extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, serviceMsg.stackName, serviceMsg.name).Inc()

			logger.Infof("service [%s] be success + 1", serviceMsg.name)
			delete(svcMap, serviceMsg.id)
		}
		svcFail := func(serviceMsg *buffMsg) {
			extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, specialTag, specialTag).Inc()
			extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, serviceMsg.stackName, specialTag).Inc()
			extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, serviceMsg.stackName, serviceMsg.name).Inc()

			logger.Infof("service [%s] be error + 1", serviceMsg.name)
			delete(svcMap, serviceMsg.id)
		}

		insCount := func(instanceMsg *buffMsg) {
			extendingTotalInstanceBootstraps.WithLabelValues(projectName, specialTag, specialTag, specialTag).Inc()
			extendingTotalInstanceBootstraps.WithLabelValues(projectName, instanceMsg.stackName, specialTag, specialTag).Inc()
			extendingTotalInstanceBootstraps.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, specialTag).Inc()
			extendingTotalInstanceBootstraps.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, instanceMsg.name).Inc()

			extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, specialTag, specialTag, specialTag)
			extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, specialTag, specialTag)
			extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, specialTag)
			extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, instanceMsg.name)

			extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, specialTag, specialTag, specialTag)
			extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, specialTag, specialTag)
			extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, specialTag)
			extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, instanceMsg.name)

			logger.Infof("instance [%s] be count + 1", instanceMsg.name)
		}
		insSuccess := func(instanceMsg *buffMsg) {
			extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, specialTag, specialTag, specialTag).Inc()
			extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, specialTag, specialTag).Inc()
			extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, specialTag).Inc()
			extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, instanceMsg.name).Inc()

			logger.Infof("instance [%s] be success + 1", instanceMsg.name)
			delete(insMap, instanceMsg.id)
		}
		insFail := func(instanceMsg *buffMsg) {
			extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, specialTag, specialTag, specialTag).Inc()
			extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, specialTag, specialTag).Inc()
			extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, specialTag).Inc()
			extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, instanceMsg.stackName, instanceMsg.serviceName, instanceMsg.name).Inc()

			logger.Infof("instance [%s] be fail + 1", instanceMsg.name)
			delete(insMap, instanceMsg.id)
		}

		for msg := range r.msgBuff {
			logger.Debugf("[[%s]]: %+v", msg.class, msg)
			switch msg.class {
			case "stack":
				// stack 1 service with 1 container with hc
				// create: 					active(initializing) -> active(healthy)
				// restart container:		active(initializing) -> active(healthy)
				// stop container:			active(initializing) -> active(healthy)
				// restart service:			active(initializing) -> active(healthy)
				// stop service:			active(initializing) -> active(healthy)
				// stop stack               active(initializing) -> active(healthy)
				// upgrade service: 		active(initializing) -> active(healthy)
				// rollback service:		active(initializing) -> active(healthy)

				// stack add 1 service with 2 container with hc
				// restart container: 		active(initializing) -> active(healthy)
				// stop container:			active(initializing) -> active(healthy)
				// restart service:			active(initializing) -> active(healthy)
				// stop service:			active(initializing) -> active(healthy)
				// stop stack:				active(initializing) -> active(healthy)
				// upgrade service:			{ active(unhealthy) -> active(initializing) }-> active(initializing) -> active(healthy)
				// rollback service:		active(initializing) -> active(healthy)

				// stack add 2 service with 2 container with hc
				// restart container:		active(initializing) -> active(healthy)
				// stop container:			active(initializing) -> active(healthy)
				// restart service:			active(initializing) -> active(healthy)
				// stop service:			active(initializing) -> active(healthy)
				// stop stack:				active(initializing) -> active(healthy)
				// upgrade service:			{ active(unhealthy) -> active(initializing) }-> active(initializing) -> active(healthy)
				// rollback service:		active(initializing) -> active(healthy)

				switch msg.state {
				case "active":
					switch msg.healthState {
					case "healthy":
						if preState, ok := stkMap[msg.id]; ok {
							switch preState {
							case stk_active_initializing:
								if svcParentIdMap[msg.id] == svc_restarting { // when restart Svc on 1Stk nSvc nIns, Stk want to know having Svc restarting in it or not.
									continue
								}
								if svcParentIdMap[msg.id] == svc_upgrading { // when restart Svc on 1Stk nSvc nIns, Stk want to know having Svc upgrading in it or not.
									continue
								}

								stkSuccess(&msg)
								//case stk_active_degraded:
								//	if svcParentIdMap[msg.id] == svc_restarting { // when restart Svc on 1Stk nSvc nIns, Stk want to know having Svc restarting in it or not.
								//		continue
								//	}
								//
								//	if svcParentIdMap[msg.id] == svc_upgrading { // when restart Svc on 1Stk nSvc nIns, Stk want to know having Svc upgrading in it or not.
								//		continue
								//	}
								//
								//	stkSuccess(&msg)
							}
						}
					case "initializing":
						if preState, ok := stkMap[msg.id]; !ok {
							stkMap[msg.id] = stk_active_initializing
							stkCount(&msg)
						} else {
							switch preState {
							case stk_active_unhealthy:
								delete(stkMap, msg.id)
							}
						}
					case "unhealthy":
						if preState, ok := stkMap[msg.id]; !ok { // try
							stkMap[msg.id] = stk_active_unhealthy
						} else {
							if preState != stk_active_initializing {
								delete(stkMap, msg.id)
							}
						}
						//case "degraded":
						//	if _, ok := stkMap[msg.id]; !ok {
						//		stkMap[msg.id] = stk_active_degraded
						//		stkCount(&msg)
						//	}
					}
				case "error":
					if _, ok := stkMap[msg.id]; ok { // try
						stkFail(&msg)
					}
				case "removed":
					if preState, ok := stkMap[msg.id]; ok {
						switch preState {
						case stk_active_initializing:
							stkFail(&msg)
						}
					}

					delete(stkMap, msg.id)
					stackMap.Delete(msg.id)
				}

			case "service":
				// stack 1 service with 1 container with hc
				// create: 					activating(healthy) -> active(healthy)
				// restart container:		active(initializing) -> active(healthy)
				// stop container:			active(initializing) -> active(healthy)
				// restart service:			restarting(initializing) -> active(healthy)
				// stop service:			active(initializing) -> active(healthy)
				// stop stack:				active(initializing) -> active(healthy)
				// upgrade service:         upgrading(initializing) -> upgraded(healthy)
				// rollback service:		active(initializing) -> active(healthy)

				// stack add 1 service with 2 container with hc
				// restart container: 		active(initializing) -> active(healthy)
				// stop container:			active(initializing) -> active(healthy)
				// restart service:         restarting(degraded) -> restarting(initializing) -> active(healthy)
				// stop service:			active(initializing) -> active(healthy)
				// stop stack:				active(initializing) -> active(healthy)
				// upgrade service:			upgrading(degraded) -> upgraded(healthy)
				// rollback service:		active(initializing) -> active(healthy)

				// stack add 2 service with 2 container with hc
				// restart container:		active(initializing) -> active(healthy)
				// stop container:			active(initializing) -> active(healthy)
				// restart service:			restarting(degraded) -> restarting(initializing) -> active(healthy)
				// stop service:			active(initializing) -> active(healthy)
				// stop stack:				active(initializing) -> active(healthy)
				// upgrade service:			upgrading(degraded) -> upgraded(healthy)
				// rollback service:		active(initializing) -> active(healthy)

				switch msg.state {
				case "activating":
					switch msg.healthState {
					case "healthy":
						if _, ok := svcMap[msg.id]; !ok {
							svcMap[msg.id] = svc_activating_healthy
							svcCount(&msg)
						}
					}
				case "active":
					if svcParentIdMap[msg.parentId] == svc_restarting { // when restart Svc on 1Stk nSvc nIns, Stk want to know having Svc restarting in it or not.
						delete(svcParentIdMap, msg.parentId)
					}

					switch msg.healthState {
					case "healthy":
						if preState, ok := svcMap[msg.id]; ok {
							switch preState {
							case svc_activating_healthy:
								svcSuccess(&msg)
							case svc_active_initializing:
								svcSuccess(&msg)
							case svc_restarting:
								svcSuccess(&msg)
							}
						}
					case "initializing":
						if _, ok := svcMap[msg.id]; !ok {
							svcMap[msg.id] = svc_active_initializing
							svcCount(&msg)
						}
					case "unhealthy":
						if _, ok := svcMap[msg.id]; ok { // try
							svcFail(&msg)
						}
					}
				case "upgraded":
					if svcParentIdMap[msg.parentId] == svc_upgrading { // when restart Svc on 1Stk nSvc nIns, Stk want to know having Svc upgrading in it or not.
						delete(svcParentIdMap, msg.parentId)
					}

					switch msg.healthState {
					case "healthy":
						if preState, ok := svcMap[msg.id]; ok {
							switch preState {
							case svc_upgrading:
								svcSuccess(&msg)
							}
						}
					case "unhealthy":
						if _, ok := svcMap[msg.id]; ok { // try
							svcFail(&msg)
						}
					}
				case "upgrading":
					svcParentIdMap[msg.parentId] = svc_upgrading // when restart Svc on 1Stk nSvc nIns, Stk want to know having Svc upgrading in it or not.

					switch msg.healthState {
					case "initializing":
						if _, ok := svcMap[msg.id]; !ok {
							svcMap[msg.id] = svc_upgrading
							svcCount(&msg)
						}
					case "degraded":
						if _, ok := svcMap[msg.id]; !ok {
							svcMap[msg.id] = svc_upgrading
							svcCount(&msg)
						}
					}
				case "restarting":
					svcParentIdMap[msg.parentId] = svc_restarting // when restart Svc on 1Stk nSvc nIns, Stk want to know having Svc restarting in it or not.

					switch msg.healthState {
					case "initializing":
						if _, ok := svcMap[msg.id]; !ok {
							svcMap[msg.id] = svc_restarting
							svcCount(&msg)
						}
					case "degraded":
						if _, ok := svcMap[msg.id]; !ok {
							svcMap[msg.id] = svc_restarting
							svcCount(&msg)
						}
					}
				case "inactive":
					delete(svcMap, msg.id)
				case "removed":
					if _, ok := svcMap[msg.id]; ok {
						switch msg.healthState {
						case "initializing":
							svcFail(&msg)
						}
					}

					delete(svcMap, msg.id)
				}

			case "instance":
				// stack add 1 service with 1 container with hc
				// create service: 	    starting() -> running(healthy)
				// restart container:	stopping(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// stop container:		stopping(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// restart service:     stopping(healthy) -> stopped(healthy) -> running(reinitializing) -> running(healthy)
				// stop service:        stopping(healthy) -> stopped(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// stop stack:          stopped(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// upgrade service:     old: stopping(healthy) -> stopped(healthy) | new: starting() -> running(initializing) -> running(healthy)
				// rollback service:    new: stopping(healthy) -> stopped(healthy) -> removed(healthy) | old: running(updating-reinitializing) -> running(reinitializing) -> running(healthy)

				// stack add 1 service with 2 container with hc
				// restart container: 	stopping(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// stop container:		stopping(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// restart service:		stopping(healthy) -> stopped(healthy) -> running(reinitializing) -> running(healthy)
				// stop service:		stopping(healthy) -> stopped(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// stop stack:			stopped(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// upgrade service:		old: stopping(healthy) -> stopped(healthy) | new: starting() -> running(initializing) -> running(healthy)
				// rollback service:	new: stopping(healthy) -> stopped(healthy) -> removed(healthy) | old: running(updating-reinitializing) -> running(reinitializing) -> running(healthy)

				// stack add 2 service with 2 container with hc
				// restart container:	stopping(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// stop container:		stopping(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// restart service:		stopping(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// stop service:		stopped(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// stop stack:			stopped(healthy) -> starting(healthy) -> running(reinitializing) -> running(healthy)
				// upgrade service:		old: stopping(healthy) -> stopped(healthy) | new: starting() -> running(initializing) -> running(healthy)
				// rollback service:	new: stopping(healthy) -> stopped(healthy) -> removed(healthy) | old: running(updating-reinitializing) -> running(reinitializing) -> running(healthy)

				switch msg.state {
				case "starting":
					if preState, ok := insMap[msg.id]; !ok {
						insMap[msg.id] = ins_starting
						insCount(&msg)
					} else {
						if len(msg.healthState) != 0 {
							switch preState {
							case ins_stopping:
								insMap[msg.id] = ins_starting
							case ins_stopped:
								insMap[msg.id] = ins_starting
							}
						}
					}
				case "stopping":
					if len(msg.healthState) != 0 {
						switch msg.healthState {
						case "healthy":
							if _, ok := insMap[msg.id]; !ok {
								insMap[msg.id] = ins_stopping
							}
						}
					}
				case "stopped":
					if len(msg.healthState) != 0 {
						switch msg.healthState {
						case "healthy":
							if preState, ok := insMap[msg.id]; !ok {
								insMap[msg.id] = ins_stopped
							} else {
								switch preState {
								case ins_stopping:
									insMap[msg.id] = ins_stopped
								}
							}
						}
					}
				case "running":
					if len(msg.healthState) != 0 {
						switch msg.healthState {
						case "healthy":
							if preState, ok := insMap[msg.id]; ok {
								switch preState {
								case ins_starting:
									insSuccess(&msg)
								case ins_running_reinitializing:
									insSuccess(&msg)
								}
							}
						case "reinitializing":
							if preState, ok := insMap[msg.id]; ok {
								switch preState {
								case ins_stopping:
									insMap[msg.id] = ins_running_reinitializing
									insCount(&msg)
								case ins_stopped:
									insMap[msg.id] = ins_running_reinitializing
									insCount(&msg)
								case ins_starting:
									insMap[msg.id] = ins_running_reinitializing
									insCount(&msg)
								}
							}
						case "updating-reinitializing":
							if _, ok := insMap[msg.id]; !ok {
								insMap[msg.id] = ins_running_reinitializing
								insCount(&msg)
							}
						case "unhealthy":
							if _, ok := insMap[msg.id]; ok { // try
								insFail(&msg)
							}
						}
					}
				case "error":
					insFail(&msg)
				case "removed":
					if _, ok := insMap[msg.id]; ok {
						switch msg.healthState {
						case "initializing":
							insFail(&msg)
						}
					}

					delete(insMap, msg.id)

				}
			}
		}
	}()

}

func newRancherExporter() *rancherExporter {
	initHttpClient()
	initProjectInfo()

	projectLinksSelf := getSubAddress(hc.endpoint, "projects", projectID).String()

	if strings.HasPrefix(projectLinksSelf, "http://") {
		projectLinksSelf = strings.Replace(projectLinksSelf, "http://", "ws://", -1)
	} else {
		projectLinksSelf = strings.Replace(projectLinksSelf, "https://", "wss://", -1)
	}

	wbsFactory := func() *websocket.Conn {
		dialAddress := projectLinksSelf + "/subscribe?eventNames=resource.change&limit=-1&sockId=1"
		httpHeaders := http.Header{}
		httpHeaders.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cattleAccessKey+":"+cattleSecretKey)))
		wbs, _, err := websocket.DefaultDialer.Dial(dialAddress, httpHeaders)
		if err != nil {
			panic(err)
		}

		return wbs
	}

	result := &rancherExporter{
		mutex:         &sync.Mutex{},
		websocketConn: wbsFactory(),

		msgBuff: make(chan buffMsg, 1<<20),

		recreateWebsocket: wbsFactory,
	}

	result.collectingExtending()

	return result
}

func initProjectInfo() {
	// get project self link
	projectsResponseBytes, err := hc.get("projects", nil)
	if err != nil {
		panic(fmt.Errorf("cannot get project info, %v", err))
	}

	projectBytes, _, _, err := jsonparser.Get(projectsResponseBytes, "data", "[0]")
	if err != nil {
		panic(fmt.Errorf("cannot get project, %v", err))
	}

	projectID, err = jsonparser.GetString(projectBytes, "id")
	if err != nil {
		panic(fmt.Errorf("cannot get project id, %v", err))
	}

	projectName, err = jsonparser.GetString(projectBytes, "name")
	if err != nil {
		panic(fmt.Errorf("cannot get project name, %v", err))
	}
}

func getSubAddress(base *url.URL, sub ...string) *url.URL {
	newURL, _ := url.Parse(base.String())

	newURL.Path += "/" + path.Join(sub...)
	return newURL
}

func loadAndInitAggregatedMetrics() (*sync.Map, *sync.Map) {
	// initialization
	gwg := &sync.WaitGroup{}
	stwg := &sync.WaitGroup{}
	swg := &sync.WaitGroup{}
	stackMap := &sync.Map{}
	serviceMap := &sync.Map{}

	if err := hc.foreachCollection(stackSubpath, nil, stwg, func(data []byte) {
		stackID, stackName := setStackAggregatedMetrics(data)
		stackMap.Store(stackID, stackName)
	}); err != nil {
		logger.Warnf("failed to set stack metrics, %v", err)
	}
	stwg.Wait()

	// collect service metrics
	if err := hc.foreachCollection(serviceSubpath, nil, swg, func(data []byte) {
		serviceID, content := setServiceAggregatedMetrics(stackMap, data)
		serviceMap.Store(serviceID, content)
	}); err != nil {
		logger.Warnf("failed to set service metrics, %v", err)
	}
	swg.Wait()

	// collect instance metrics
	if err := hc.foreachCollection(instanceSubpath, nil, gwg, func(data []byte) {
		setInstanceAggregatedMetrics(serviceMap, data)
	}); err != nil {
		logrus.Warnf("failed to set instance metrics, %v", err)
	}
	gwg.Wait()

	return stackMap, serviceMap
}
