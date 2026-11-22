package config

// TemporalConfig says where the Temporal server is. The task queue is fixed
// (workflows.TaskQueue), not configurable.
type TemporalConfig struct {
	HostPort  string
	Namespace string
}

func loadTemporalConfig() TemporalConfig {
	return TemporalConfig{
		HostPort:  getEnv("TEMPORAL_HOSTPORT", "localhost:7233"),
		Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
	}
}
