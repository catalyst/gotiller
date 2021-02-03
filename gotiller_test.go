package gotiller

import (
    "os"
    "flag"
    "fmt"

    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/catalyst/gotiller/util"
    "github.com/catalyst/gotiller/sources"
)

const bogus_environment = "blah"
const panic_nothing_to_do = "Nothing to do for environment " + bogus_environment

var test_dir string

func TestMain(m *testing.M) {
    flag.StringVar(&test_dir, "d", "", "Test config/results dir")
    flag.Parse()
    // logger.Panicf("%#v %s", os.Args, test_dir)
    os.Exit(m.Run())
}

func Test_Execute(t *testing.T) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    if test_dir != "" {
        fmt.Printf("Running for special dir %s\n", test_dir)
        do_execute_test(t, test_dir)
        return
    }

    sources.RunTests(t, func(t *testing.T, dir string) {
        t.Run(fmt.Sprint(dir), func(t *testing.T) {
            t.Parallel()

            do_execute_test(t, dir)
        })
    })
}

func do_execute_test(t *testing.T, dir string) {
    target_dir := t.TempDir()

    defer func() {
        if r := recover(); r != nil {
            assert.Equal(t, panic_nothing_to_do, r)
        }
    }()

    Execute(dir, "", target_dir, true)
    sources.AssertRunForEnvironment(t, dir, "default", target_dir)
}

func Test_ExecuteNothing(t *testing.T) {
    conf_dir := t.TempDir()
    assert.PanicsWithValue(t, panic_nothing_to_do, func () {
        target_dir := t.TempDir()
        Execute(conf_dir, bogus_environment, target_dir, true)
    }, "Execute() in bogus directory")
}
