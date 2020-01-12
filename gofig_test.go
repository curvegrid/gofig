// Copyright (c) 2019 Curvegrid Inc.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gofig

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Str      string
	Bool     bool
	Int      int
	Int64    int64
	Uint     uint
	Uint64   uint64
	Float    float64
	Duration Duration
	Sub      SubTestStruct
	Skipped  string `json:"-" toml:"-" yaml:"-" env:"-" flag:"-"`
}

type SubTestStruct struct {
	RenamedStr string `json:"str" toml:"str" yaml:"str" env:"str" flag:"str"`
}

func TestSetEnvPrefixPackage(t *testing.T) {
	// Case 2: env prefix set
	expected := "env"
	os.Setenv("GF_STR", expected)

	s := &TestStruct{}
	gf = New(ContinueOnError) // package-level
	SetEnvPrefix("GF")
	err := gf.ParseWithArgs(s, []string{})
	assert.NoError(t, err)

	os.Unsetenv("GF_STR")
	assert.Equal(t, expected, s.Str)
}

func TestSetEnvPrefixObject(t *testing.T) {
	// Case 2: env prefix set
	expected := "env"
	os.Setenv("GF_STR", expected)

	s := &TestStruct{}
	gf := New(ContinueOnError)
	gf.SetEnvPrefix("GF")
	err := gf.ParseWithArgs(s, []string{})
	assert.NoError(t, err)

	os.Unsetenv("GF_STR")
	assert.Equal(t, expected, s.Str)
}

func TestAddConfigFilePackage(t *testing.T) {
	s := &TestStruct{}

	for _, fileExt := range cfgFileExt {
		fileExt := fileExt[1:] // trim the "."
		t.Run(fileExt, func(t *testing.T) {
			gf = New(ContinueOnError) // package-level
			AddConfigFile("fake_file1")
			AddConfigFile("gofig_test_" + fileExt)
			AddConfigFile("fake_file2")
			err := gf.ParseWithArgs(s, []string{})

			assert.NoError(t, err)
			assert.Equal(t, "config-file", s.Str)
		})
	}

}

func TestAddConfigFileObject(t *testing.T) {
	s := &TestStruct{}

	for _, fileExt := range cfgFileExt {
		fileExt := fileExt[1:] // trim the "."
		t.Run(fileExt, func(t *testing.T) {
			gf := New(ContinueOnError)
			gf.AddConfigFile("fake_file1")
			gf.AddConfigFile("gofig_test_" + fileExt)
			gf.AddConfigFile("fake_file2")
			err := gf.ParseWithArgs(s, []string{})
			assert.NoError(t, err)

			assert.Equal(t, "config-file", s.Str)
		})
	}
}

func TestParsePanic(t *testing.T) {
	gf := New(PanicOnError)
	assert.Panics(t, func() {
		_ = gf.ParseWithArgs(nil, nil)
	})
}

func TestAddConfigFlagPackage(t *testing.T) {
	gf = New(ContinueOnError) // package-level gf
	SetConfigFileFlag("c", "My test config file")

	// Case 1: existing file
	s := &TestStruct{}
	args := []string{"-c", "gofig_test_toml.toml"}
	err := gf.ParseWithArgs(s, args)
	assert.NoError(t, err)

	assert.Equal(t, "config-file", s.Str)
}

const (
	gfPackage = "package"
	gfObject  = "object"
)

var gfTypes = []string{gfPackage, gfObject}

