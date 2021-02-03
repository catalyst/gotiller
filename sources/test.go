// Test helpers, for tests that execute test-run tests

package sources

import (
    "os"
    "path/filepath"
    "strings"
    "fmt"
    "runtime"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/catalyst/gotiller/util"
)

var (
    _, b, _, _      = runtime.Caller(0)
    base_dir        = filepath.Dir(filepath.Dir(b))
    TestsDefBaseDir = filepath.Join(base_dir, "test-run")
)

// Env vars that will be set for each run
var TestEnvVars = map[string]string {
    "x": "v_from_env_x",
}

// Helper type to set/clear env vars
type EnvForPrefix string
func (ep *EnvForPrefix) Clear() {
    prefix := string(*ep)
    if prefix == "" {
        return;
    }

    for _, e := range os.Environ() {
        pair := strings.SplitN(e, "=", 2)
        if name := pair[0]; strings.HasPrefix(name, prefix) {
            if err := os.Unsetenv(name); err != nil {
                panic (err.Error())
            }
        }
    }
}
func (ep *EnvForPrefix) Set(v, val string) {
    prefix := string(*ep)
    if prefix == "" {
        return;
    }

    os.Setenv(prefix + v, val)
}

// Find suitable env_vars_prefix in config(s)
func FindEnvVarsPrefix (dir string) EnvForPrefix {
    var env_vars_prefix string

    config_path := filepath.Join(dir, ConfigFname)
    if _, err := os.Stat(config_path); err == nil {
        config := LoadConfigFile(config_path)
        if prefix, exists := config["env_vars_prefix"]; exists {
            env_vars_prefix = prefix.(string)
        }
    }

    config_pattern := filepath.Join(dir, ConfigD, "*" + ConfigSuffix)
    if matches, _ := filepath.Glob(config_pattern); matches != nil {
        for _, m := range matches {
            config := LoadConfigFile(m)
            if prefix, exists := config["env_vars_prefix"]; exists {
                env_vars_prefix = prefix.(string)
            }
        }
    }

    return EnvForPrefix(env_vars_prefix)
}

// test-run iterator, that sets up the stage - env vars
func RunTests(t *testing.T, test_fn func(t *testing.T, dir string)) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    logger.SetDebug(true)

    dir_entries := util.ReadDir(TestsDefBaseDir)

    for _, entry := range dir_entries {
        if entry.IsDir() {
            scenario := entry.Name()
            dir := filepath.Join(TestsDefBaseDir, scenario)

            ep := FindEnvVarsPrefix(dir)

            defer ep.Clear()

            ep.Clear()
            for name, val := range TestEnvVars {
                ep.Set(name, val)
            }

            test_fn(t, dir)
        }
    }
}

// Helper type for setting up executed vars chains expectations
type VarsLink struct {
    Source string
    Vars
}
type VarsChain []*VarsLink
func (vcp *VarsChain) append(source string, vars Vars) {
    vc := *vcp
    vc = append(vc, &VarsLink{source, vars})
    *vcp = vc
}

// Marshal Processor sources vars for environment/template into VarsChain
func DumpVarsChain(p *Processor, environment string, tpl string) VarsChain {
    var vc VarsChain

    for _, si := range p.Sources {
        ds := si.DeployablesForEnvironment(environment)
        v := ds.Vars

        if v != nil {
            vc.append(si.Name + " vars", v)
        }

        if ds != nil {
            if t, exists := ds.Specs[tpl]; exists {
                vc.append(si.Name, t.Vars)
            }
        }
    }

    return vc
}

type ExpectedVarsChains map[string]VarsChain

// Assert that Processor vars chains for environment templates match expectations.
// Expectations are stored in vars_chain/<environment>.yaml files.
// What we are testing here is that config.d/ values superseed common.yaml,
// and environments/* superseed common,yaml environments
func AssertVarsChain(t *testing.T, p *Processor, dir string, environment string) {
    expected_path := filepath.Join(dir, "vars_chain", environment + ".yaml")
    expected_vcs := make(ExpectedVarsChains)
    util.ReadYaml(expected_path, expected_vcs)

    for tpl, expected_vc := range expected_vcs {
        assert.Equal(t, expected_vc, DumpVarsChain(p, environment, tpl), expected_path + " for " + tpl)
    }
}

// Assert that processed templates match expectations storet in the results/ dir
func AssertRunForEnvironment(t *testing.T, dir string, environment string, result_dir string) {
    defer func() {
        if r := recover(); r != nil {
            err := util.PrintDirTree(result_dir)
            if err != nil {
                fmt.Println(err)
            }

            panic(r)
        }
    }()

    expected_dir := util.ResolveLink( filepath.Join(dir, "results", environment) )
    err := filepath.Walk(expected_dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if info.IsDir() {
            return nil
        }

        if fname := filepath.Base(path); strings.HasPrefix(fname, ".") {
            return nil
        }

        rel_path := strings.TrimPrefix(path, expected_dir)
        expected_bytes := util.SlurpFile(path)

        result_path := filepath.Join(result_dir, rel_path)
        target_bytes := util.SlurpFile(result_path)

        assert.Equal(t, string(expected_bytes), string(target_bytes), rel_path + " content")

        return nil
    })
    assert.Nil(t, err)

    err = filepath.Walk(result_dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() {
            rel_path := strings.TrimPrefix(path, result_dir)
            expected_path := filepath.Join(expected_dir, rel_path)
            _, err := os.Stat(expected_path)
            if err != nil {
                assert.Fail(t, "Unexpected generated file " + rel_path)
            }
        }

        return nil
    })
    assert.Nil(t, err)
}
