package manager

import (
	"fmt"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type configs struct {
	Targets map[string]Config
}

var Targets = make(map[string]Target)

type TargetConfigStore interface {
	GetConfigs() []TargetConfig
}

type Unmarshaler func(any) error

type TargetConfig interface {
	GetPlatform() string
	GetUnmarshaler() Unmarshaler
}

type DefaultViperConfigStore struct{}

func (DefaultViperConfigStore) GetConfigs() (configs []TargetConfig) {
	viper.SetConfigType("yml")
	viper.SetConfigName("org-manager")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
	targets := viper.GetStringMap("targets")
	for name := range targets {
		configs = append(configs, &DefaultViperConfig{targetName: name})
	}
	return configs
}

type DefaultViperConfig struct {
	targetName string
}

func (c DefaultViperConfig) GetPlatform() string {
	return viper.GetString(fmt.Sprintf("targets.%s.platform", c.targetName))
}

func (c DefaultViperConfig) GetUnmarshaler() Unmarshaler {
	configKey := fmt.Sprintf("targets.%s", c.targetName)
	return func(rawVal any) error {
		return viper.UnmarshalKey(configKey, rawVal)
	}
}

type DatabaseConfigStore struct {
	db *gorm.DB
}

func init() {
	InitWithTargetConfigStore(&DefaultViperConfigStore{})
}

func InitWithTargetConfigStore(store TargetConfigStore) {
	for _, config := range store.GetConfigs() {
		target, err := InitTarget(config.GetPlatform(), config.GetUnmarshaler())
		if err != nil {
			panic(err)
		}
		Targets[TargetKey(target)] = target
	}
}
