package convert

import (
    "log"
    "os"
    "path/filepath"
    "strings"
    "regexp"
    "text/template"

    "github.com/catalyst/gotiller/util"
    "github.com/catalyst/gotiller/sources"
)

const ConfigEtcPath = "/etc/tiller"

type RenamedTemplates = map[string]string

type Converter struct {
    SourceDir        string
    TargetDir        string
    StripVarPrefix   string
    RenamedTemplates
}
func (c *Converter) init() {
    c.RenamedTemplates = make(RenamedTemplates)

    /*
    This is a hack. We assume that for the purposes of target matching
    new format is close enough to the old format.
    Proper thing would be to convert without template renaming first,
    then rename. But this is good enough.
    */
    processor := sources.LoadConfigsFromDir(c.SourceDir)
    environments := processor.ListEnvironments()
    for _, e := range environments {
        for name, s := range processor.Specs(e) {
            if t := s.Target; t != "" {
                c.RenameTemplate(name, t)
            }
        }
    }
}

func (c *Converter) RenameTemplate(t string, target string) {
    if _, exists := c.RenamedTemplates[t]; exists {
        return
    }

    //give template a better name
    new_t := strings.TrimSuffix(t, ".erb")
    if !strings.ContainsRune(new_t, '.') {
        if target == "" {
            return
        }

        if ext := filepath.Ext(target); ext != "" {
            new_t += ext
        }
    }

    if new_t != t {
        c.RenamedTemplates[t] = new_t
    }
}

func (c *Converter) RenamedTemplate(t string) string {
    if new_t, exists := c.RenamedTemplates[t]; exists {
        return new_t
    }
    return t
}

func NewConverter(in_dir string, out_dir string) *Converter {
    c := &Converter{
        SourceDir     : in_dir,
        TargetDir     : out_dir,
        StripVarPrefix: "env_",
    }
    c.init()
    return c
}

func (c *Converter) Convert() {
    c.ConvertMainConfig()
    c.ConvertConfigD()
    c.ConvertEnvironments()
    c.ConvertTemplates()
}

func Convert(in_dir string, out_dir string) {
    if (out_dir == "") {
        panic("output gotiller config dir not given")
    }
    finfo, err := os.Stat(out_dir)
    if err == nil {
        if !finfo.IsDir() {
            panic(out_dir + " is not a dir")
        }
    } else {
        util.Mkdir(out_dir)
    }

    if in_dir == "" {
        if _, err := os.Stat(sources.ConfigFname); err == nil {
            in_dir = "."
        } else {
            in_dir = ConfigEtcPath
        }
    }

    converter := NewConverter(in_dir, out_dir)
    converter.Convert()
}

func (c *Converter) ConvertMainConfig() {
    tiller_config_path := filepath.Join(c.SourceDir, sources.ConfigFname)

    if _, err := os.Stat(tiller_config_path); err == nil {
        converted_config_path := filepath.Join(c.TargetDir, sources.ConfigFname)

        c.convert_config_file(tiller_config_path, converted_config_path)
    }
}

func (c *Converter) ConvertConfigD() {
    tiller_config_subdir := filepath.Join(c.SourceDir, sources.ConfigD)

    if _, err := os.Stat(tiller_config_subdir); err == nil {
        converted_config_subdir := filepath.Join(c.TargetDir, sources.ConfigD)
        util.Mkdir(converted_config_subdir)

        for _, entry := range util.ReadDir(tiller_config_subdir) {
            t := entry.Name()

            tiller_config_path := filepath.Join(tiller_config_subdir, t)
            converted_config_path := filepath.Join(converted_config_subdir, t)
            c.convert_config_file(tiller_config_path, converted_config_path)
        }
    }
}

func (c *Converter) convert_config_file(in_path string, out_path string) {
    config := make(util.AnyMap)
    util.ReadYaml(in_path, config)

    c.convert_config(config)
    util.WriteYaml(out_path, config)
}

