package configs

import (
	"github.com/dnonakolesax/viper"
)

const (
	servicePortKey                = "service.port"
	servicePortDefault            = 8800
	serviceBasePathKey            = "service.base-path"
	serviceBasePathDefault        = "/compiler"
	serviceMetricsPortKey         = "service.metrics-port"
	serviceMetricsPortDefault     = 8801
	serviceMetricsEndpointKey     = "service.metrics-endpoint"
	serviceMetricsEndpointDefault = "/metrics"
)

type ServiceConfig struct {
	Port            int
	BasePath        string
	MetricsPort     int
	MetricsEndpoint string
}

func (sc *ServiceConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault(servicePortKey, servicePortDefault)
	v.SetDefault(serviceBasePathKey, serviceBasePathDefault)
	v.SetDefault(serviceMetricsPortKey, serviceMetricsPortDefault)
	v.SetDefault(serviceMetricsEndpointKey, serviceMetricsEndpointDefault)
}

func (sc *ServiceConfig) Load(v *viper.Viper) {
	sc.Port = v.GetInt(servicePortKey)
	sc.BasePath = v.GetString(serviceBasePathKey)
	sc.MetricsPort = v.GetInt(serviceMetricsPortKey)
	sc.MetricsEndpoint = v.GetString(serviceMetricsEndpointKey)
}
