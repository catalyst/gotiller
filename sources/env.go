// Infrastructure for env vars

package sources

import (
    "os"
    "strings"

    "github.com/catalyst/gotiller/util"
)

// An env vars version of DeployablesSource type. NOSpecs, just Vars.
type EnvVarsSource struct {
    *DeployablesSource
}
func (v *EnvVarsSource) MergeConfig(origin string, prefix interface{}) {
    prefix_s := prefix.(string)
    logger.Debugf("Merging env %s vars from %s\n", prefix_s, origin)

    env_vars := make(util.AnyMap)
    for _, e := range os.Environ() {
        pair := strings.SplitN(e, "=", 2)
        if name := pair[0]; strings.HasPrefix(name, prefix_s) {
            env_vars[strings.TrimPrefix(name, prefix_s)] = pair[1]
        }
    }
    deployables := util.AnyMap{GlobalVarsKey: env_vars}
    v.DeployablesSource.MergeConfig(origin + " env_vars " + prefix_s, deployables)
}

func MakeEnvVarsSource() SourceInterface {
    ds := MakeDeployablesSource()
    return &EnvVarsSource{ds.(*DeployablesSource)}
}

func init() {
    RegisterSource("env_vars_prefix", MakeEnvVarsSource, 100, false)
}
