package config

type InstanceID string

type ServiceConfig struct {
	Automation bool `json:"automation"`
}

type InstanceConfig struct {
	Services map[flux.ServiceID]ServiceConfig `json:"services"`
}

type ConfigDB interface {
	Set(instance InstanceID, config InstanceConfig) error
	Get(instance InstanceID) (InstanceConfig, error)
}