func TestAddConfigFlag(t *testing.T) {
	for _, gfType := range gfTypes {
		t.Run(gfType, func(t *testing.T) {
			// Case 1: existing file
			for _, fileExt := range cfgFileExt {
				t.Run(fileExt[1:], func(t *testing.T) {
					s := &TestStruct{}
					cfgFile := "gofig_test_" + fileExt[1:] + fileExt
					args := []string{"-c", cfgFile}
					var err error

					if gfType == gfPackage {
						gf = New(ContinueOnError) // package-level
						SetConfigFileFlag("c", "My test config file")
						err = gf.ParseWithArgs(s, args)
					} else {
						gf := New(ContinueOnError)
						gf.SetConfigFileFlag("c", "My test config file")
						err = gf.ParseWithArgs(s, args)
					}

					assert.NoError(t, err)
					assert.Equal(t, "config-file", s.Str)
				})
			}

			// Case 2: non-existing file (must return an error)
			t.Run("non-existing", func(t *testing.T) {
				s := &TestStruct{}
				args := []string{"-c", "fake_file.toml"}
				var err error

				if gfType == gfPackage {
					gf = New(ContinueOnError) // package-level
					SetConfigFileFlag("c", "My test config file")
					err = gf.ParseWithArgs(s, args)
				} else {
					gf := New(ContinueOnError)
					gf.SetConfigFileFlag("c", "My test config file")
					err = gf.ParseWithArgs(s, args)
				}

				assert.Error(t, err)
			})

			// Case 3: unsupported extension (must return an error)
			t.Run("unsupported extension", func(t *testing.T) {
				s := &TestStruct{}
				args := []string{"-c", "gofig_test_unsuppext.unsuppext"}
				var err error

				if gfType == gfPackage {
					gf = New(ContinueOnError) // package-level
					SetConfigFileFlag("c", "My test config file")
					err = gf.ParseWithArgs(s, args)
				} else {
					gf := New(ContinueOnError)
					gf.SetConfigFileFlag("c", "My test config file")
					err = gf.ParseWithArgs(s, args)
				}

				assert.EqualError(t, err, "config file type not supported")
			})
		})
	}
}

func TestParse(t *testing.T) {
	for _, fileExt := range cfgFileExt {
		t.Run(fileExt, func(t *testing.T) {
			cfgFile := "gofig_test_" + fileExt[1:] + fileExt

			// Case 1: User defined only (nothing should be overridden)
			t.Run("UserDefined", testParseUserDefined(cfgFile))

			// Case 2: config only (no env or flags)
			t.Run("Config", testParseConfig(cfgFile))

			// Case 3: config + env (no flags)
			t.Run("ConfigEnv", testParseConfigEnv(cfgFile))

			// Case 4: config + env + flags
			t.Run("ConfigEnvFlag", testParseConfigEnvFlag(cfgFile))
		})
	}
}

func buildTestStruct() *TestStruct {
	return &TestStruct{
		Str:      "user-defined",
		Bool:     false,
		Int:      -99,
		Int64:    -99,
		Uint:     99,
		Uint64:   99,
		Float:    1.99,
		Duration: Duration(time.Duration(99) * time.Second),
		Sub: SubTestStruct{
			RenamedStr: "renamed-user-defined",
		},
	}
}

func testParseUserDefined(cfgFile string) func(t *testing.T) {
	return func(t *testing.T) {
		s := struct {
			UserDefined TestStruct
		}{
			UserDefined: *buildTestStruct(),
		}
		expected := s

		gf := New(ContinueOnError)
		gf.SetConfigFileFlag("c", "My test config file")
		args := []string{"-c", cfgFile}
		err := gf.ParseWithArgs(&s, args)
		assert.NoError(t, err)

		assert.Equal(t, expected, s)
	}
}

func testParseConfig(cfgFile string) func(t *testing.T) {
	return func(t *testing.T) {
		s := buildTestStruct()

		expected := &TestStruct{
			Str:      "config-file",
			Bool:     true,
			Int:      -1,
			Int64:    -1,
			Uint:     1,
			Uint64:   1,
			Float:    1.1,
			Duration: Duration(time.Duration(1) * time.Second),
			Sub: SubTestStruct{
				RenamedStr: "renamed-config-file",
			},
		}

		gf := New(ContinueOnError)
		gf.SetConfigFileFlag("c", "My test config file")
		args := []string{"-c", cfgFile}
		err := gf.ParseWithArgs(s, args)
		assert.NoError(t, err)

		assert.Equal(t, expected, s)
	}
}

