package main

import (
	"fmt"

	"github.com/curvegrid/gofig"
)

type Config struct {
	Debug       bool   `flag:"d" desc:"enable debugging"`
	Environment string `json:"env" toml:"env" yaml:"env" env:"env" flag:"e" desc:"environment name"`
	Port        int    `desc:"port to listen on"`
}

func main() {
	cfg := Config{}
	cfg.Port = 5243 // user-defined default value

	gofig.SetEnvPrefix("GF")
	gofig.SetConfigFileFlag("c", "config file")
	gofig.AddConfigFile("default") // gofig will try to load default.json, default.toml and default.yaml
	gofig.Parse(&cfg)

	fmt.Printf("Config: %+v\n", cfg)
}
