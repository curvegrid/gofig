// Copyright (c) 2019 Curvegrid Inc.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gofig

import (
	"fmt"
	"os"
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

func TestSetEnvPrefix(t *testing.T) {
	// Case 2: env prefix set
	expected := "env"
	os.Setenv("GF_STR", expected)

	s := &TestStruct{}
	gf = New(ContinueOnError)
	gf.SetEnvPrefix("GF")
	err := gf.ParseWithArgs(s, []string{})
	assert.NoError(t, err)

	os.Unsetenv("GF_STR")
	assert.Equal(t, expected, s.Str)
}

func TestAddConfigFile(t *testing.T) {
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

func TestAddConfigFlag(t *testing.T) {
	// Case 1: existing file
	for _, fileExt := range cfgFileExt {
		t.Run(fileExt[1:], func(t *testing.T) {
			s := &TestStruct{}
			cfgFile := "gofig_test_" + fileExt[1:] + fileExt

			gf := New(ContinueOnError)
			gf.SetConfigFileFlag("c", "My test config file")
			args := []string{"-c", cfgFile}
			err := gf.ParseWithArgs(s, args)
			assert.NoError(t, err)

			assert.Equal(t, "config-file", s.Str)
		})
	}

	// Case 2: non-existing file (must return an error)
	t.Run("non-existing", func(t *testing.T) {
		s := &TestStruct{}

		gf = New(ContinueOnError)
		gf.SetConfigFileFlag("c", "My test config file")
		args := []string{"-c", "fake_file.toml"}
		err := gf.ParseWithArgs(s, args)
		assert.Error(t, err)
	})
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
