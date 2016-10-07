package instance

import (
	"github.com/weaveworks/fluxy"
)

type ServiceConfig struct {
	Automated bool `json:"automation"`
}

type Config struct {
	Services map[flux.ServiceID]ServiceConfig `json:"services"`
}

type InstanceConfig struct {
	ID     flux.InstanceID
	Config Config
}

func MakeConfig() Config {
	return Config{
		Services: map[flux.ServiceID]ServiceConfig{},
	}
}

type UpdateFunc func(config Config) (Config, error)

type DB interface {
	UpdateConfig(instance flux.InstanceID, update UpdateFunc) error
	GetConfig(instance flux.InstanceID) (Config, error)
	All() ([]InstanceConfig, error)
}
