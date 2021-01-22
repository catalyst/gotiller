package gotiller

import (
    "path/filepath"
    "fmt"
    "runtime"

    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/catalyst/gotiller/util"
    "github.com/catalyst/gotiller/sources"
)

var (
    _, b, _, _ = runtime.Caller(0)
    base_dir   = filepath.Dir(b)
)

const x_val = "v_from_env_x"
const panic_nothing_to_do = "Nothing to do\n"

func Test_Execute(t *testing.T) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    scenarios_dir := filepath.Join(base_dir, "test-execute")

    dir_entries := util.ReadDir(scenarios_dir)

    for _, entry := range dir_entries {
        if entry.IsDir() {
            scenario := entry.Name()
            dir := filepath.Join(scenarios_dir, scenario)

            do_execute_tests(dir, t)
        }
    }

    conf_dir := t.TempDir()
    assert.PanicsWithValue(t, panic_nothing_to_do, func () {
        target_dir := t.TempDir()
        Execute(conf_dir, "blah", target_dir, true)
    }, "Execute() in bogus directory")
}

func do_execute_tests(dir string, t *testing.T) {
    ep := sources.FindEnvVarsPrefix(dir)

    t.Run(fmt.Sprint(dir), func(t *testing.T) {
        t.Parallel()

        ep.Clear()
        ep.Set("x", x_val)

        target_dir := t.TempDir()

        defer func() {
            if r := recover(); r != nil {
                assert.Equal(t, panic_nothing_to_do, r)
            }
        }()

        Execute(dir, "", target_dir, true)
        sources.AssertRunForEnvironment(t, dir, "default", target_dir)
    })
}