func testParseConfigEnv(cfgFile string) func(t *testing.T) {
	return func(t *testing.T) {
		s := buildTestStruct()

		expected := &TestStruct{
			Str:      "env",
			Bool:     false,
			Int:      -2,
			Int64:    -2,
			Uint:     2,
			Uint64:   2,
			Float:    2.2,
			Duration: Duration(time.Duration(2) * time.Second),
			Sub: SubTestStruct{
				RenamedStr: "renamed-env",
			},
		}

		os.Setenv("GF_STR", expected.Str)
		os.Setenv("GF_BOOL", fmt.Sprintf("%v", expected.Bool))
		os.Setenv("GF_INT", fmt.Sprintf("%v", expected.Int))
		os.Setenv("GF_INT64", fmt.Sprintf("%v", expected.Int64))
		os.Setenv("GF_UINT", fmt.Sprintf("%v", expected.Uint))
		os.Setenv("GF_UINT64", fmt.Sprintf("%v", expected.Uint64))
		os.Setenv("GF_FLOAT", fmt.Sprintf("%v", expected.Float))
		os.Setenv("GF_DURATION", fmt.Sprintf("%v", expected.Duration.String()))
		os.Setenv("GF_SKIPPED", "env")
		os.Setenv("GF_SUB_STR", expected.Sub.RenamedStr)

		gf = New(ContinueOnError)
		gf.SetEnvPrefix("GF")
		gf.SetConfigFileFlag("c", "My test config file")
		args := []string{"-c", cfgFile}
		err := gf.ParseWithArgs(s, args)
		assert.NoError(t, err)

		assert.Equal(t, expected, s)
	}
}

func testParseConfigEnvFlag(cfgFile string) func(t *testing.T) {
	return func(t *testing.T) {
		s := buildTestStruct()

		expected := &TestStruct{
			Str:      "flag",
			Bool:     true,
			Int:      -3,
			Uint:     3,
			Float:    3.3,
			Duration: Duration(time.Duration(3) * time.Second),
			Sub: SubTestStruct{
				RenamedStr: "renamed-flag",
			},
		}

		os.Setenv("GF_STR", "env")
		os.Setenv("GF_BOOL", fmt.Sprintf("%v", "false"))
		os.Setenv("GF_INT", fmt.Sprintf("%v", "-2"))
		os.Setenv("GF_INT64", fmt.Sprintf("%v", "-2"))
		os.Setenv("GF_UINT", fmt.Sprintf("%v", "2"))
		os.Setenv("GF_UINT64", fmt.Sprintf("%v", "2"))
		os.Setenv("GF_FLOAT", fmt.Sprintf("%v", "2.2"))
		os.Setenv("GF_DURATION", fmt.Sprintf("%v", "2s"))
		os.Setenv("GF_SKIPPED", "env")
		os.Setenv("GF_SUB_STR", "env")

		args := []string{
			"-c", cfgFile,
			"-str", expected.Str,
			"-bool",
			"-int", fmt.Sprintf("%v", expected.Int),
			"-int64", fmt.Sprintf("%v", expected.Int64),
			"-uint", fmt.Sprintf("%v", expected.Uint),
			"-uint64", fmt.Sprintf("%v", expected.Uint64),
			"-float", fmt.Sprintf("%v", expected.Float),
			"-duration", fmt.Sprintf("%v", expected.Duration.String()),
			"-sub-str", expected.Sub.RenamedStr,
		}

		gf = New(ContinueOnError)
		gf.SetEnvPrefix("GF")
		gf.SetConfigFileFlag("c", "My test config file")
		err := gf.ParseWithArgs(s, args)
		assert.NoError(t, err)

		assert.Equal(t, expected, s)
	}
}

type EnvErrorTest struct {
	Name   string
	Type   reflect.Type
	ErrMsg string
}

