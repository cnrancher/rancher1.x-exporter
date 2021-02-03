package main

import (
	"github.com/buger/jsonparser"
)

const (
	stackSubpath = "stacks"
)

func setStackMetrics(stackBytes []byte) (string, string) {
	stackId, _ := jsonparser.GetString(stackBytes, "id")
	stackName, _ := jsonparser.GetString(stackBytes, "name")
	stackSystem, _ := jsonparser.GetUnsafeString(stackBytes, "system")
	stackType, _ := jsonparser.GetString(stackBytes, "type")
	stackHealthState, _ := jsonparser.GetString(stackBytes, "healthState")
	stackState, _ := jsonparser.GetString(stackBytes, "state")
	for _, y := range healthStates {
		if stackHealthState == y {
			infinityWorksStacksHealth.WithLabelValues(stackId, stackName, y, stackSystem).Set(1)
		} else {
			infinityWorksStacksHealth.WithLabelValues(stackId, stackName, y, stackSystem).Set(0)
		}
	}

	for _, y := range stackStates {
		if stackState == y {
			infinityWorksStacksState.WithLabelValues(stackId, stackName, y, stackSystem).Set(1)
		} else {
			infinityWorksStacksState.WithLabelValues(stackId, stackName, y, stackSystem).Set(0)
		}
	}
	extendingStackHeartbeat.WithLabelValues(projectName, stackName, stackSystem, stackType).Set(float64(1))
	return stackId, stackName
}

func setStackAggregatedMetrics(stackBytes []byte) (string, string) {
	stackId, _ := jsonparser.GetString(stackBytes, "id")
	stackName, _ := jsonparser.GetString(stackBytes, "name")
	stackHealthState, _ := jsonparser.GetString(stackBytes, "healthState")
	stackState, _ := jsonparser.GetString(stackBytes, "state")

	// init bootstrap
	extendingTotalStackBootstraps.WithLabelValues(projectName, specialTag)
	extendingTotalStackBootstraps.WithLabelValues(projectName, stackName)
	extendingTotalSuccessStackBootstrap.WithLabelValues(projectName, specialTag)
	extendingTotalSuccessStackBootstrap.WithLabelValues(projectName, stackName)
	extendingTotalErrorStackBootstrap.WithLabelValues(projectName, specialTag)
	extendingTotalErrorStackBootstrap.WithLabelValues(projectName, stackName)

	switch stackState {
	case "active":
		if stackHealthState == "unhealthy" {
			extendingTotalStackInitializations.WithLabelValues(projectName, specialTag).Inc()
			extendingTotalStackInitializations.WithLabelValues(projectName, stackName).Inc()
			extendingTotalSuccessStackInitialization.WithLabelValues(projectName, specialTag)
			extendingTotalSuccessStackInitialization.WithLabelValues(projectName, stackName)
			extendingTotalErrorStackInitialization.WithLabelValues(projectName, specialTag).Inc()
			extendingTotalErrorStackInitialization.WithLabelValues(projectName, stackName).Inc()
		} else if stackHealthState == "healthy" {
			extendingTotalStackInitializations.WithLabelValues(projectName, specialTag).Inc()
			extendingTotalStackInitializations.WithLabelValues(projectName, stackName).Inc()
			extendingTotalSuccessStackInitialization.WithLabelValues(projectName, specialTag).Inc()
			extendingTotalSuccessStackInitialization.WithLabelValues(projectName, stackName).Inc()
			extendingTotalErrorStackInitialization.WithLabelValues(projectName, specialTag)
			extendingTotalErrorStackInitialization.WithLabelValues(projectName, stackName)
		}
	case "error":
		extendingTotalStackInitializations.WithLabelValues(projectName, specialTag).Inc()
		extendingTotalStackInitializations.WithLabelValues(projectName, stackName).Inc()
		extendingTotalSuccessStackInitialization.WithLabelValues(projectName, specialTag)
		extendingTotalSuccessStackInitialization.WithLabelValues(projectName, stackName)
		extendingTotalErrorStackInitialization.WithLabelValues(projectName, specialTag).Inc()
		extendingTotalErrorStackInitialization.WithLabelValues(projectName, stackName).Inc()
	}

	return stackId, stackName
}
