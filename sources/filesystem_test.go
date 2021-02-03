package sources

import (
    "path/filepath"
    "fmt"

    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/catalyst/gotiller/util"
)


func Test_Run(t *testing.T) {
    RunTests(t, do_run_tests)
}

func do_run_tests(t *testing.T, dir string) {
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