func (c *Converter) convert_config(config util.AnyMap) {
    var default_vars util.AnyMap

    for k, v := range config {
        switch k {
            // unchanged
            case "default_environment":
                break

            // implicit
            case "data_sources", "template_sources":
                delete(config, k)

            case "defaults":
                if g, exists := v.(util.AnyMap)["global"]; exists {
                    // defaults.global -> default_vars
                    default_vars = c.convert_vars(g.(util.AnyMap))
                    delete(v.(util.AnyMap), "global")
                }

                if v != nil {
                    config["defaults"] = c.convert_environment(v.(util.AnyMap))
                }

            case "environment":
                if prefix , exists := v.(util.AnyMap)["prefix"]; exists {
                    c.StripVarPrefix = prefix.(string)
                }

            case "environments":
                for e, templates := range v.(util.AnyMap) {
                    if templates != nil {
                        v.(util.AnyMap)[e] = c.convert_environment(templates.(util.AnyMap))
                    }
                }

            default:
                log.Printf("%s not supported, leaving untouched", k)
        }
    }

    if default_vars != nil {
        config["default_vars"] = default_vars
    }
}

func (c *Converter) ConvertEnvironments() {
    tiller_environments_subdir := filepath.Join(c.SourceDir, sources.EnvironmentsSubdir)

    if _, err := os.Stat(tiller_environments_subdir); err == nil {
        converted_environments_subdir := filepath.Join(c.TargetDir, sources.EnvironmentsSubdir)
        util.Mkdir(converted_environments_subdir)

        for _, entry := range util.ReadDir(tiller_environments_subdir) {
            t := entry.Name()

            tiller_config_path := filepath.Join(tiller_environments_subdir, t)
            converted_config_path := filepath.Join(converted_environments_subdir, t)
            c.convert_environment_config_file(tiller_config_path, converted_config_path)
        }
    }
}

func (c *Converter) convert_environment_config_file(in_path string, out_path string) {
    config := make(util.AnyMap)
    util.ReadYaml(in_path, config)

    c.convert_environment(config)
    util.WriteYaml(out_path, config)
}

func (c *Converter) convert_environment(templates util.AnyMap) util.AnyMap {
    converted := make(util.AnyMap)

    for t, target := range templates {
        new_t := c.RenamedTemplate(t)

        c.convert_target(target.(util.AnyMap))
        converted[new_t] = target
    }

    return converted
}

func (c *Converter) convert_target(target util.AnyMap) {
    for k, v := range target {
        switch k {
            case "config":  // config -> vars
                target["vars"] = c.convert_vars(v.(util.AnyMap))
                delete(target, k)
        }
    }
}

func (c *Converter) convert_vars(vars util.AnyMap) util.AnyMap {
    new_vars := make(util.AnyMap)
    for v, val := range vars {
        new_vars[strings.TrimPrefix(v, c.StripVarPrefix)] = val
    }
    return new_vars
}

func (c *Converter) ConvertTemplates() {
    tiller_templates_subdir := filepath.Join(c.SourceDir, sources.TemplatesSubdir)
    converted_templates_subdir := filepath.Join(c.TargetDir, sources.TemplatesSubdir)
    util.Mkdir(converted_templates_subdir)

    for _, entry := range util.ReadDir(tiller_templates_subdir) {
        t := entry.Name()
        tiller_template_path := filepath.Join(tiller_templates_subdir, t)

        new_t := c.RenamedTemplate(t)
        converted_template_path := filepath.Join(converted_templates_subdir, new_t)

        content := util.SlurpFile(tiller_template_path)
        converted := c.convert_template(string(content))
        util.WriteFile(converted_template_path, []byte(converted))

        if _, err := template.New("").Parse(converted); err != nil {
            log.Printf(`Template "%s" not converted cleanly: %s`, converted_template_path, err)
        }
    }
}

func (c *Converter) convert_template(content string) string {
    return tag_re.ReplaceAllStringFunc(content, func (tag string) string {
        return c.convert_tag(tag)
    })
}

