package sources

import (
    "os"
    "strings"
)

type EnvVarsSource struct {
    VarsSource
}
func (v *EnvVarsSource) MergeConfig(origin string, prefix interface{}) {
    prefix_s := prefix.(string)
    logger.Debugf("Merging env %s vars from %s\n", prefix_s, origin)

    env_vars := make(Vars)
    for _, e := range os.Environ() {
        pair := strings.SplitN(e, "=", 2)
        if name := pair[0]; strings.HasPrefix(name, prefix_s) {
            env_vars[strings.TrimPrefix(name, prefix_s)] = pair[1]
        }
    }
    v.AddHistory(origin + ":" + prefix_s, env_vars)
    v.Vars.Merge(env_vars)
}

func MakeEnvVarsSource() SourceInterface {
    vs := MakeVarsSource()
    return &EnvVarsSource{*vs.(*VarsSource)}
}

func init() {
    RegisterSource("env_vars_prefix", MakeEnvVarsSource, 100, false)
}
