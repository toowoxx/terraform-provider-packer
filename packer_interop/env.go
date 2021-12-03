package packer_interop

import (
	"os"
	"strings"
)

func EnvVars(additionalEnvVars map[string]string, passThroughCurrent bool) map[string]string {
	envVars := map[string]string{}
	if passThroughCurrent {
		for _, envVarStr := range os.Environ() {
			split := strings.SplitN(envVarStr, "=", 2)
			if len(split) != 2 {
				continue
			}
			envVars[split[0]] = split[1]
		}
	}
	for key, value := range additionalEnvVars {
		envVars[key] = value
	}
	envVars[TPPRunPacker] = "true"
	return envVars
}
