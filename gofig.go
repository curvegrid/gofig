// Copyright (c) 2019 Curvegrid Inc.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gofig

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	yaml "gopkg.in/yaml.v2"
)

// Duration wraps time.Duration so we can augment it with encoding.TextMarshaler and
// flag.Value interfaces.
type Duration time.Duration

// UnmarshalText unmarshals a byte slice into a Duration value.
func (d *Duration) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	*d = Duration(duration)
	return err
}

// String returns a Duration as a string.
func (d *Duration) String() string {
	if d != nil {
		duration := time.Duration(*d)
		return duration.String()
	}
	return ""
}

// Set parses the provided string into a Duration.
func (d *Duration) Set(s string) error {
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(duration)

	return nil
}

// ErrHandling defines how to handle parsing errors
type ErrHandling int

const (
	// ContinueOnError will return an err from Parse() if an error is found
	ContinueOnError ErrHandling = iota
	// ExitOnError will call os.Exit(2) if an error is found when parsing
	ExitOnError
	// PanicOnError will panic() if an error is found when parsing flags
	PanicOnError
)

const (
	flagSeparator = "-"
	envSeparator  = "_"
	jsonExtension = ".json"
	tomlExtension = ".toml"
	yamlExtension = ".yaml"
)

var cfgFileExt = []string{jsonExtension, tomlExtension, yamlExtension}

var gf *Gofig

func init() {
	gf = New(ExitOnError)
}

// Gofig is the main gofig structure
type Gofig struct {
	envPrefix   string
	cfgFlagName string
	cfgFiles    []string
	errHandling ErrHandling
	flagSet     *flag.FlagSet
}

// New returns an initialized Gofig instance.
func New(errHandling ErrHandling) *Gofig {
	return &Gofig{
		errHandling: errHandling,
		flagSet:     flag.NewFlagSet(os.Args[0], flag.ContinueOnError),
	}
}

// SetConfigFileFlag adds a config file flag
func SetConfigFileFlag(name string, desc string) {
	gf.SetConfigFileFlag(name, desc)
}

// SetConfigFileFlag adds a config file flag
func (gf *Gofig) SetConfigFileFlag(name string, desc string) {
	gf.cfgFlagName = name
	gf.flagSet.String(gf.cfgFlagName, "", desc)
}

// AddConfigFile adds one or more config file(s) (WITHOUT THE FILE EXTENSION) to try to load a startup.
// Supports JSON (.json), TOML (.toml) and YAML (.yaml) configuration files. Config files
// are tried in order they are added and the search stop at the first existing file.
func AddConfigFile(path ...string) { gf.AddConfigFile(path...) }

// AddConfigFile adds one or more config file(s) (WITHOUT THE FILE EXTENSION) to try to load a startup.
// Supports JSON (.json), TOML (.toml) and YAML (.yaml) configuration files. Config files
// are tried in order they are added and the search stop at the first existing file.
func (gf *Gofig) AddConfigFile(path ...string) {
	gf.cfgFiles = append(gf.cfgFiles, path...)
}

// SetEnvPrefix defines a prefix that ENVIRONMENT variables will use.
// If the prefix is "xyz", environment variables must start with "XYZ_".
func SetEnvPrefix(prefix string) { gf.SetEnvPrefix(prefix) }

// SetEnvPrefix defines a prefix that ENVIRONMENT variables will use.
// If the prefix is "xyz", environment variables must start with "XYZ_".
func (gf *Gofig) SetEnvPrefix(prefix string) {
	gf.envPrefix = prefix
}

// Parse parses the struct to build the flags, parse/decode the optional config file,
// decode the environment variables and finally parse the arguments.
func Parse(v interface{}) { _ = gf.Parse(v) }

// Parse parses the struct to build the flags, parse/decode the optional config file,
// decode the environment variables and finally parse the arguments.
func (gf *Gofig) Parse(v interface{}) error {
	return gf.ParseWithArgs(v, os.Args[1:])
}

// ParseWithArgs parses the struct to build the flags, parse/decode the optional config file,
// decode the environment variables and finally parse the arguments.
func (gf *Gofig) ParseWithArgs(v interface{}, args []string) error {
	err := gf.parse(v, args)
	if err != nil {
		switch gf.errHandling {
		case ExitOnError:
			fmt.Println(err)
			os.Exit(2)
		case PanicOnError:
			panic(err)
		default: // includes ContinueOnError
			return err
		}
	}
	return nil
}

func (gf *Gofig) parse(v interface{}, args []string) (err error) {
	// build the flag list from the struct
	err = parseStruct(v, gf.flagBuilder, "flag")
	if err != nil {
		return err
	}
	// parse the optional config file (override user-defined values)
	err = gf.parseConfigFile(v, args)
	if err != nil {
		return err
	}
	// decode the env variables (override config file values)
	err = parseStruct(v, gf.envDecoder, "env")
	if err != nil {
		return err
	}
	// parse the flags (override the env variables values)
	return gf.flagSet.Parse(args)
}

type fieldParser = func(path []string, val *reflect.Value, tags *reflect.StructTag) error

// errInvalidValue is returned by DecodeEnv when the target provided is not a non-nil pointer to struct.
var errInvalidValue = errors.New("invalid interface value, it must be a non-nil pointer to struct")

