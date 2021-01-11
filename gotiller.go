package gotiller

import (
    "os"
    "os/user"
    "io/ioutil"
    "path/filepath"
    "strings"
    "strconv"
    "sync"
    "fmt"

    "text/template"
    "github.com/catalyst/gotiller/util"
    "github.com/catalyst/gotiller/log"
)

const (
    ConfigSuffix       = ".yaml"
    ConfigFname        = "common" + ConfigSuffix
    ConfigD            = "config.d"
    EnvironmentsSubdir = "environments"
    TemplatesSubdir    = "templates"
)

type Vars      map[string]string

type Target struct {
    Target string
    User   string
    Group  string
    Perms  os.FileMode
    Vars   Vars
}
func (t *Target) merge(targets ...*Target) {
    for _, target := range targets {
        if target.Target != "" {
            t.Target = target.Target
        }
        if target.User != "" {
            t.User = target.User
        }
        if target.Group != "" {
            t.Group = target.Group
        }
        if target.Perms != os.FileMode(0) {
            t.Perms = target.Perms
        }
        if target.Vars != nil {
            t.Vars = merge_vars(t.Vars, target.Vars)
        }
    }
}

type Templates map[string]*Target

type Config struct {
    Defaults           Templates
    DefaultVars        Vars      `yaml:"default_vars"`
    DefaultEnvironment string    `yaml:"default_environment"`
    EnvVarsPrefix      string    `yaml:"env_vars_prefix"`
    Environments       map[string]Templates
}
func (c *Config) merge(configs ...*Config) {
    for _, config := range configs {
        if config == nil {
            continue
        }

        c.Defaults = merge_templates(c.Defaults, config.Defaults)

        c.DefaultVars = merge_vars(c.DefaultVars, config.DefaultVars)

        if config.DefaultEnvironment != "" {
            c.DefaultEnvironment = config.DefaultEnvironment
        }

        if config.EnvVarsPrefix != "" {
            c.EnvVarsPrefix = config.EnvVarsPrefix
        }

        switch {
            case c.Environments == nil:
                c.Environments = config.Environments

            case config.Environments == nil:
                break

            default:
                for e, templates := range config.Environments {
                    r_templates, exists := c.Environments[e]
                    if exists {
                        c.Environments[e] = merge_templates(r_templates, templates)
                    } else {
                        c.Environments[e] = templates
                    }
                }
        }
    }
}

type GoTiller struct {
    Dir     string
    *Config
    EnvVars          Vars
    template.FuncMap
}
func (gt *GoTiller) init() {
    var config *Config

    config_path := filepath.Join(gt.Dir, ConfigFname)
    if _, err := os.Stat(config_path); err == nil {
        config = slurp_config(config_path)
    } else {
        config = &Config{
            Environments: make(map[string]Templates),
        }
    }

    configd_dir := filepath.Join(gt.Dir, ConfigD)
    if _, err := os.Stat(configd_dir); err == nil {
        dir_entries, err := ioutil.ReadDir(configd_dir)
        if err != nil {
            logger.Panic(err)
        }

        for _, entry := range dir_entries {
            configd_path := filepath.Join(configd_dir, entry.Name())
            config.merge(slurp_config(configd_path))
        }
    }

    environment_pattern := filepath.Join(gt.Dir, EnvironmentsSubdir, "*" + ConfigSuffix)
    if matches, _ := filepath.Glob(environment_pattern); matches != nil {
        for _, m := range matches {
            environment := strings.TrimSuffix(filepath.Base(m), ConfigSuffix)
            if _, exists := config.Environments[environment]; !exists {
                config.Environments[environment] = nil
            }
        }
    }
    if config.Environments == nil {
        logger.Panicf("No environments found in %s", gt.Dir)
    }

    if config.EnvVarsPrefix != "" {
        gt.EnvVars = extract_env_vars(config.EnvVarsPrefix)
    }

    gt.Config = config
}

var logger = log.New()

func New(dir string) *GoTiller {
    gt := &GoTiller{
        Dir: dir,
    }
    gt.init()
    return gt
}

func (gt *GoTiller) ListEnvironments() []string {
    var environments_s []string
    for e, _ := range gt.Environments {
        environments_s = append(environments_s, e)
    }
    return environments_s
}

func (gt *GoTiller) Templates(environment string) Templates {
    templates_m := gt.Config.Defaults

    if gt.Config.Environments != nil {
        if templates, exists := gt.Config.Environments[environment]; exists {
            templates_m = merge_templates(templates_m, templates)
        }
    }

    environment_config_path := filepath.Join(gt.Dir, EnvironmentsSubdir, environment + ConfigSuffix)
    if _, err := os.Stat(environment_config_path); err == nil {
        templates_m = merge_templates(templates_m, slurp_environment_config(environment_config_path))
    }

    return templates_m
}

