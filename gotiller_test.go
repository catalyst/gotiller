package gotiller

import (
    "os"
    "io/ioutil"
    "path/filepath"
    "fmt"
    "strings"
    "runtime"

    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/catalyst/gotiller/util"
)

var t1 = Templates {
    "template1": &Target {
        Target: "target1",
        User  : "user1",
        Vars  : Vars {
            "var1": "val1",
            "varX": "val1X",
        },
    },
    "templateX": &Target {
        Target: "targetX1",
        User  : "user1",
        Group : "group1",
        Vars  : Vars {
            "var1": "val1",
            "varX": "val1X",
        },
    },
}
var t2 = Templates {
    "template2": &Target {
        Target: "target2",
        User  : "user2",
        Vars  : Vars {
            "var1": "val2",
            "varX": "val2X",
        },
    },
    "templateX": &Target {
        Target: "targetX2",
        User  : "user2",
        Perms : os.FileMode(0644),
        Vars  : Vars {
            "var2": "val2",
            "varX": "val2X",
        },
    },
}
var t1_t2 = Templates {
    "template1": &Target {
        Target: "target1",
        User  : "user1",
        Vars  : Vars {
            "var1": "val1",
            "varX": "val1X",
        },
    },
    "template2": &Target {
        Target: "target2",
        User  : "user2",
        Vars  : Vars {
            "var1": "val2",
            "varX": "val2X",
        },
    },
    "templateX": &Target {
        Target: "targetX2",
        User  : "user2",
        Group : "group1",
        Perms : os.FileMode(0644),
        Vars  : Vars {
            "var1": "val1",
            "var2": "val2",
            "varX": "val2X",
        },
    },
}
func Test_merge_templates(t *testing.T) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    tr := make(Templates)
    tr.merge(t1, t2)

    assert.Equal(t, t1_t2, tr, "merge_templates()")
}

var c1 = Config {
    Defaults          : t1,
    DefaultVars       : Vars {
        "varA": "valA1",
        "varB": "valB1",
    },
    DefaultEnvironment: "env1",
    EnvVarsPrefix     : "prefix1",
    Environments      : map[string]Templates {
        "env1": t1,
        "env2": t2,
    },
}
var c2 = Config {
    Defaults          : t2,
    DefaultVars       : Vars {
        "varA": "valA2",
    },
    DefaultEnvironment: "env2",
    Environments      : map[string]Templates {
        "env1": t2,
    },
}
var c1_c2 = Config {
    Defaults          : t1_t2,
    DefaultVars       : Vars {
        "varA": "valA2",
        "varB": "valB1",
    },
    DefaultEnvironment: "env2",
    EnvVarsPrefix     : "prefix1",
    Environments      : map[string]Templates {
        "env1": t1_t2,
        "env2": t2,
    },
}
func Test_config_merge(t *testing.T) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    c1.merge(&c2)
    assert.Equal(t, c1_c2, c1, "Config.merge()")
}


func Test_extract_env_vars(t *testing.T) {
    env_vars_prefix := "gotiller_test_"

    defer clear_env(env_vars_prefix)

    var_a := "a"

    clear_env(env_vars_prefix)
    os.Setenv(env_vars_prefix + var_a, var_a)

    assert.Equal(t, Vars{var_a: var_a}, extract_env_vars(env_vars_prefix), "env_vars_prefix()")
}

func clear_env(prefix string) {
    for _, e := range os.Environ() {
        pair := strings.SplitN(e, "=", 2)
        if name := pair[0]; strings.HasPrefix(name, prefix) {
            if err := os.Unsetenv(name); err != nil {
                panic (err.Error())
            }
        }
    }
}

