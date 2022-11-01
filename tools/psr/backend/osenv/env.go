// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package osenv

import (
	"fmt"
	"os"
)

type Environment interface {
	LoadFromEnv(cc []EnvVarDesc) error
	GetEnv(string) string
}

type envData struct {
	envVars map[string]string
}

var _ Environment = &envData{}

var GetEnvFunc = os.Getenv

type EnvVarDesc struct {
	Key        string
	DefaultVal string
	Required   bool
}

func NewEnv() Environment {
	return &envData{envVars: make(map[string]string)}
}

// LoadFromEnv get environment vars specified by the from EnvVarDesc list and loads them into a map
func (e *envData) LoadFromEnv(cc []EnvVarDesc) error {
	for _, c := range cc {
		if err := e.addItemConfig(c); err != nil {
			return err
		}
	}
	return nil
}

// GetEnv returns a value for a specific key
func (e *envData) GetEnv(key string) string {
	return e.envVars[key]
}

// addItemToConfig gets the env var item and loads it into a map.
// If the env var is missing and required then return an error
// If the env var is missing and not required then return the default
func (e *envData) addItemConfig(c EnvVarDesc) error {
	val := GetEnvFunc(c.Key)
	if len(val) == 0 {
		if c.Required {
			return fmt.Errorf("Failed, missing required Env var %s", c.Key)
		}
		val = c.DefaultVal
	}
	e.envVars[c.Key] = val
	return nil
}
