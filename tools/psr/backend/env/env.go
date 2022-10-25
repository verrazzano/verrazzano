package env

import (
	"fmt"
	"os"
)

type EnvLoader interface {
	LoadFromEnv(cc []EnvVarDesc, env map[string]string) error
	GetEnv(string) string
}

type Env struct {
	EnvVars map[string]string
}

type EnvVarDesc struct {
	Key        string
	DefaultVal string
	Required   bool
}

// LoadFromEnv get environment vars from the os and loads them into a map
func (e *Env) LoadFromEnv(cc []EnvVarDesc) error {
	for _, c := range cc {
		if err := e.addItemConfig(c); err != nil {
			return err
		}
	}
	return nil
}

// addItemToConfig gets the env var item and loads it into a map.
// If the env var is missing and required then return an error
// If the env var is missing and not required then return the default
func (e *Env) addItemConfig(c EnvVarDesc) error {
	val := os.Getenv(c.Key)
	if len(val) == 0 {
		if c.Required {
			return fmt.Errorf("Failed, missing required Env var %s", c.Key)
		}
		val = c.DefaultVal
	}
	e.EnvVars[c.Key] = val
	return nil
}
