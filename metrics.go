package main

import "github.com/prometheus/client_golang/prometheus"

var (
	// health & state of host, stack, service
	infinityWorksHostsState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "host_state",
			Help:      "State of defined host as reported by the Rancher API",
		}, []string{"id", "name", "state"})

	infinityWorksHostAgentsState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "host_agent_state",
			Help:      "State of defined host agent as reported by the Rancher API",
		}, []string{"id", "name", "state"})

	infinityWorksStacksHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "stack_health_status",
			Help:      "HealthState of defined stack as reported by Rancher",
		}, []string{"id", "name", "health_state", "system"})

	infinityWorksStacksState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "stack_state",
			Help:      "State of defined stack as reported by Rancher",
		}, []string{"id", "name", "state", "system"})

	infinityWorksServicesScale = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "service_scale",
			Help:      "scale of defined service as reported by Rancher",
		}, []string{"name", "stack_name", "system"})

	infinityWorksServicesHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "service_health_status",
			Help:      "HealthState of the service, as reported by the Rancher API",
		}, []string{"id", "stack_id", "name", "stack_name", "health_state", "system"})

	infinityWorksServicesState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "service_state",
			Help:      "State of the service, as reported by the Rancher API",
		}, []string{"id", "stack_id", "name", "stack_name", "state", "system"})

	/**
	Extended
	*/

	// total counter of stack, service, instance

	extendingTotalStackInitializations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "stacks_initialization_total",
		Help:      "Current total number of the initialization stacks in Rancher",
	}, []string{"environment_name", "name"})

	extendingTotalSuccessStackInitialization = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "stacks_initialization_success_total",
		Help:      "Current total number of the healthy and active initialization stacks in Rancher",
	}, []string{"environment_name", "name"})

	extendingTotalErrorStackInitialization = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "stacks_initialization_error_total",
		Help:      "Current total number of the unhealthy or error initialization stacks in Rancher",
	}, []string{"environment_name", "name"})

	extendingTotalServiceInitializations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "services_initialization_total",
		Help:      "Current total number of the initialization services in Rancher",
	}, []string{"environment_name", "stack_name", "name"})

	extendingTotalSuccessServiceInitialization = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "services_initialization_success_total",
		Help:      "Current total number of the healthy and active initialization services in Rancher",
	}, []string{"environment_name", "stack_name", "name"})

	extendingTotalErrorServiceInitialization = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "services_initialization_error_total",
		Help:      "Current total number of the unhealthy or error initialization services in Rancher",
	}, []string{"environment_name", "stack_name", "name"})

	extendingTotalInstanceInitializations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "instances_initialization_total",
		Help:      "Current total number of the initialization instances in Rancher",
	}, []string{"environment_name", "stack_name", "service_name", "name"})

	extendingTotalSuccessInstanceInitialization = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "instances_initialization_success_total",
		Help:      "Current total number of the healthy and active initialization instances in Rancher",
	}, []string{"environment_name", "stack_name", "service_name", "name"})

	extendingTotalErrorInstanceInitialization = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "instances_initialization_error_total",
		Help:      "Current total number of the unhealthy or error initialization instances in Rancher",
	}, []string{"environment_name", "stack_name", "service_name", "name"})

	extendingTotalStackBootstraps = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "stacks_bootstrap_total",
		Help:      "Current total number of the bootstrap stacks in Rancher",
	}, []string{"environment_name", "name"})

	extendingTotalSuccessStackBootstrap = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "stacks_bootstrap_success_total",
		Help:      "Current total number of the healthy and active bootstrap stacks in Rancher",
	}, []string{"environment_name", "name"})

	extendingTotalErrorStackBootstrap = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "stacks_bootstrap_error_total",
		Help:      "Current total number of the unhealthy or error bootstrap stacks in Rancher",
	}, []string{"environment_name", "name"})

	extendingTotalServiceBootstraps = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "services_bootstrap_total",
		Help:      "Current total number of the bootstrap services in Rancher",
	}, []string{"environment_name", "stack_name", "name"})

	extendingTotalSuccessServiceBootstrap = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "services_bootstrap_success_total",
		Help:      "Current total number of the healthy and active bootstrap services in Rancher",
	}, []string{"environment_name", "stack_name", "name"})

	extendingTotalErrorServiceBootstrap = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "services_bootstrap_error_total",
		Help:      "Current total number of the unhealthy or error bootstrap services in Rancher",
	}, []string{"environment_name", "stack_name", "name"})

	extendingTotalInstanceBootstraps = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "instances_bootstrap_total",
		Help:      "Current total number of the bootstrap instances in Rancher",
	}, []string{"environment_name", "stack_name", "service_name", "name"})

	extendingTotalSuccessInstanceBootstrap = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "instances_bootstrap_success_total",
		Help:      "Current total number of the healthy and active bootstrap instances in Rancher",
	}, []string{"environment_name", "stack_name", "service_name", "name"})

	extendingTotalErrorInstanceBootstrap = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "instances_bootstrap_error_total",
		Help:      "Current total number of the unhealthy or error bootstrap instances in Rancher",
	}, []string{"environment_name", "stack_name", "service_name", "name"})

	// startup gauge
	extendingInstanceBootstrapMsCost = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "instance_bootstrap_ms",
		Help:      "The bootstrap milliseconds of instances in Rancher",
	}, []string{"environment_name", "stack_name", "service_name", "name", "system", "type"})

	// heartbeat
	extendingStackHeartbeat = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "stack_heartbeat",
		Help:      "The heartbeat of stacks in Rancher",
	}, []string{"environment_name", "name", "system", "type"})

	extendingServiceHeartbeat = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "service_heartbeat",
		Help:      "The heartbeat of services in Rancher",
	}, []string{"environment_name", "stack_name", "name", "system", "type"})

	extendingInstanceHeartbeat = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "instance_heartbeat",
		Help:      "The heartbeat of instances in Rancher",
	}, []string{"environment_name", "stack_name", "service_name", "name", "system", "type"})
)
