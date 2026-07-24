package redis

import (
	e "github.com/ChatDetectiveORG/shared/errors"
	"time"

	"github.com/spf13/viper"
)

const PodType = "commands"

type Config struct {
	RedisConfig    *RedisConfig
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	Database int

	MaxIdle     int
	MaxActive   int
	IdleTimeout time.Duration
	Wait        bool

	ConnectionTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
}

// Fetches config from environment variables
func FetchConfig() (*Config, *e.ErrorInfo) {
	viper.AutomaticEnv()

	config := &Config{
		RedisConfig: &RedisConfig{
			Host:              viper.GetString("REDIS_HOST"),
			Port:              viper.GetString("REDIS_PORT"),
			Password:          viper.GetString("REDIS_PASSWORD"),
			Database:          viper.GetInt("REDIS_DB"),
			MaxIdle:           viper.GetInt("REDIS_MAX_IDLE"),
			MaxActive:         viper.GetInt("REDIS_MAX_ACTIVE"),
			IdleTimeout:       viper.GetDuration("REDIS_IDLE_TIMEOUT"),
			Wait:              viper.GetBool("REDIS_WAIT"),
			ConnectionTimeout: viper.GetDuration("REDIS_CONNECTION_TIMEOUT"),
			ReadTimeout:       viper.GetDuration("REDIS_READ_TIMEOUT"),
			WriteTimeout:      viper.GetDuration("REDIS_WRITE_TIMEOUT"),
		},
	}

	return config, e.Nil()
}
