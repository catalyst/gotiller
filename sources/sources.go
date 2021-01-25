package sources

import (
    "os"
    "os/user"
    "sync"
    "path/filepath"
    "strconv"
    "sort"
    "fmt"
    "text/template"

    "github.com/catalyst/gotiller/util"
    "github.com/catalyst/gotiller/log"
)

var logger = log.DefaultLogger

var FuncMap = template.FuncMap{
    "sequence": util.Sequence,
}

type Vars      map[string]string
func (vs *Vars) Merge(vars ...Vars) {
    r_vars := *vs

    for _, v := range vars {
        if v == nil {
            continue
        }

        for k, val := range v {
            if ev, exists := r_vars[k]; exists {
                if val != ev {
                    logger.Debugf("Changing var %s to %s\n", k, val)
                    r_vars[k] = val
                }
            } else {
                logger.Debugf("Setting var %s to %s\n", k, val)
                r_vars[k] = val
            }
        }
    }

    *vs = r_vars
}
func (vs *Vars) SetMissing(vars ...Vars) {
    r_vars := *vs

    for _, v := range vars {
        if v == nil {
            continue
        }

        for k, val := range v {
            if _, exists := r_vars[k]; !exists {
                logger.Debugf("Setting missing var %s to %s\n", k, val)
                r_vars[k] = val
            }
        }
    }

    *vs = r_vars
}

func MakeVars(vs util.AnyMap) Vars {
    vs_v := make(Vars)
    for n, v := range vs {
        vs_v[n] = util.ToString(v)
    }
    return vs_v
}

