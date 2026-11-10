package main

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Full Cobra + Viper configuration layering. The config file stays optional:
// only genuinely broken reads surface as errors.
func initConfigComplete() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homeDir()
		if err != nil {
			return fmt.Errorf("home directory: %w", err)
		}
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".myapp")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("MYAPP")                            // MYAPP_PORT, MYAPP_LOG_LEVEL
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_")) // log-level -> MYAPP_LOG_LEVEL
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config: %w", err)
		}
	}
	return nil
}

type Config struct {
	Port     int    `mapstructure:"port"`
	Host     string `mapstructure:"host"`
	LogLevel string `mapstructure:"log-level"`
	Database struct {
		DSN     string `mapstructure:"dsn"`
		MaxConn int    `mapstructure:"max-conn"`
	} `mapstructure:"database"`
}

func loadConfig() (Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshaling config: %w", err)
	}
	return cfg, nil
}

// Hot reload for long-running processes (servers, daemons).
func watchConfig() {
	viper.OnConfigChange(func(e fsnotify.Event) {
		if cfg, err := loadConfig(); err == nil {
			applyConfig(cfg)
		}
	})
	viper.WatchConfig()
}

func applyConfig(cfg Config) {
	_ = cfg
}

func homeDir() (string, error) {
	return osUserHomeDir()
}