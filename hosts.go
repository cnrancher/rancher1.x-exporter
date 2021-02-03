package main

import (
	"github.com/buger/jsonparser"
)

const (
	hostSubpath = "hosts"
)

func setHostMetrics(hostBytes []byte) {
	hostName, _ := jsonparser.GetString(hostBytes, "name")
	hostState, _ := jsonparser.GetString(hostBytes, "state")
	hostId, _ := jsonparser.GetString(hostBytes, "id")
	hostAgentState, _ := jsonparser.GetString(hostBytes, "agentState")

	if len(hostName) == 0 {
		hostName, _ = jsonparser.GetString(hostBytes, "hostname")
	}

	for _, y := range hostStates {
		if hostState == y {
			infinityWorksHostsState.WithLabelValues(hostId, hostName, y).Set(1)
		} else {
			infinityWorksHostsState.WithLabelValues(hostId, hostName, y).Set(0)
		}
	}

	for _, y := range agentStates {
		if hostAgentState == y {
			infinityWorksHostAgentsState.WithLabelValues(hostId, hostName, y).Set(1)
		} else {
			infinityWorksHostAgentsState.WithLabelValues(hostId, hostName, y).Set(0)
		}
	}
}