func TestParseConfigEnvErrors(t *testing.T) {
	tests := []EnvErrorTest{
		{
			Name:   "BOOL",
			Type:   reflect.TypeOf(false),
			ErrMsg: `strconv.ParseBool: parsing "unparseable": invalid syntax`,
		},
		{
			Name:   "INT",
			Type:   reflect.TypeOf(int(-4)),
			ErrMsg: `error parsing environment variable 'GF_INT' with value 'unparseable' into int`,
		},
		{
			Name:   "INT64",
			Type:   reflect.TypeOf(int64(-4)),
			ErrMsg: `error parsing environment variable 'GF_INT64' with value 'unparseable' into int64`,
		},
		{
			Name:   "UINT",
			Type:   reflect.TypeOf(uint(4)),
			ErrMsg: `error parsing environment variable 'GF_UINT' with value 'unparseable' into uint`,
		},
		{
			Name:   "UINT64",
			Type:   reflect.TypeOf(uint64(4)),
			ErrMsg: `error parsing environment variable 'GF_UINT64' with value 'unparseable' into uint64`,
		},
		{
			Name:   "FLOAT",
			Type:   reflect.TypeOf(float64(4.4)),
			ErrMsg: `error parsing environment variable 'GF_FLOAT' with value 'unparseable' into float64`,
		},
		{
			Name:   "DURATION",
			Type:   reflect.TypeOf(Duration(0)),
			ErrMsg: `error parsing environment variable 'GF_DURATION' with value 'unparseable' into gofig.Duration`,
		},
	}

	for _, test := range tests {
		os.Setenv("GF_"+test.Name, "unparseable")

		sType := reflect.StructOf([]reflect.StructField{{
			Name: test.Name,
			Type: test.Type,
		}})

		s := reflect.New(sType).Interface()

		gf := New(ContinueOnError)
		gf.SetEnvPrefix("GF")
		err := gf.ParseWithArgs(s, nil)
		assert.EqualError(t, err, test.ErrMsg)
	}
}

type UintStruct struct {
	D uint
}

type IntStruct struct {
	A int
	B *IntStruct
	C UintStruct
}

func TestParseStruct(t *testing.T) {
	gf := New(ContinueOnError)

	// Case: nil v
	err := parseStruct(nil, gf.flagBuilder, "flag")
	assert.EqualError(t, err, errInvalidValue.Error())

	// Case: non-pointer v
	var nonPtr int
	err = parseStruct(nonPtr, gf.flagBuilder, "flag")
	assert.EqualError(t, err, errInvalidValue.Error())

	// Case: non-struct v
	var nonStruct int = 42
	err = parseStruct(&nonStruct, gf.flagBuilder, "flag")
	assert.EqualError(t, err, errInvalidValue.Error())

	// Case: embedded struct
	embeddedStruct := &IntStruct{
		A: 5,
		B: &IntStruct{
			A: 6,
			B: nil,
		},
	}
	err = parseStruct(embeddedStruct, gf.flagBuilder, "flag")
	assert.NoError(t, err)

	// Case: embedded struct parse error
	gf = New(ContinueOnError)
	embeddedStruct = &IntStruct{
		B: &IntStruct{
			C: UintStruct{
				D: 7,
			},
		},
	}

	const failedParse = "Error: failed parse"

	failParser := func(path []string, val *reflect.Value, tags *reflect.StructTag) error {
		if strings.Join(path, flagSeparator) == "b-c-d" {
			return errors.New(failedParse)
		}

		return gf.flagBuilder(path, val, tags)
	}

	err = parseStruct(embeddedStruct, failParser, "flag")
	assert.EqualError(t, err, failedParse)
}

func TestParseConfigFlag(t *testing.T) {
	gf := New(ContinueOnError)
	gf.SetConfigFileFlag("c", "My test config file")

	cfgFile := "config_file.toml"
	args := []string{
		"-c=" + cfgFile,
	}

	cfgFlag := gf.parseConfigFlag(args)

	assert.Equal(t, cfgFile, cfgFlag)
}

func TestDuration(t *testing.T) {
	// Case: string value of nil duration
	var d *Duration
	val := d.String()
	assert.Equal(t, "", val)

	// Case: unable to parse duration
	err := d.Set("invalid duration")
	assert.EqualError(t, err, "time: invalid duration invalid duration")
}
