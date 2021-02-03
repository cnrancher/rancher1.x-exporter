package main

import (
	"fmt"
	"sync"

	"github.com/buger/jsonparser"
)

type serviceContent struct {
	StackID     string
	StackName   string
	ServiceID   string
	ServiceName string
}

const (
	serviceSubpath = "services"
)

func setServiceMetrics(stacks *sync.Map, serviceBytes []byte) (string, *serviceContent) {
	stackId, _ := jsonparser.GetString(serviceBytes, "stackId")
	IstackName, _ := stacks.Load(stackId)
	stackName := fmt.Sprintf("%v", IstackName)

	serviceId, _ := jsonparser.GetString(serviceBytes, "id")
	serviceName, _ := jsonparser.GetString(serviceBytes, "name")
	serviceSystem, _ := jsonparser.GetUnsafeString(serviceBytes, "system")
	serviceType, _ := jsonparser.GetString(serviceBytes, "type")
	serviceHealthState, _ := jsonparser.GetString(serviceBytes, "healthState")
	serviceState, _ := jsonparser.GetString(serviceBytes, "state")
	serviceScale, _ := jsonparser.GetInt(serviceBytes, "scale")

	infinityWorksServicesScale.WithLabelValues(serviceName, stackName, serviceSystem).Set(float64(serviceScale))
	for _, y := range healthStates {
		if serviceHealthState == y {
			infinityWorksServicesHealth.WithLabelValues(serviceId, stackId, serviceName, stackName, y, serviceSystem).Set(1)
		} else {
			infinityWorksServicesHealth.WithLabelValues(serviceId, stackId, serviceName, stackName, y, serviceSystem).Set(0)
		}
	}

	for _, y := range serviceStates {
		if serviceState == y {
			infinityWorksServicesState.WithLabelValues(serviceId, stackId, serviceName, stackName, y, serviceSystem).Set(1)
		} else {
			infinityWorksServicesState.WithLabelValues(serviceId, stackId, serviceName, stackName, y, serviceSystem).Set(0)
		}
	}

	extendingServiceHeartbeat.WithLabelValues(projectName, stackName, serviceName, serviceSystem, serviceType).Set(float64(1))
	return serviceId, &serviceContent{
		ServiceID:   serviceId,
		ServiceName: serviceName,
		StackID:     stackId,
		StackName:   stackName,
	}
}

func setServiceAggregatedMetrics(stacks *sync.Map, serviceBytes []byte) (string, *serviceContent) {
	stackId, _ := jsonparser.GetString(serviceBytes, "stackId")
	IstackName, _ := stacks.Load(stackId)
	stackName := fmt.Sprintf("%v", IstackName)

	serviceId, _ := jsonparser.GetString(serviceBytes, "id")
	serviceName, _ := jsonparser.GetString(serviceBytes, "name")
	serviceHealthState, _ := jsonparser.GetString(serviceBytes, "healthState")
	serviceState, _ := jsonparser.GetString(serviceBytes, "state")

	extendingTotalServiceBootstraps.WithLabelValues(projectName, specialTag, specialTag)
	extendingTotalServiceBootstraps.WithLabelValues(projectName, stackName, specialTag)
	extendingTotalServiceBootstraps.WithLabelValues(projectName, stackName, serviceName)
	extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, specialTag, specialTag)
	extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, stackName, specialTag)
	extendingTotalSuccessServiceBootstrap.WithLabelValues(projectName, stackName, serviceName)
	extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, specialTag, specialTag)
	extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, stackName, specialTag)
	extendingTotalErrorServiceBootstrap.WithLabelValues(projectName, stackName, serviceName)

	switch serviceState {
	case "active":
		extendingTotalServiceInitializations.WithLabelValues(projectName, specialTag, specialTag).Inc()
		extendingTotalServiceInitializations.WithLabelValues(projectName, stackName, specialTag).Inc()
		extendingTotalServiceInitializations.WithLabelValues(projectName, stackName, serviceName).Inc()

		if serviceHealthState == "unhealthy" {
			extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, specialTag, specialTag)
			extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, stackName, specialTag)
			extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, stackName, serviceName)
			extendingTotalErrorServiceInitialization.WithLabelValues(projectName, specialTag, specialTag).Inc()
			extendingTotalErrorServiceInitialization.WithLabelValues(projectName, stackName, specialTag).Inc()
			extendingTotalErrorServiceInitialization.WithLabelValues(projectName, stackName, serviceName).Inc()
		} else if serviceHealthState == "healthy" {
			extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, specialTag, specialTag).Inc()
			extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, stackName, specialTag).Inc()
			extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, stackName, serviceName).Inc()
			extendingTotalErrorServiceInitialization.WithLabelValues(projectName, specialTag, specialTag)
			extendingTotalErrorServiceInitialization.WithLabelValues(projectName, stackName, specialTag)
			extendingTotalErrorServiceInitialization.WithLabelValues(projectName, stackName, serviceName)
		}
	case "error":
		extendingTotalServiceInitializations.WithLabelValues(projectName, specialTag, specialTag).Inc()
		extendingTotalServiceInitializations.WithLabelValues(projectName, stackName, specialTag).Inc()
		extendingTotalServiceInitializations.WithLabelValues(projectName, stackName, serviceName).Inc()
		extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, specialTag, specialTag)
		extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, stackName, specialTag)
		extendingTotalSuccessServiceInitialization.WithLabelValues(projectName, stackName, serviceName)
		extendingTotalErrorServiceInitialization.WithLabelValues(projectName, specialTag, specialTag).Inc()
		extendingTotalErrorServiceInitialization.WithLabelValues(projectName, stackName, specialTag).Inc()
		extendingTotalErrorServiceInitialization.WithLabelValues(projectName, stackName, serviceName).Inc()
	}
	return serviceId, &serviceContent{
		ServiceID:   serviceId,
		ServiceName: serviceName,
		StackID:     stackId,
		StackName:   stackName,
	}
}
