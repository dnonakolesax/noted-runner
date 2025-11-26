package configs

import (
	"time"

	"github.com/dnonakolesax/viper"
)

const (
	volumeSourceKey     = "docker.volume.source"
	volumeSourceDefault = "notedcode"
	volumeTargetKey     = "docker.volume.target"
	volumeTargetDefault = "/noted/codes"
)

type VolumeConfig struct {
	Source string
	Target string
}

func (vc *VolumeConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault(volumeSourceKey, volumeSourceDefault)
	v.SetDefault(volumeTargetKey, volumeTargetDefault)
}

func (vc *VolumeConfig) Load(v *viper.Viper) {
	vc.Source = v.GetString(volumeSourceKey)
	vc.Target = v.GetString(volumeTargetKey)
}

const (
	envRMQAddrKey          = "docker.env.rmq_addr"
	enrRMQAddrDefault      = "amqp://guest:guest@rabbit:5672/"
	envMountPathKey        = "docker.env.mount_path"
	envMountPathDefault    = "/noted/codes/kernels"
	envExportPrefixKey     = "docker.env.export_prefix"
	envExportPrefixDefault = "Export_block_"
	envBlockPrefixKey      = "docker.env.block_prefix"
	envBlockPrefixDefault  = "block_"
	envChanNameKey         = "docker.env.chan_name"
	envChanNameDefault     = "noted-kernels"
	envBlockTimeoutKey     = "docker.env.block_timeout"
	envBlockTimeoutDefault = 30 * time.Second
)

type EnvConfig struct {
	RMQAddr      string
	MountPath    string
	ExportPrefix string
	BlockPrefix  string
	ChanName     string
	BlockTimeout time.Duration
}

func (ec *EnvConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault(envRMQAddrKey, enrRMQAddrDefault)
	v.SetDefault(envMountPathKey, envMountPathDefault)
	v.SetDefault(envExportPrefixKey, envExportPrefixDefault)
	v.SetDefault(envBlockPrefixKey, envBlockPrefixDefault)
	v.SetDefault(envChanNameKey, envChanNameDefault)
	v.SetDefault(envBlockTimeoutKey, envBlockTimeoutDefault)
}

func (ec *EnvConfig) Load(v *viper.Viper) {
	ec.RMQAddr = v.GetString(envRMQAddrKey)
	ec.MountPath = v.GetString(envMountPathKey)
	ec.ExportPrefix = v.GetString(envExportPrefixKey)
	ec.BlockPrefix = v.GetString(envBlockPrefixKey)
	ec.ChanName = v.GetString(envChanNameKey)
	ec.BlockTimeout = v.GetDuration(envBlockTimeoutKey)
}

const (
	dockerHostKey        = "docker.host"
	dockerHostDefault    = "unix:///var/run/docker.sock"
	dockerImageKey       = "docker.image"
	dockerImageDefault   = "dnonakolesax/noted-kernel:0.0.2"
	dockerNetworkKey     = "docker.network"
	dockerNetworkDefault = "noted-rmq-runners"
	dockerPrefixKey      = "docker.prefix"
	dockerPrefixDefault  = "noted-kernel_"
	dockerAppPortKey     = "docker.app-port"
	dockerAppPortDefault = "8080"
)

type DockerConfig struct {
	Volume  VolumeConfig
	Env     EnvConfig
	Host    string
	Image   string
	Network string
	Prefix  string
	AppPort string
}

func (dc *DockerConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault(dockerHostKey, dockerHostDefault)
	v.SetDefault(dockerImageKey, dockerImageDefault)
	v.SetDefault(dockerNetworkKey, dockerNetworkDefault)
	v.SetDefault(dockerPrefixKey, dockerPrefixDefault)
	v.SetDefault(dockerAppPortKey, dockerAppPortDefault)
	dc.Volume.SetDefaults(v)
	dc.Env.SetDefaults(v)
}

func (dc *DockerConfig) Load(v *viper.Viper) {
	dc.Host = v.GetString(dockerHostKey)
	dc.Image = v.GetString(dockerImageKey)
	dc.Network = v.GetString(dockerNetworkKey)
	dc.Prefix = v.GetString(dockerPrefixKey)
	dc.AppPort = v.GetString(dockerAppPortKey)
	dc.Volume.Load(v)
	dc.Env.Load(v)
}
