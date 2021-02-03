package sources

import (
    "io"
    "os"
    "os/user"
    "sync"
    "path/filepath"
    "sort"
    "fmt"
    "encoding/base64"
    "text/template"

    "github.com/catalyst/gotiller/util"
    "github.com/catalyst/gotiller/log"
)

const GlobalVarsKey = "_vars"

var logger = log.DefaultLogger

type Vars      map[string]string
func (vs Vars) Merge(vars ...Vars) {
    for _, v := range vars {
        if v == nil {
            continue
        }

        for k, val := range v {
            if ev, exists := vs[k]; exists {
                if val != ev {
                    logger.Debugf("Changing var %s to %s\n", k, val)
                    vs[k] = val
                }
            } else {
                logger.Debugf("Setting var %s to %s\n", k, val)
                vs[k] = val
            }
        }
    }
}
func (vs Vars) SetMissing(vars ...Vars) {
    for _, v := range vars {
        if v == nil {
            continue
        }

        for k, val := range v {
            if _, exists := vs[k]; !exists {
                logger.Debugf("Setting missing var %s to %s\n", k, val)
                vs[k] = val
            }
        }
    }
}
func (vs Vars) Clone() Vars {
    vs_v := make(Vars)
    for n, v := range vs {
        vs_v[n] = v
    }
    return vs_v
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
func (s *Spec) Deploy(t *Template, base_dir string) {
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

    t.Write(out, s.Vars)

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

        uid := util.AtoI(uid_s)
        gid := util.AtoI(gid_s)
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

type Specs map[string]*Spec

type Deployables struct {
    Vars
    Specs
}
func (ds *Deployables) Merge(ds1 *Deployables) {
    if ds1 == nil {
        return
    }

    if ds1.Vars != nil {
        if ds.Vars == nil {
            ds.Vars = make(Vars)
        }
        ds.Vars.Merge(ds1.Vars)
    }

    for name, spec := range ds1.Specs {
        r_spec, exists := ds.Specs[name]
        if exists {
            logger.Debugf("Merging spec %s\n", name)
        } else {
            logger.Debugf("Setting spec %s\n", name)
            r_spec = &Spec{}
            ds.Specs[name] = r_spec
        }
        r_spec.Merge(spec)
    }
}
func (ds *Deployables) Overlay(ds1 *Deployables) {
    if ds1.Vars != nil {
        for _, spec := range ds.Specs {
            if spec.Vars == nil {
                spec.Vars = make(Vars)
            }
            spec.Vars.Merge(ds1.Vars)
        }
    }

    ds.Merge(ds1)
}
func (ds *Deployables) PreparedSpecs() Specs {
    specs := make(Specs)
    for name, spec := range ds.Specs {
        s := *spec

        logger.Debugf("Merging missing vars into spec %s\n", name)
        if s.Vars == nil {
            s.Vars = make(Vars)
        }
        s.Vars.SetMissing(ds.Vars)

        specs[name] = &s
    }
    return specs
}

func MakeDeployables(m util.AnyMap) *Deployables {
    var vars Vars
    specs :=  make(Specs)
    for n, s := range m {
        if n == GlobalVarsKey {
            logger.Debugf("Making vars for %s\n", n)
            vars = MakeVars(s.(util.AnyMap))
            continue
        }

        logger.Debugf("Making deployable spec for %s\n", n)
        specs[n] = MakeSpec(s.(util.AnyMap))
    }
    return &Deployables{vars, specs}
}

var FuncMap = template.FuncMap{
    "iadd"       : func(x, y int) int { return x + y  },
    "imul"       : func(x, y int) int { return x * y  },
    "idiv"       : func(x, y int) int { return x / y  },
    "imod"       : func(x, m int) int { return x % m  },
    "tostr"      : util.ToString,
    "strtoi"     : util.AtoI,
    "safe"       : util.SafeValue,
    "coalesce"   : util.Coalesce,
    "tolower"    : util.SafeToLower,
    "match"      : util.SafeMatch,
    "regexrepl"  : util.SafeReplaceAllString,
    "quotedlist" : util.QuotedList,
    "sequence"   : util.Sequence,
    "timeoffset" : util.TimeOffset,
    "isfile"     : util.IsFile,
    "decode64"   : func(in string) string {
        data, err := base64.StdEncoding.DecodeString(in)
        if err != nil {
            panic(err)
        }
        return string(data)
    },
}
func RegisterTemplateFunc(name string, fn interface{}) {
    FuncMap[name] = fn
}
func CloneFuncMap() template.FuncMap {
    func_map := make(template.FuncMap)
    for n, f := range FuncMap {
        func_map[n] = f
    }
    return func_map
}

type Template struct {
    Path    string
    Content string
}
func (t *Template) Write(out io.Writer, v Vars) {
    func_map := CloneFuncMap()
    func_map["val"] = func(var_name string) string { return v[var_name] }

    t_exec := template.Must( template.New("").Funcs(func_map).Parse(t.Content) )
    if err := t_exec.Execute(out, v); err != nil {
        panic(err)
    }
}

type Templates          map[string]*Template

type SourceInterface interface {
    MergeConfig(origin string, values interface{})
    DeployablesForEnvironment(environment string)  *Deployables
    Template(string)                               *Template
    AllEnvironments()                              []string
    AllTemplates()                                 Templates
}
type SourceFactory func () SourceInterface
type BaseSource struct {
    MergeHistory
}
func (s *BaseSource) DeployablesForEnvironment(environment string) *Deployables { return nil }
func (s *BaseSource) Template                 (n string)           *Template    { return nil }
func (s *BaseSource) AllEnvironments          ()                   []string     { return nil }
func (s *BaseSource) AllTemplates             ()                   Templates    { return nil }
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

func (p *Processor) Specs(environment string) Specs {
    deployables := &Deployables{nil, make(Specs)}

    logger.Debugf("Getting deployables and default vars for %s\n", environment)
    for _, si := range p.Sources {
        logger.Debugf("From %s\n", si.Name)

        d := si.DeployablesForEnvironment(environment)
        if d != nil {
            deployables.Overlay(d)
        }
    }
    logger.Debugln("Filling missing vars from defaults")

    return deployables.PreparedSpecs()
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

func (p *Processor) Template(name string) *Template {
    logger.Debugf("Getting template for %s\n", name)
    last_si := len(p.Sources) - 1
    for i := last_si; i >= 0; i-- {
        si := p.Sources[i]
        if t := si.Template(name); t != nil {
            return t
        }
    }
    return nil
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
    specs := p.Specs(environment)
    if len(specs) == 0 {
        if environment == "" {
            logger.Panic("No environment specified - nothing to do")
        }
        logger.Panicf("Nothing to do for environment %s", environment)
    }

    var wg sync.WaitGroup
    error_ch := make(chan string)
    var errs []string
    go func() {
        for err:= range error_ch {
            errs = append(errs, err)
        }
    }()

    for n, s := range specs {
        if _, exists := s.Vars["environment"]; !exists {
            s.Vars["environment"] = environment
        }

        wg.Add(1)
        // Need to pass params, cause loop params are volatile.
        go func(name string, s *Spec) {
            defer func() {
                wg.Done()

                if r := recover(); r != nil {
                    error_ch <- fmt.Sprintf("%s", r)
                }
            }()

            t := p.Template(name)
            if t == nil {
                logger.Panicf("No template for %s", name)
            }

            logger.Printf("Deploying %s\n", name)
            s.Deploy(t, target_base_dir)
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
