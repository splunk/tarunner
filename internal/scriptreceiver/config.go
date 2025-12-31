package scriptreceiver

import "github.com/splunk/tarunner/internal/conf"

type Config struct {
	BaseDir   string          `mapstructure:"base_dir"`
	Input     conf.Input      `mapstructure:"input"`
	Transform *conf.Transform `mapstructure:"transform"`
}
