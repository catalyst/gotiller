package gotiller

import (
    "os"
    "os/user"
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
func (vs *Vars) merge(vars ...Vars) {
    r_vars := *vs

    for _, v := range vars {
        if v == nil {
            continue
        }

        for k, val := range v {
            if val != r_vars[k] {
                logger.Debugf("Setting var %s to %s\n", k, val)
                r_vars[k] = val
            }
        }
    }

    *vs = r_vars
}

type Target struct {
    Target string
    User   string
    Group  string
    Perms  os.FileMode
    Vars   Vars
}
func (t *Target) merge(targets ...*Target) {
    for _, target := range targets {
        if target.Target != "" && target.Target != t.Target {
            logger.Debugf("Setting target to %s\n", target.Target)
            t.Target = target.Target
        }
        if target.User != "" && target.User != t.User {
            logger.Debugf("Setting target owner to user %s\n", target.User)
            t.User = target.User
        }
        if target.Group != "" && target.Group != t.Group {
            logger.Debugf("Setting target group to %s\n", target.Group)
            t.Group = target.Group
        }
        if target.Perms != os.FileMode(0) && target.Perms != t.Perms {
            logger.Debugf("Setting target permissions to %s\n", target.Perms)
            t.Perms = target.Perms
        }
        if target.Vars != nil {
            if t.Vars == nil {
                t.Vars = make(Vars)
            }
            t.Vars.merge(target.Vars)
        }
    }
}

type Templates map[string]*Target
func (ts *Templates) merge(templates ...Templates) {
    r_templates := *ts

    for _, t := range templates {
        if t == nil {
            continue
        }

        for tpl, target := range t {
            if r_target, exists := r_templates[tpl]; exists {
                logger.Debugf("Merging template %s\n", tpl)
                r_target.merge(target)
            } else {
                logger.Debugf("Setting template %s\n", tpl)
                r_templates[tpl] = target
            }
        }
    }

    *ts = r_templates
}

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

        if c.Defaults == nil {
            c.Defaults = config.Defaults
        } else {
            logger.Debugln("Merging defaults")
            c.Defaults.merge(config.Defaults)
        }

        if c.DefaultVars == nil {
            c.DefaultVars = config.DefaultVars
        } else {
            logger.Debugln("Merging default_vars")
            c.DefaultVars.merge(config.DefaultVars)
        }

        if config.DefaultEnvironment != "" && config.DefaultEnvironment != c.DefaultEnvironment {
            logger.Printf("Setting default_environment to %s\n", config.DefaultEnvironment)
            c.DefaultEnvironment = config.DefaultEnvironment
        }

        if config.EnvVarsPrefix != "" && config.EnvVarsPrefix != c.EnvVarsPrefix {
            logger.Printf("Setting env_vars_prefix to %s\n", config.EnvVarsPrefix)
            c.EnvVarsPrefix = config.EnvVarsPrefix
        }

        switch {
            case c.Environments == nil:
                c.Environments = config.Environments

            case config.Environments == nil:
                break

            default:
                for e, templates := range config.Environments {
                    t := c.Environments[e]
                    logger.Debugf("Merging environment %s\n", e)
                    t.merge(templates)
                }
        }
    }
}

type TargetSource struct {
    Source string
    *Target
}
type TemplatesTargetChain map[string][]*TargetSource
func (tcp *TemplatesTargetChain) append(source string, templates Templates) {
    tc := *tcp
    for template, target := range templates {
        ts := TargetSource{source, target}
        tc[template] = append(tc[template], &ts)
    }
    *tcp = tc
}

type VarsSource struct {
    Source string
    Vars
}
type VarsChain []*VarsSource
func (vcp *VarsChain) append(source string, vars Vars) {
    vc := *vcp
    vs := VarsSource{source, vars}
    vc = append(vc, &vs)
    *vcp = vc
}

