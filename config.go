package orgmanager

import (
	"fmt"

	"github.com/spf13/viper"
)

type configs struct {
	Targets map[string]Config
}

var Targets = make(map[string]Target)

func init() {
	viper.SetConfigType("yml")
	viper.SetConfigName("org-manager")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
	conf := new(configs)
	if err := viper.Unmarshal(&conf); err != nil {
		panic(fmt.Errorf("Fatal error unmarshal config file: %w \n", err))
	}
	for name := range conf.Targets {
		target, err := InitTarget(fmt.Sprintf("targets.%s", name))
		if err != nil {
			panic(err)
		}
		Targets[name] = target
	}
}