var (
    tag_re = regexp.MustCompile(`<%(=|-)?\s*(#\s*)?([^%]*?)\s*(-)?%>`)
    conditional_op_re = regexp.MustCompile(`^([^?]+?)\s+\?\s+(.*?)\s+:\s+(.*)?`)
)
func (c *Converter) convert_tag(tag string) string {
    m := tag_re.FindStringSubmatch(tag)
    leader := m[1]
    comment := m[2]
    content := m[3]
    end_strip := m[4]

    var front_strip, converted string
    switch leader {
        case "=":
            // we can only try ?: conversion for straight output
            if m := conditional_op_re.FindStringSubmatch(content); m != nil {
                cond, if_true, if_false := m[1], m[2], m[3]
                return c.convert_conditional(cond, if_true, if_false, front_strip, end_strip)
            }

            converted, _ = c.convert_exp(content)
        default:
            if comment == "" {
                front_strip = leader
                converted = c.convert_statement(content)
            } else {
                front_strip, end_strip = "-", "-"
                converted = "/* " + content + " */"
            }
    }

    return join_exps(" ", "{{" + front_strip, converted, end_strip + "}}")
}

func (c *Converter) convert_conditional(
    cond string, if_true string, if_false string,
    front_strip string, end_strip string,
) string {
    converted_cond := c.convert_statement("if " + cond)
    converted_true, is_var := c.convert_exp(if_true)
    if !is_var {
        converted_true = enclose_in_brackets(converted_true)
    }
    converted_false, is_var := c.convert_exp(if_true)
    if !is_var {
        converted_false = enclose_in_brackets(converted_false)
    }

    return join_exps(" ",
        "{{" + front_strip, converted_cond, "}}{{",
        converted_true,
        "}}{{ else }}{{",
        converted_false,
        "}}{{ end", end_strip + "}}",
    )
}

var statement_re = regexp.MustCompile(`^(\w+)\b\s*(.*)$`)
func (c *Converter) convert_statement(statement string) string {
    if m := statement_re.FindStringSubmatch(statement); m != nil {
        stmt, exp := m[1], m[2]

        switch stmt {
            case "end", "else":
                if exp != "" {
                    log.Printf(`Content after "%s", discarding: %s`, stmt, exp)
                }
                return stmt
            case "if":
                converted_exp, _ := c.convert_exp(exp)
                return join_exps(" ", stmt, converted_exp)
        }
    }

    log.Printf("Don't know what to do with %s, leaving unchanged", statement)
    return statement

}

const (
    not_op_re            = `(!|not)`
    fn__re               = `(defined\?)`
    var_re               = `[A-Za-z][\w.]*`
    number_re            = `\d*(\.\d+)?`
    string_re            = `["']([^"']*)["']`
    not_comparison_op_re = `[^=!<>]`
)
var (
    tiller_op_to_fn = map[string]string {
        "!"       : "not",
        "not"     : "not",
        "or"      : "or",
        "and"     : "and",
        "||"      : "or",
        "&&"      : "and",
        "=="      : "eq",
        "==="     : "eq",
        "!="      : "ne",
        "!=="     : "ne",
        ">="      : "ge",
        ">"       : "gt",
        "<="      : "le",
        "<"       : "lt",
    }
    tiller_fn_to_fn = map[string]string {
        "defined?": "",
    }

    bracket_re        = regexp.MustCompile(`^` + not_op_re + `?` + fn__re + `?` + `?\s*\(\s*(.*)\s*\)$`)
    unary_exp_re      = regexp.MustCompile(`^` + not_op_re + `\s*(` + var_re + `)$`)
    var_exp_re        = regexp.MustCompile(`^` + var_re + `$`)
    number_exp_re     = regexp.MustCompile(`^` + number_re + `$`)
    string_exp_re     = regexp.MustCompile(`^` + string_re + `$`)
    comparison_op_re  = regexp.MustCompile(
        `^(` + not_comparison_op_re + `+?)\s*` +
        `(===?|!==?|>|>=|<|<=)` +
        `\s*(` + not_comparison_op_re + `+)$`,
    )

    logical_ops  = []struct{op string; re *regexp.Regexp}{
        {"or" , regexp.MustCompile(`\s+or\s+`)  },
        {"and", regexp.MustCompile(`\s+and\s+`) },
        {"||" , regexp.MustCompile(`\s*\|\|\s*`)},
        {"&&" , regexp.MustCompile(`\s*&&\s*`)  },
    }
)

