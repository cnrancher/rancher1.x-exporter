package main

import (
	"sync"

	"github.com/buger/jsonparser"
	"github.com/sirupsen/logrus"
)

const (
	instanceSubpath = "instances"
)

func setInstanceMetrics(services *sync.Map, instanceBytes []byte) {
	instanceName, _ := jsonparser.GetString(instanceBytes, "name")
	instanceId, _ := jsonparser.GetString(instanceBytes, "id")
	instanceSystem, _ := jsonparser.GetUnsafeString(instanceBytes, "system")
	instanceType, _ := jsonparser.GetString(instanceBytes, "type")

	var stackName, serviceName string
	labels := []string{projectName}

	serviceId, err := jsonparser.GetString(instanceBytes, "serviceIds", "[0]")
	if err != nil {
		logrus.Warnf("failed to get service from container instance %s", instanceId)
	} else {
		value, ok := services.Load(serviceId)
		if ok {
			content := value.(*serviceContent)
			stackName = content.StackName
			serviceName = content.ServiceName
		}
	}

	labels = append(labels, stackName, serviceName, instanceName, instanceSystem, instanceType)
	extendingInstanceHeartbeat.WithLabelValues(labels...).Set(float64(1))

	if instanceFirstRunningTS, _ := jsonparser.GetInt(instanceBytes, "firstRunningTS"); instanceFirstRunningTS != 0 {
		instanceCreatedTS, _ := jsonparser.GetInt(instanceBytes, "createdTS")
		extendingInstanceBootstrapMsCost.WithLabelValues(labels...).Set(float64(instanceFirstRunningTS - instanceCreatedTS))
	}
}

func setInstanceAggregatedMetrics(services *sync.Map, instanceBytes []byte) {
	instanceName, _ := jsonparser.GetString(instanceBytes, "name")
	instanceId, _ := jsonparser.GetString(instanceBytes, "id")
	instanceSystem, _ := jsonparser.GetUnsafeString(instanceBytes, "system")
	instanceType, _ := jsonparser.GetString(instanceBytes, "type")
	instanceState, _ := jsonparser.GetString(instanceBytes, "state")
	instanceFirstRunningTS, _ := jsonparser.GetInt(instanceBytes, "firstRunningTS")
	instanceCreatedTS, _ := jsonparser.GetInt(instanceBytes, "createdTS")

	var stackName, serviceName string

	labels := []string{projectName}

	serviceId, err := jsonparser.GetString(instanceBytes, "serviceIds", "[0]")
	if err != nil {
		logrus.Warnf("failed to get service from container instance %s", instanceId)
	} else {
		value, ok := services.Load(serviceId)
		if ok {
			content := value.(*serviceContent)
			stackName = content.StackName
			serviceName = content.ServiceName
		}
	}

	labels = append(labels, stackName, serviceName, instanceName, instanceSystem, instanceType)

	extendingTotalInstanceBootstraps.WithLabelValues(projectName, specialTag, specialTag, specialTag)
	extendingTotalInstanceBootstraps.WithLabelValues(projectName, stackName, specialTag, specialTag)
	extendingTotalInstanceBootstraps.WithLabelValues(projectName, stackName, serviceName, specialTag)
	extendingTotalInstanceBootstraps.WithLabelValues(projectName, stackName, serviceName, instanceName)
	extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, specialTag, specialTag, specialTag)
	extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, stackName, specialTag, specialTag)
	extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, stackName, serviceName, specialTag)
	extendingTotalSuccessInstanceBootstrap.WithLabelValues(projectName, stackName, serviceName, instanceName)
	extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, specialTag, specialTag, specialTag)
	extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, stackName, specialTag, specialTag)
	extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, stackName, serviceName, specialTag)
	extendingTotalErrorInstanceBootstrap.WithLabelValues(projectName, stackName, serviceName, instanceName)

	switch instanceState {
	case "stopped":
		fallthrough
	case "running":
		extendingTotalInstanceInitializations.WithLabelValues(projectName, specialTag, specialTag, specialTag).Inc()
		extendingTotalInstanceInitializations.WithLabelValues(projectName, stackName, specialTag, specialTag).Inc()
		extendingTotalInstanceInitializations.WithLabelValues(projectName, stackName, serviceName, specialTag).Inc()
		extendingTotalInstanceInitializations.WithLabelValues(projectName, stackName, serviceName, instanceName).Inc()
		extendingTotalSuccessInstanceInitialization.WithLabelValues(projectName, specialTag, specialTag, specialTag).Inc()
		extendingTotalSuccessInstanceInitialization.WithLabelValues(projectName, stackName, specialTag, specialTag).Inc()
		extendingTotalSuccessInstanceInitialization.WithLabelValues(projectName, stackName, serviceName, specialTag).Inc()
		extendingTotalSuccessInstanceInitialization.WithLabelValues(projectName, stackName, serviceName, instanceName).Inc()
		extendingTotalErrorInstanceInitialization.WithLabelValues(projectName, specialTag, specialTag, specialTag)
		extendingTotalErrorInstanceInitialization.WithLabelValues(projectName, stackName, specialTag, specialTag)
		extendingTotalErrorInstanceInitialization.WithLabelValues(projectName, stackName, serviceName, specialTag)
		extendingTotalErrorInstanceInitialization.WithLabelValues(projectName, stackName, serviceName, instanceName)

		if instanceFirstRunningTS != 0 {
			instanceStartupTime := instanceFirstRunningTS - instanceCreatedTS
			extendingInstanceBootstrapMsCost.WithLabelValues(labels...).Set(float64(instanceStartupTime))
		}
	}
}
