package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	BotToken string
	DB       DataBaseConfig
}

type DataBaseConfig struct {
	Host     string
	Port     uint16
	Username string
	Name     string
	Password string
	SSLMode  string
}

func LoadConfig(path string) (*Config, error) {

	config := new(Config)

	viper.SetConfigFile(path)

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func MustLoadConfig(path string) *Config {
	config, err := LoadConfig(path)
	if err != nil {
		panic(err)
	}

	return config
}
