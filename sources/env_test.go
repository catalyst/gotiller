package sources

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

const env_vars_prefix string = "gotiller_test_"

func Test_EnvVarsSource(t *testing.T) {
    ep := EnvForPrefix(env_vars_prefix)

    defer ep.Clear()

    var_a := "a"

    ep.Clear()
    ep.Set(var_a, var_a)

    evs := MakeEnvVarsSource()

    evs.MergeConfig("test", env_vars_prefix)

    assert.Equal(t, Vars{var_a: var_a}, evs.(*EnvVarsSource).DeployablesSource.Vars, "Test_EnvVarsSource.()")
}