func (c *Converter) convert_exp(exp string) (string, bool) {
    if m := bracket_re.FindStringSubmatch(exp); m != nil {
        not_op, fn, subexp := m[1], m[2], m[3]
        if !has_unbalanced_bracket(subexp) {
            return c.convert_unary_exp(not_op, fn, subexp)
        }
    }

    var split_attempted []string

    OpLoop:
    for _, op_struct := range logical_ops {
        op, tiller_re := op_struct.op, op_struct.re
        if s := tiller_re.Split(exp, -1); len(s) > 1 {
            split_attempted = append(split_attempted, op)
            var s_converted_exp []string
            for _, subexp := range s {
                if has_unbalanced_bracket(subexp) {
                    continue OpLoop
                }
                converted_exp, is_var := c.convert_exp(subexp)
                if !is_var {
                    converted_exp = enclose_in_brackets(converted_exp)
                }
                s_converted_exp = append(s_converted_exp, converted_exp)
            }

            return join_exps(" ", append([]string{tiller_op_to_fn[op]}, s_converted_exp...)...), false
        }
    }

    if m := comparison_op_re.FindStringSubmatch(exp); m != nil {
        left, op, right := m[1], m[2], m[3]
        split_attempted = append(split_attempted, op)
        if !has_unbalanced_bracket(left) &&  !has_unbalanced_bracket(right) {
            converted_left, is_var := c.convert_exp(left)
            if !is_var {
                converted_left = enclose_in_brackets(converted_left)
            }

            converted_right, is_var := c.convert_exp(right)
            if !is_var {
                converted_right = enclose_in_brackets(converted_right)
            }

            return join_exps(" ", tiller_op_to_fn[op], converted_left, converted_right), false
        }
    }

    if split_attempted != nil {
        log.Printf(`Can't split on %s, leaving unchanged: %s`, split_attempted, exp)
        return exp, false
    }

    if m := unary_exp_re.FindStringSubmatch(exp); m != nil {
        return c.convert_unary_exp(m[1], "", m[2])
    }

    if number_exp_re.MatchString(exp) {
        return exp, true
    }

    if string_exp_re.MatchString(exp) {
        return exp, true  // XXX convert quotes
    }

    if var_exp_re.MatchString(exp) {
        exp = strings.TrimPrefix(exp, c.StripVarPrefix)

        return join_exps("", ".", exp), true
    }

    log.Printf(`Don't know what to do with "%s", leaving unchanged`, exp)
    return exp, false
}

func (c *Converter) convert_unary_exp(not_op string, fn string, exp string) (string, bool) {
    converted_exp, is_var := c.convert_exp(exp)

    var go_fn string
    if fn != "" {
        if go_fn = tiller_fn_to_fn[fn]; go_fn != "" {
            converted_exp = join_exps("", go_fn, enclose_in_brackets(converted_exp))
            // keep is_var, it is fn(blah) so behaves like a single var
        }
    }

    if not_op != "" {
        not_fn := tiller_op_to_fn[not_op]

        if is_var {
            is_var = false
        } else {
            converted_exp = enclose_in_brackets(converted_exp)
            is_var = true
        }
        converted_exp = join_exps(" ", not_fn, converted_exp)
    }

    return converted_exp, is_var
}

func join_exps(join string, exp ...string) string {
    return strings.Join(exp, join)
}

func enclose_in_brackets(exp string) string {
    return join_exps("", "(", exp, ")")
}

var unbalanced_bracket_re = regexp.MustCompile(`^[^(]*\)`)
func has_unbalanced_bracket(exp string) bool {
    return unbalanced_bracket_re.MatchString(exp)
}
