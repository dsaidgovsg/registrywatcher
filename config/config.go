package config

import (
	"fmt"
	"github.com/spf13/viper"
)

func SetUpConfig(configFileName string) *viper.Viper {
	conf := viper.New()
	conf.SetConfigName(configFileName)
	conf.SetConfigType("toml")
	conf.AddConfigPath("./config")
	conf.AddConfigPath("../config")
	conf.AutomaticEnv()
	err := conf.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("reading config file failed: %v", err))
	}

	return conf
}
