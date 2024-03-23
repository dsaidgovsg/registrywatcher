package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Environment string

const (
	CI          Environment = "ci"
	Testing     Environment = "test"
	Development Environment = "dev"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

func GetEnv() Environment {
	env := os.Getenv("ENV")
	switch env {
	case "ci":
		return CI
	case "test":
		return Testing
	case "dev":
		return Development
	case "staging":
		return Staging
	case "production":
		return Production
	case "":
		panic("Environment not set")
	default:
		panic(fmt.Sprintf("Invalid environment: %s", env))
	}
}

type Config struct {
	DSN string
}

func InitConfig(env Environment) *Config {
	var c Config

	viper.SetConfigName(string(env))
	viper.SetConfigType("yaml")
	viper.AddConfigPath(fmt.Sprintf("./env/%s", env))
	viper.AddConfigPath(fmt.Sprintf("../../env/%s", env))

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to read config file: %v\n", err))
	}

	// Set default values or handle missing keys here

	// Bind configuration values to struct fields here
	err = viper.Unmarshal(&c)
	if err != nil {
		panic(fmt.Sprintf("unable to decode into struct, %v", err))
	}

	// Example: viper.Get("key") to retrieve configuration value
	return &c
}