type GoTiller struct {
    Dir              string
    *Config
    EnvVars          Vars
    template.FuncMap
}
func (gt *GoTiller) init() {
    logger.Printf("GoTiller root: %s\n", gt.Dir)

    var config *Config
    config_path := filepath.Join(gt.Dir, ConfigFname)
    if _, err := os.Stat(config_path); err == nil {
        logger.Debugf("Reading main config %s\n", ConfigFname)
        config = slurp_config(config_path)
    } else {
        logger.Debugf("No main config %s\n", ConfigFname)
        config = new(Config)
    }

    config_pattern := filepath.Join(gt.Dir, ConfigD, "*" + ConfigSuffix)
    if matches, _ := filepath.Glob(config_pattern); matches != nil {
        logger.Debugf("Entering %s\n", ConfigD)
        for _, m := range matches {
            fname := strings.TrimSuffix(filepath.Base(m), ConfigSuffix)

            logger.Debugf("Merging config %s\n", fname)
            c := slurp_config(m)
            config.merge(c)
        }
    }

    gt.Config = config

    if config.EnvVarsPrefix != "" {
        gt.EnvVars = extract_env_vars(config.EnvVarsPrefix)
    }
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
    environments := make(map[string]bool)

    for e, _ := range gt.Environments {
        environments[e] = true
    }

    environment_pattern := filepath.Join(gt.Dir, EnvironmentsSubdir, "*" + ConfigSuffix)
    if matches, _ := filepath.Glob(environment_pattern); matches != nil {
        for _, m := range matches {
            e := strings.TrimSuffix(filepath.Base(m), ConfigSuffix)
            environments[e] = true
        }
    }

    var environments_s []string
    for e, _ := range environments {
        environments_s = append(environments_s, e)
    }

    return environments_s
}

func (gt *GoTiller) TemplatesChain(environment string) TemplatesTargetChain {
    tc := make(TemplatesTargetChain)

    if gt.Config.Defaults != nil {
        tc.append("config defaults", gt.Config.Defaults)
    }

    if gt.Config.Environments != nil {
        if templates, exists := gt.Config.Environments[environment]; exists {
            tc.append("config environments " + environment, templates)
        }
    }

    environment_config_path := filepath.Join(gt.Dir, EnvironmentsSubdir, environment + ConfigSuffix)
    if _, err := os.Stat(environment_config_path); err == nil {
        logger.Printf("Reading environment config for %s\n", environment)
        tc.append("environment " + environment, slurp_environment_config(environment_config_path))
    }

    return tc
}

func (gt *GoTiller) VarsForTarget(target *Target) Vars {
    vars := make(Vars)

    logger.Debugf("Getting vars for %s\n", target.Target)
    if gt.Config.DefaultVars != nil {
        logger.Debugln("Getting vars from default_vars")
        vars.merge(gt.Config.DefaultVars)
    }

    if target.Vars != nil {
        logger.Debugln("Getting vars from template")
        vars.merge(target.Vars)
    }

    if gt.EnvVars != nil {
        logger.Debugln("Getting vars from env")
        vars.merge(gt.EnvVars)
    }

    return vars
}

func (gt *GoTiller) DumpVarsChain(environment string, tpl string) VarsChain {
    var vc VarsChain

    if gt.Config.DefaultVars != nil {
        vc.append("config default_vars", gt.Config.DefaultVars)
    }

    if tc := gt.TemplatesChain(environment)[tpl]; tc != nil {
        for _, ts := range tc {
            vc.append(ts.Source, ts.Target.Vars)
        }
    }

    if gt.EnvVars != nil {
        vc.append("env", gt.EnvVars)
    }

    return vc
}

func (gt *GoTiller) Templates(environment string) Templates {
    templates_m := make(Templates)

    logger.Debugf("Getting templates for %s\n", environment)
    for tpl, tc := range gt.TemplatesChain(environment) {
        t := new(Target)

        logger.Debugf("Merging for template %s\n", tpl)
        for _, ts := range tc {
            t.merge(ts.Target)
        }
        templates_m[tpl] = t
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
            if gt.Config.Defaults == nil {
                logger.Panicln("Environment not specified and there is no defaults")
            }
        }
    }

    logger.Printf("Executing for %s\n", environment)

    templates := gt.Templates(environment)
    if len(templates) == 0 {
        logger.Panicln("Nothing to do")
    }

    var wg sync.WaitGroup
    error_ch := make(chan string)
    var errs []string
    go func() {
        for err:= range error_ch {
            errs = append(errs, err)
        }
    }()

    for tpl, target := range templates {
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
    vars := gt.VarsForTarget(target)

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
            u, err = user.Current()
        } else {
            u, err = user.Lookup(target.User)
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
