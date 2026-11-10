# Viper Config Sources and File Formats

Viper detects format from file extension. Supported extensions: YAML (`.yaml`, `.yml`), TOML (`.toml`), JSON (`.json`), HCL (`.hcl`), INI (`.ini`), Properties (`.properties`), dotenv (`.env`).

Force a format when there is no extension.

```go
viper.SetConfigType("yaml")
viper.SetConfigFile("/etc/myapp/config")
```

## Config file search

```go
viper.SetConfigName("config")
viper.SetConfigType("yaml")
viper.AddConfigPath("$HOME/.myapp")
viper.AddConfigPath("/etc/myapp/")
viper.AddConfigPath(".")

if err := viper.ReadInConfig(); err != nil {
    var notFound *viper.ConfigFileNotFoundError
    if !errors.As(err, &notFound) {
        return err
    }
}

fmt.Println("Using config:", viper.ConfigFileUsed())
```

Viper searches paths in order, stops at the first match.

## Merging multiple config files

`MergeInConfig` merges a second config file into the current state. Later values override earlier ones for the same key.

```go
viper.SetConfigFile("base.yaml")
viper.ReadInConfig()

viper.SetConfigFile("override.yaml")
viper.MergeInConfig()
```

Pattern: ship a base config with the binary, let users drop an override in `~/.myapp/override.yaml`.

## Multiple config files via SetConfigFile

For environment-based config loading.

```go
env := os.Getenv("APP_ENV")
if env == "" {
    env = "development"
}
viper.SetConfigFile(fmt.Sprintf("config.%s.yaml", env))
viper.ReadInConfig()
```

## Remote KV stores (etcd, Consul)

Viper supports remote KV stores via the `viper/remote` sub-package. This keeps remote config behind an opt-in import.

```go
import _ "github.com/spf13/viper/remote"

viper.AddRemoteProvider("etcd3", "http://127.0.0.1:2379", "/config/myapp.yaml")
viper.SetConfigType("yaml")
viper.ReadRemoteConfig()

viper.AddRemoteProvider("consul", "localhost:8500", "myapp/config")
viper.SetConfigType("json")
viper.ReadRemoteConfig()
```

Caution: remote config adds network latency to startup and a runtime dependency. Use it only when you need centralized config across many service instances. For most applications, files + env vars are sufficient.

Watch for remote changes.

```go
go func() {
    for {
        time.Sleep(5 * time.Second)
        viper.WatchRemoteConfig()
    }
}()
```

## Embedding config with go:embed

Load config from embedded assets (for self-contained binaries).

```go
//go:embed config.yaml
var defaultConfig []byte

func init() {
    viper.SetConfigType("yaml")
    viper.ReadConfig(bytes.NewReader(defaultConfig))
}
```

`ReadConfig` accepts any `io.Reader`. This is useful for shipping default config inside the binary, then layering user overrides on top via `MergeInConfig`.