const perms = os.FileMode(0600)
var target = Target {
    Target: "/etc/dummy/target.conf",
    Perms : perms,
    Vars  : Vars {
        "varT": "valT",
        "varA": "valTA",
        "varB": "valTB",
        "varC": "valTC",
    },
}
var vars = Vars {
    "varV": "valV",
    "varA": "valVA",
    "varB": "valVB",
    "varD": "valVD",
}
var default_vars = Vars {
    "varDV": "valDV",
    "varA": "valDVA",
    "varC": "valDVC",
    "varD": "valDVD",
}
var merged_vars = Vars {  // default_vars + target.Vars + vars
    "varT": "valT",
    "varV": "valV",
    "varDV": "valDV",
    "varA": "valVA",
    "varB": "valVB",
    "varC": "valTC",
    "varD": "valVD",
}
const template_in string = `
{{.varA}}
{{.varB}}
{{.varC}}
{{.varD}}
{{.varT}}
{{.varV}}
{{.varDV}}
`
const content_out string = `
valVA
valVB
valTC
valVD
valT
valV
valDV
`
func Test_merge_vars(t *testing.T) {
    vr := make(Vars)
    vr.merge(default_vars, target.Vars, vars)

    assert.Equal(t, merged_vars, vr, "merge_vars()")
}

var (
    _, b, _, _ = runtime.Caller(0)
    base_dir   = filepath.Dir(b)
)

const x_val = "v_from_env_x"

func Test_Execute(t *testing.T) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    scenarios_dir := filepath.Join(base_dir, "test-execute")

    dir_entries, err := ioutil.ReadDir(scenarios_dir)
    if err != nil {
        panic(err)
    }

    for _, entry := range dir_entries {
        if entry.IsDir() {
            scenario := entry.Name()
            dir := filepath.Join(scenarios_dir, scenario)

            do_execute_tests(dir, t)
        }
    }
}

func do_execute_tests(dir string, t *testing.T) {
    config_path := filepath.Join(dir, ConfigFname)
    environments_dir := filepath.Join(dir, EnvironmentsSubdir)

    environments := make(map[string]bool)

    config := slurp_config(config_path)
    if config.Environments != nil {
        for env := range config.Environments {
            environments[env] = true
        }
    }

    environments_dir_entries, err := ioutil.ReadDir(environments_dir)
    if err != nil {
        panic(err)
    }
    for _, environment_entry := range environments_dir_entries {
        environment := strings.TrimSuffix(environment_entry.Name(), ".yaml")
        environments[environment] = true
    }

    assert.NotEmpty(t, environments, "environments to test " + dir)

    if config.EnvVarsPrefix != "" {
        defer clear_env(config.EnvVarsPrefix)

        clear_env(config.EnvVarsPrefix)
        os.Setenv(config.EnvVarsPrefix + "x", x_val)
    }

    logger.SetDebug(true)
    gt := New(dir)

    for environment, _ := range environments {
        t.Run(fmt.Sprint(dir, environment), func(t *testing.T) {
            // fmt.Printf("%#v\n", os.Environ())
            t.Parallel()

            target_dir := t.TempDir()

            gt.Execute(environment, target_dir)
            assert_execution(t, dir, environment, target_dir)

            do_var_chain_tests(dir, environment, gt, t)
        })
    }

    if config.DefaultEnvironment != "" {
        t.Parallel()

        target_dir := t.TempDir()

        gt.Execute("", target_dir)
        assert_execution(t, dir, config.DefaultEnvironment, target_dir)
    }
}

type ExpectedVarsChains map[string]VarsChain

func do_var_chain_tests(dir string, environment string, gt *GoTiller, t *testing.T) {
    expected_path := filepath.Join(dir, "vars_chain", environment + ".yaml")
    expected_vcs := make(ExpectedVarsChains)
    util.ReadYaml(expected_path, expected_vcs)

    for tpl, expected_vc := range expected_vcs {
        assert.Equal(t, expected_vc, gt.DumpVarsChain(environment, tpl), expected_path + " for " + tpl)
    }
}

func assert_execution(t *testing.T, dir string, environment string, result_dir string) {
    defer func() {
        if r := recover(); r != nil {
            err := util.PrintDirTree(result_dir)
            if err != nil {
                fmt.Println(err)
            }

            panic(r)
        }
    }()

    expected_dir := filepath.Join(dir, "results", environment)
    err := filepath.Walk(expected_dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() {
            rel_path := strings.TrimPrefix(path, expected_dir)
            expected_bytes := util.SlurpFile(path)

            result_path := filepath.Join(result_dir, rel_path)
            target_bytes := util.SlurpFile(result_path)

            assert.Equal(t, string(expected_bytes), string(target_bytes), rel_path + " content")
        }

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