func Execute(dir string, environment string, target_base_dir string, verbose bool) {
    logger.Printf("Executing from %s\n", dir)
    if target_base_dir != "" {
        logger.Printf("Writing to %s\n", target_base_dir)
    }

    if verbose {
        logger.SetDebug(true)
    }

    executor := New(dir)
    executor.Execute(environment, target_base_dir)
}

func (gt *GoTiller) Execute(environment string, target_base_dir string) {
    if environment == "" {
        if gt.Config.DefaultEnvironment != "" {
            environment = gt.Config.DefaultEnvironment
            logger.Println("Executing for default environment")
        } else {
            logger.Panicln("Environment not specified")
        }
    }

    logger.Printf("Executing for %s\n", environment)

    var wg sync.WaitGroup
    error_ch := make(chan string)
    var errs []string
    go func() {
        for err:= range error_ch {
            errs = append(errs, err)
        }
    }()

    for tpl, target := range gt.Templates(environment)  {
        wg.Add(1)
        // Need to pass params, cause loop params are volatile.
        go func(tpl string, target *Target) {
            defer func() {
                wg.Done()

                if r := recover(); r != nil {
                    error_ch <- fmt.Sprintf("%s", r)
                }
            }()

            gt.Deploy(tpl, target, target_base_dir)
        }(tpl, target)
    }
    wg.Wait()

    if errs != nil {
        logger.Panicf("%#v", errs)
    }
}

func (gt *GoTiller) Deploy(tpl string, target *Target, target_base_dir string) {
    template_path := filepath.Join(gt.Dir, TemplatesSubdir, tpl)
    vars := merge_vars(gt.Config.DefaultVars, target.Vars, gt.EnvVars)

    logger.Printf("%s -> %s\n", template_path, target.Target)
    // logger.Printf("%s -> %s Error: %s\n", template_path, target.Target, err)

    target_path := target.Target
    if target_path == "" {
        panic("No target for " + tpl)
    }
    if target_base_dir != "" {
        target_path = filepath.Join(target_base_dir, target_path)
    }

    dir, _ := filepath.Split(target_path)
    util.Mkdir(dir)

    out, err := os.Create(target_path)
    if err != nil {
        panic(err)
    }

    t := template.Must( template.New(tpl).Funcs(gt.FuncMap).ParseFiles(template_path) )
    if err := t.Execute(out, vars); err != nil {
        panic(err)
    }

    if target.Perms != os.FileMode(0) {
        if err := out.Chmod(target.Perms); err != nil {
            panic(err)
        }
    }

    if target.User != "" || target.Group != "" {
        var (
            u *user.User
            err error
        )
        if target.User == "" {
            u, err = user.Lookup(target.User)
        } else {
            u, err = user.Current()
        }
        if err != nil {
            panic(err)
        }
        uid_s, gid_s := u.Uid, u.Gid

        if target.Group != "" {
            g, err := user.LookupGroup(target.Group)
            if err != nil {
                panic(err)
            }
            gid_s = g.Gid
        }

        uid, err := strconv.Atoi(uid_s)
        if err != nil {
            panic(err)
        }
        gid, err := strconv.Atoi(gid_s)
        if err != nil {
            panic(err)
        }
        if err = out.Chown(uid, gid); err != nil {
            panic(err)
        }
    }

    if err := out.Close(); err != nil {
        panic(err)
    }
}

func extract_env_vars (prefix string) Vars {
    env_vars := make(Vars)
    for _, e := range os.Environ() {
        pair := strings.SplitN(e, "=", 2)
        if name := pair[0]; strings.HasPrefix(name, prefix) {
            env_vars[strings.TrimPrefix(name, prefix)] = pair[1]
        }
    }
    return env_vars
}

func slurp_config(path string) *Config {
    config := new(Config)

    util.ReadYaml(path, config)

    return config
}

func slurp_environment_config(path string) Templates {
    templates := make(Templates)

    util.ReadYaml(path, templates)

    return templates
}

func merge_templates(templates ...Templates) Templates {
    r_templates := make(Templates)

    for _, t := range templates {
        if t == nil {
            continue
        }

        for template, target := range t {
            if r_target, exists := r_templates[template]; exists {
                r_target.merge(target)
            } else {
                r_templates[template] = target
            }
        }
    }

    return r_templates
}

func merge_vars(vars ...Vars) Vars {
    r_vars := make(Vars)

    for _, v := range vars {
        if v == nil {
            continue
        }

        for k, val := range v {
            r_vars[k] = val
        }
    }

    return r_vars
}