type Spec struct {
    Target   string
    User     string
    Group    string
    Perms    os.FileMode
    Vars     Vars
}
func (s *Spec) Merge(s1 *Spec) {
    if s1.Target != "" && s1.Target != s.Target {
        logger.Debugf("Setting target to %s\n", s1.Target)
        s.Target = s1.Target
    }
    if s1.User != "" && s1.User != s.User {
        logger.Debugf("Setting target owner to user %s\n", s1.User)
        s.User = s1.User
    }
    if s1.Group != "" && s1.Group != s.Group {
        logger.Debugf("Setting target group to %s\n", s1.Group)
        s.Group = s1.Group
    }
    if s1.Perms != os.FileMode(0) && s1.Perms != s.Perms {
        logger.Debugf("Setting target permissions to %s\n", s1.Perms)
        s.Perms = s1.Perms
    }
    if s1.Vars != nil {
        if s.Vars == nil {
            s.Vars = make(Vars)
        }
        s.Vars.Merge(s1.Vars)
    }
}
func (s *Spec) Deploy(tpl_content string, base_dir string) {
    target_path := s.Target
    if target_path == "" {
        panic("No target")
    }
    if base_dir != "" {
        target_path = filepath.Join(base_dir, target_path)
    }

    logger.Printf("Writing %s\n", target_path)

    dir, _ := filepath.Split(target_path)
    util.Mkdir(dir)

    out, err := os.Create(target_path)
    if err != nil {
        panic(err)
    }

    t_exec := template.Must( template.New("").Funcs(FuncMap).Parse(tpl_content) )
    if err := t_exec.Execute(out, s.Vars); err != nil {
        panic(err)
    }

    if s.Perms != os.FileMode(0) {
        if err := out.Chmod(s.Perms); err != nil {
            panic(err)
        }
    }

    if s.User != "" || s.Group != "" {
        var (
            u *user.User
            err error
        )
        if s.User == "" {
            u, err = user.Current()
        } else {
            u, err = user.Lookup(s.User)
        }
        if err != nil {
            panic(err)
        }
        uid_s, gid_s := u.Uid, u.Gid

        if s.Group != "" {
            g, err := user.LookupGroup(s.Group)
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

func MakeSpec(m util.AnyMap) *Spec {
    d := Spec{
        Target:   util.ToString(m["target"]),
        User:     util.ToString(m["user"]),
        Group:    util.ToString(m["group"]),
    }
    if v, exists := m["perms"]; exists {
        d.Perms = os.FileMode(v.(int))
    }
    if v, exists := m["vars"]; exists {
        d.Vars = MakeVars(v.(util.AnyMap))
    }

    logger.Debugf("Made deployable %v\n", d)
    return &d
}

type Deployables map[string]*Spec
func (ds *Deployables) Merge(ds1 Deployables) {
    if ds1 == nil {
        return
    }

    r_deployables := *ds

    for d_name, spec := range ds1 {
        r_spec, exists := r_deployables[d_name]
        if exists {
            logger.Debugf("Merging deployable %s\n", d_name)
        } else {
            logger.Debugf("Setting deployable %s\n", d_name)
            r_spec = &Spec{}
            r_deployables[d_name] = r_spec

        }
        r_spec.Merge(spec)
    }

    *ds = r_deployables
}

func (ds *Deployables) MergeVars(vars Vars) {
    for tpl, d := range *ds {
        logger.Debugf("Merging missing vars into deployable %s\n", tpl)
        if d.Vars == nil {
            d.Vars = make(Vars)
        }
        d.Vars.Merge(vars)
    }
}

func (ds *Deployables) SetMissingVars(vars Vars) {
    for tpl, d := range *ds {
        logger.Debugf("Merging missing vars into deployable %s\n", tpl)
        if d.Vars == nil {
            d.Vars = make(Vars)
        }
        d.Vars.SetMissing(vars)
    }
}

func MakeDeployables(m util.AnyMap) Deployables {
    d := make(Deployables)
    for n, s := range m {
        logger.Debugf("Making deployable spec for %s\n", n)
        spec := MakeSpec(s.(util.AnyMap))
        d[n] = spec
    }
    return d
}

type Template struct {
    Path    string
    Content string
}
type Templates          map[string]*Template

type SourceInterface interface {
    MergeConfig(origin string, values interface{})
    DeployablesForEnvironment(environment string) Deployables
    DefaultVars()                                 Vars
    Template(string)                              string
    AllEnvironments()                             []string
    AllTemplates()                                Templates
}
type SourceFactory func () SourceInterface
type BaseSource struct {
    MergeHistory
}
func (s *BaseSource) DeployablesForEnvironment(environment string) Deployables { return nil }
func (s *BaseSource) DefaultVars              ()                   Vars        { return nil }
func (s *BaseSource) Template                 (n string)           string      { return "" }
func (s *BaseSource) AllEnvironments          ()                   []string    { return nil }
func (s *BaseSource) AllTemplates             ()                   Templates   { return nil }
func MakeBaseSource() BaseSource {
    return BaseSource{MergeHistory{}}
}

type MergeEvent struct {
    Origin string
    Loaded interface{}
}
type MergeHistory []*MergeEvent
func (h *MergeHistory) AddHistory(origin string, values interface{}) {
    r_h := *h
    r_h = append(r_h, &MergeEvent{origin, values})
    *h = r_h
}

type RegisteredSource struct {
    SourceFactory
    name          string
    order         int
}

type RegisteredSources map[string]*RegisteredSource
func (rs *RegisteredSources) Register(name string, factory SourceFactory, order int, force bool) {
    r_rs := *rs

    if _, exists := r_rs[name]; exists {
        if !force {
            logger.Panicf("Source %s already registered", name)
        }
        logger.Printf("Source %s is being replaced", name)
    }

    r_rs[name] = &RegisteredSource{factory, name, order}

    *rs = r_rs
}
func (rs *RegisteredSources) NewProcessor() *Processor {
    var sorted []*RegisteredSource

    for _, s := range *rs {
        sorted = append(sorted, s)
    }
    sort.SliceStable(sorted, func(i, j int) bool {
        return sorted[i].order < sorted[j].order
    })

    p := new(Processor)

    for _, s := range sorted {
        p.add(s.name, s.SourceFactory())
    }
    return p
}


type SourceInstance struct {
    Name   string
    SourceInterface
}
type Processor struct {
    DefaultEnvironment string
    Sources            []*SourceInstance
}
func (p *Processor) add(name string, s SourceInterface) {
    p.Sources = append(p.Sources, &SourceInstance{name, s})
}

func (p *Processor) Get(name string) SourceInterface {
    for _, si := range p.Sources {
        if si.Name == name {
            return si.SourceInterface
        }
    }

    return nil
}

func (p *Processor) MergeConfig(origin string, config util.AnyMap) {
    logger.Debugf("Merging %s\n", origin)
    for name, c := range config {
        switch name {
            case "default_environment":
                p.DefaultEnvironment = c.(string)
                logger.Debugf("Setting DefaultEnvironment to %s\n", p.DefaultEnvironment)
            default:
                si := p.Get(name)
                if si == nil {
                    logger.Printf("Source %s not registred", name)
                    break
                }
                si.MergeConfig(origin, c)
        }
    }
}

func (p *Processor) Deployables(environment string) Deployables {
    deployables_m := make(Deployables)
    vars_m := make(Vars)

    logger.Debugf("Getting deployables and default vars for %s\n", environment)
    for _, si := range p.Sources {
        logger.Debugf("From %s\n", si.Name)

        vars := si.DefaultVars()
        vars_m.Merge(vars)
        deployables_m.MergeVars(vars)

        ds := si.DeployablesForEnvironment(environment)
        deployables_m.Merge(ds)
    }
    logger.Debugln("Filling missing vars from defaults")
    deployables_m.SetMissingVars(vars_m)

    return deployables_m
}

func (p *Processor) ListTemplates() []map[string]string {
    tss := []map[string]string{}

    for _, si := range p.Sources {
        if si_ts :=  si.AllTemplates(); si_ts != nil {
            ts := make(map[string]string)
            for name, t := range si_ts {
                ts[name] = t.Path
            }
            tss = append(tss, ts)
        }
    }

    return tss
}

func (p *Processor) Template(name string) string {
    logger.Debugf("Getting template for %s\n", name)
    last_si := len(p.Sources) - 1
    for i := last_si; i >= 0; i-- {
        si := p.Sources[i]
        if t := si.Template(name); t != "" {
            return t
        }
    }
    return ""
}

func (p *Processor) ListEnvironments() []string {
    environments := make(map[string]bool)

    for _, si := range p.Sources {
        for _, e := range si.AllEnvironments() {
            environments[e] = true
        }
    }

    var environments_s []string
    for e, _ := range environments {
        environments_s = append(environments_s, e)
    }

    return environments_s
}

func (p *Processor) RunForEnvironment(environment string, target_base_dir string) {
    deployables := p.Deployables(environment)
    if len(deployables) == 0 {
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

    for n, s := range deployables {
        wg.Add(1)
        // Need to pass params, cause loop params are volatile.
        go func(name string, s *Spec) {
            defer func() {
                wg.Done()

                if r := recover(); r != nil {
                    error_ch <- fmt.Sprintf("%s", r)
                }
            }()

            tpl_content := p.Template(name)
            if tpl_content == "" {
                logger.Panicf("No template content for %s", name)
            }

            logger.Printf("Deploying %s\n", name)
            s.Deploy(tpl_content, target_base_dir)
        }(n, s)
    }
    wg.Wait()

    if errs != nil {
        logger.Panicf("%#v", errs)
    }
}

var registered_sources = make(RegisteredSources)

func RegisterSource(name string, factory SourceFactory, order int, force bool) {
    registered_sources.Register(name, factory, order, force)
}

func NewProcessor() *Processor {
    return registered_sources.NewProcessor()
}
