package sources

import (
    "path/filepath"
    "fmt"
    "runtime"

    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/catalyst/gotiller/util"
)

var (
    _, b, _, _ = runtime.Caller(0)
    base_dir   = filepath.Dir(filepath.Dir(b))
)

const x_val = "v_from_env_x"

func Test_Deploy(t *testing.T) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    logger.SetDebug(true)

    scenarios_dir := filepath.Join(base_dir, "test-execute")

    dir_entries := util.ReadDir(scenarios_dir)

    for _, entry := range dir_entries {
        if entry.IsDir() {
            scenario := entry.Name()
            dir := filepath.Join(scenarios_dir, scenario)

            do_execute_tests(dir, t)
        }
    }
}

func do_execute_tests(dir string, t *testing.T) {
    ep := FindEnvVarsPrefix(dir)

    defer ep.Clear()

    ep.Clear()
    ep.Set("x", x_val)

    processor := LoadConfigsFromDir(dir)

    environments := processor.ListEnvironments()

    expected_environments := util.SlurpFileAsLines( filepath.Join(dir, "environments.list") )
    assert.Equal(t, expected_environments, environments, "ListEnvironments()")

    for _, environment := range environments {
        t.Run(fmt.Sprint(dir, environment), func(t *testing.T) {
            // fmt.Printf("%#v\n", os.Environ())
            t.Parallel()

            target_dir := t.TempDir()

            processor.RunForEnvironment(environment, target_dir)
            AssertRunForEnvironment(t, dir, environment, target_dir)

            TestVarsChain(t, processor, dir, environment)
        })
    }
}
