# Config with Cobra + Viper

Complete Cobra+Viper wiring — flag binding, precedence rules, configuration layering — lives in `fabianoflorentino/golang-agent-skills@golang-cli`. This reference covers where config files sit in the layout and the loader shape that follows.

## Where config lives

```
myapp/
├── cmd/myapp/
│   ├── main.go                # entry point
│   ├── root.go                # root command + Viper init
│   ├── serve.go               # subcommand with flags
│   └── config.go              # config struct + loader
└── configs/
    └── config.yaml            # default config file
```

## The config struct

Define configuration as a struct with `mapstructure` tags matching the YAML keys, then unmarshal:

```go
package main

import (
    "fmt"

    "github.com/spf13/viper"
)

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
```

**Sources matter more than shape:**

- Load config from environment variables, files, or flags — never hardcode it.
- Sensitive values (credentials, DSNs) come from env vars or a secret manager, never from config files.
