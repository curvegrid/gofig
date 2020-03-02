# GoFig

<img src="gofig.png" alt="gofig mascot" width="200"/>

Easy config parsing for server applications.

## Features

- generates flags (command line options) by parsing a structure
- supports optional config file lookup in different path (JSON, TOML and YAML files)
- supports optional config file flag (JSON, TOML and YAML files)
- supports environment variables
- supports user-defined default values

Types supported for flags and environment variables:

|type|env|flag|
|-|-|-|
|string|✔|✔|
|bool|✔|✔|
|int|✔|✔|
|int64|✔|✔|
|uint|✔|✔|
|uint64|✔|✔|
|float64|✔|✔|
|gofig.Duration|✔|✔|

> *Other types except for the list above such as `float32` are not supported.*

> *For the usage of `gofig.Duration`, please refer to [ParseDuration](https://golang.org/pkg/time/#ParseDuration)*

## Order of priority

Each item takes precedence (override) over the item below it:
- flag
- env
- config
- default (user-defined value)

## Struct tags
- json:
  - `json`: custom configuration key name (`-` to disable this json key)
- toml:
  - `toml`: custom configuration key name (`-` to disable this toml key)
- yaml:
  - `yaml`: custom configuration key name (`-` to disable this yaml key)
- env:
  - `env`: custom environment variable name (`-` to disable this env var)
- flag:
  - `flag`: custom flag name (`-` to disable this flag)
  - `desc`: flag description

## Example

```go
package main

import (
	"fmt"
	"time"

	"github.com/curvegrid/gofig"
)

type Config struct {
	Debug       bool           `flag:"d" desc:"enable debugging"`
	Environment string         `json:"env" toml:"env" yaml:"env" env:"env" flag:"e" desc:"environment name"`
	Port        int            `desc:"port to listen on"`
	Timeout     gofig.Duration `desc:"server timeout"`
}

func main() {
	cfg := Config{}
	cfg.Port = 5243 // user-defined default value
	cfg.Timeout = gofig.Duration(30 * time.Second)

	gofig.SetEnvPrefix("GF")
	gofig.SetConfigFileFlag("c", "config file")
	gofig.AddConfigFile("default") // gofig will try to load default.json, default.toml and default.yaml
	gofig.Parse(&cfg)

	fmt.Printf("Config: %+v\n", cfg)
}
```