// parseStruct recursively parse a struct and call the parser function on each field
func parseStruct(v interface{}, parser fieldParser, cfgTag string, parents ...string) (err error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errInvalidValue
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errInvalidValue
	}

	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		tags := rt.Field(i).Tag

		// support field key renaming/skipping
		key := strings.Split(tags.Get(cfgTag), ",")[0]
		if key == "-" {
			continue
		} else if key == "" {
			key = rt.Field(i).Name
		}
		path := append(parents, strings.ToLower(key))

		// check if it's a struct and if yes we call ourself recursively
		switch f.Kind() {
		case reflect.Ptr:
			if f.Elem().Kind() != reflect.Struct {
				break
			}
			f = f.Elem()
			fallthrough
		case reflect.Struct:
			si := f.Addr().Interface()
			err = parseStruct(si, parser, cfgTag, path...)
			if err != nil {
				return err
			}
			continue
		}

		err = parser(path, &f, &tags)
		if err != nil {
			return err
		}
	}
	return
}

func (gf *Gofig) flagBuilder(path []string, val *reflect.Value, tags *reflect.StructTag) error {
	key := strings.Join(path, flagSeparator)
	desc := tags.Get("desc")

	v := val.Interface()
	pv := val.Addr().Interface()
	switch val.Kind() {
	case reflect.String:
		gf.flagSet.StringVar(pv.(*string), key, v.(string), desc)
	case reflect.Bool:
		gf.flagSet.BoolVar(pv.(*bool), key, v.(bool), desc)
	case reflect.Int:
		gf.flagSet.IntVar(pv.(*int), key, v.(int), desc)
	case reflect.Int64:
		durationPtr, ok := pv.(*Duration)
		if ok {
			gf.flagSet.Var(durationPtr, key, desc)
		} else {
			gf.flagSet.Int64Var(pv.(*int64), key, v.(int64), desc)
		}
	case reflect.Uint:
		gf.flagSet.UintVar(pv.(*uint), key, v.(uint), desc)
	case reflect.Uint64:
		gf.flagSet.Uint64Var(pv.(*uint64), key, v.(uint64), desc)
	case reflect.Float64:
		gf.flagSet.Float64Var(pv.(*float64), key, v.(float64), desc)
	}
	return nil
}

func (gf *Gofig) getEnvKey(path []string) string {
	// build the env key
	if gf.envPrefix != "" {
		path = append([]string{gf.envPrefix}, path...) // prepend the prefix
	}
	return strings.ToUpper(strings.Join(path, envSeparator))
}

func (gf *Gofig) envDecoder(path []string, f *reflect.Value, tags *reflect.StructTag) error {
	key := gf.getEnvKey(path)
	val, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}

	switch f.Kind() {
	case reflect.String:
		f.SetString(val)
	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		f.SetBool(b)
	case reflect.Int, reflect.Int64:
		if f.Type() == reflect.TypeOf(Duration(0)) {
			d, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("error parsing environment variable '%v' with value '%v' into %v", key, val, f.Type())
			}
			f.SetInt(d.Nanoseconds())
		} else {
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil || f.OverflowInt(n) {
				return fmt.Errorf("error parsing environment variable '%v' with value '%v' into %v", key, val, f.Kind())
			}
			f.SetInt(n)
		}
	case reflect.Uint, reflect.Uint64:
		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil || f.OverflowUint(n) {
			return fmt.Errorf("error parsing environment variable '%v' with value '%v' into %v", key, val, f.Kind())
		}
		f.SetUint(n)
	case reflect.Float64:
		n, err := strconv.ParseFloat(val, f.Type().Bits())
		if err != nil || f.OverflowFloat(n) {
			return fmt.Errorf("error parsing environment variable '%v' with value '%v' into %v", key, val, f.Kind())
		}
		f.SetFloat(n)
	}
	return nil
}

func (gf *Gofig) parseConfigFlag(args []string) string {
	name := "-" + gf.cfgFlagName
	for i, a := range args {
		if a == name && len(os.Args) > i+1 {
			return args[i+1]
		}
		as := strings.SplitN(a, "=", 2)
		if as[0] == name && len(as) > 1 {
			return as[1]
		}
	}
	return ""
}

func (gf *Gofig) parseConfigFile(v interface{}, args []string) error {
	cfgFlag := gf.parseConfigFlag(args)

	var f *os.File
	if cfgFlag != "" {
		f, err := os.Open(cfgFlag)
		if err != nil {
			return err
		}
		return gf.decodeConfigFile(f, v)
	}

	for _, cfgFile := range gf.cfgFiles {
		for _, ext := range cfgFileExt {
			var err error
			f, err = os.Open(cfgFile + ext)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return err
			}
			return gf.decodeConfigFile(f, v)
		}
	}
	return nil
}

func (gf *Gofig) decodeConfigFile(f *os.File, v interface{}) error {
	switch filepath.Ext(f.Name()) {
	case jsonExtension:
		return json.NewDecoder(f).Decode(v)
	case tomlExtension:
		_, err := toml.DecodeReader(f, v)
		return err
	case yamlExtension:
		return yaml.NewDecoder(f).Decode(v)
	}
	return fmt.Errorf("config file type not supported")
}
