package convert

import (
    "log"
    "os"
    "io/ioutil"
    "path/filepath"
    "strings"
    "regexp"
    "text/template"

    "github.com/catalyst/gotiller"
    "github.com/catalyst/gotiller/util"
)

const ConfigEtcPath = "/etc/tiller"

type AnyMap = map[string]interface{}
type RenamedTemplates = map[string]string

type Converter struct {
    SourceDir         string
    TargetDir         string
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
    gt := gotiller.New(c.SourceDir)
    for _, e := range gt.Environments() {
        for template, target := range gt.Templates(e) {
            c.RenameTemplate(template, target.Target)
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

func FromTiller(in_dir string, out_dir string) {
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
        if _, err := os.Stat(gotiller.ConfigFname); err == nil {
            in_dir = "."
        } else {
            in_dir = ConfigEtcPath
        }
    }

    converter := NewConverter(in_dir, out_dir)
    converter.Convert()
}

func NewConverter(in_dir string, out_dir string) *Converter {
    c := &Converter{
        SourceDir: in_dir,
        TargetDir: out_dir,
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

func (c *Converter) ConvertMainConfig() {
    tiller_config_path := filepath.Join(c.SourceDir, gotiller.ConfigFname)
    converted_config_path := filepath.Join(c.TargetDir, gotiller.ConfigFname)

    c.convert_config_file(tiller_config_path, converted_config_path)
}

func (c *Converter) ConvertConfigD() {
    c.ConvertConfigSubdir(gotiller.ConfigD)
}

func (c *Converter) ConvertEnvironments() {
    c.ConvertConfigSubdir(gotiller.EnvironmentsSubdir)
}

func (c *Converter) ConvertConfigSubdir(subdir string) {
    tiller_config_subdir := filepath.Join(c.SourceDir, subdir)
    converted_config_subdir := filepath.Join(c.TargetDir, subdir)

    util.Mkdir(converted_config_subdir)

    if _, err := os.Stat(tiller_config_subdir); err == nil {
        dir_entries, err := ioutil.ReadDir(tiller_config_subdir)
        if err != nil {
            log.Panic(err)
        }

        for _, entry := range dir_entries {
            tiller_config_path := filepath.Join(tiller_config_subdir, entry.Name())
            converted_config_path := filepath.Join(converted_config_subdir, entry.Name())
            c.convert_config_file(tiller_config_path, converted_config_path)
        }
    }
}

func (c *Converter) convert_config_file(in_path string, out_path string) {
    config := make(AnyMap)
    util.ReadYaml(in_path, config)

    c.convert_config(config)
    util.WriteYaml(out_path, config)
}

func (c *Converter) convert_config(config AnyMap) {
    var default_vars AnyMap

    for k, v := range config {
        switch k {
            // unchanged
            case "default_environment":
                break

            // implicit
            case "data_sources", "template_sources":
                delete(config, k)

            case "defaults":
                if g, exists := v.(AnyMap)["global"]; exists {
                    default_vars = g.(AnyMap)  // defaults.global -> default_vars
                    delete(v.(AnyMap), "global")
                }

                if v != nil {
                    config["defaults"] = c.convert_environment(v.(AnyMap))
                }

            case "environments":
                for e, templates := range v.(AnyMap) {
                    if templates != nil {
                        v.(AnyMap)[e] = c.convert_environment(templates.(AnyMap))
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

func (c *Converter) convert_environment(templates AnyMap) AnyMap {
    converted := make(AnyMap)

    for t, target := range templates {
        new_t := c.RenamedTemplate(t)

        convert_target(target.(AnyMap))
        converted[new_t] = target
    }

    return converted
}

func convert_target(target AnyMap) {
    for k, v := range target {
        switch k {
            case "config":  // config -> vars
                target["vars"] = v
                delete(target, k)
        }
    }
}

func (c *Converter) ConvertTemplates() {
    tiller_templates_subdir := filepath.Join(c.SourceDir, gotiller.TemplatesSubdir)
    converted_templates_subdir := filepath.Join(c.TargetDir, gotiller.TemplatesSubdir)
    util.Mkdir(converted_templates_subdir)

    dir_entries, err := ioutil.ReadDir(tiller_templates_subdir)
    if err != nil {
        log.Panic(err)
    }

    for _, entry := range dir_entries {
        t := entry.Name()
        tiller_template_path := filepath.Join(tiller_templates_subdir, t)

        new_t := c.RenamedTemplate(t)
        converted_template_path := filepath.Join(converted_templates_subdir, new_t)

        content := util.SlurpFile(tiller_template_path)
        converted := convert_template(string(content))
        util.WriteFile(converted_template_path, []byte(converted))

        if _, err := template.New("").Parse(converted); err != nil {
            log.Printf(`Template "%s" not converted cleanly: %s`, converted_template_path, err)
        }
    }
}

func convert_template(content string) string {
    return tag_re.ReplaceAllStringFunc(content, tag_fn)
}

var tag_re = regexp.MustCompile(`<%(#|=|-)?\s*([^%]*?)\s*(-)?%>`)
func tag_fn(tag string) string {
    m := tag_re.FindStringSubmatch(tag)
    leader := m[1]
    content := m[2]
    end_strip := m[3]

    var front_strip, converted string
    switch leader {
        case "#":
            front_strip, end_strip = "-", "-"
            content = "/* " + content + " */"
        case "=":
            // we can only try ?: conversion for straight output
            if m := conditional_op_re.FindStringSubmatch(content); m != nil {
                cond, if_true, if_false := m[1], m[2], m[3]
                return convert_conditional(cond, if_true, if_false, front_strip, end_strip)
            }

            converted, _ = convert_exp(content, true)
        default:
            front_strip = leader
            converted = convert_statement(content)
    }

    return join_exps(" ", "{{" + front_strip, converted, end_strip + "}}")
}

var conditional_op_re = regexp.MustCompile(`^([^?]+?)\s+\?\s+(.*?)\s+:\s+(.*)?`)
func convert_conditional(cond string, if_true string, if_false string, front_strip string, end_strip string) string {
    converted_cond  := convert_statement("if " + cond)
    converted_true,  _ := convert_exp(if_true, true)
    converted_false, _ := convert_exp(if_true, true)

    return join_exps(" ",
        "{{" + front_strip, converted_cond, "}}{{",
        converted_true,
        "}}{{ else }}{{",
        converted_false,
        "}}{{ end", end_strip + "}}",
    )
}

var statement_re = regexp.MustCompile(`^(\w+)\b\s*(.*)$`)
func convert_statement(statement string) string {
    if m := statement_re.FindStringSubmatch(statement); m != nil {
        stmt, exp := m[1], m[2]

        switch stmt {
            case "end", "else":
                if exp != "" {
                    log.Printf(`Content after "%s", discarding: %s`, stmt, exp)
                }
                return stmt
            case "if":
                converted_exp, _ := convert_exp(exp, true)
                return join_exps(" ", stmt, converted_exp)
        }
    }

    log.Printf("Don't know what to do with %s, leaving unchanged", statement)
    return statement

}

const (
    unary_op_re          = `(!|not|defined\?)`
    var_re               = `[\w.]+`
    not_comparison_op_re = `[^=!<>]`
)
var (
    tiller_op_to_fn = map[string]string {
        "!"       : "not",
        "not"     : "not",
        "defined?": "",
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

    bracket_re        = regexp.MustCompile(`^` + unary_op_re + `?\s*\(\s*(.*)\s*\)$`)
    unary_exp_re      = regexp.MustCompile(`^` + unary_op_re + `\s*(` + var_re + `)$`)
    var_exp_re        = regexp.MustCompile(`^` + var_re + `$`)
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

func convert_exp(exp string, no_enclosing_brackets bool) (string, bool) {
    if m := bracket_re.FindStringSubmatch(exp); m != nil {
        op, subexp := m[1], m[2]
        if !has_unbalanced_bracket(subexp) {
            return convert_unary_exp(op, subexp, no_enclosing_brackets)
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
                converted_exp, is_var := convert_exp(subexp, true)
                if !is_var {
                    converted_exp = enclose_in_brackets(converted_exp)
                }
                s_converted_exp = append(s_converted_exp, converted_exp)
            }

            converted_exp := join_exps(" ", append([]string{tiller_op_to_fn[op]}, s_converted_exp...)...)
            if no_enclosing_brackets {
                return converted_exp, false
            }
            return enclose_in_brackets(converted_exp), false
        }
    }

    if m := comparison_op_re.FindStringSubmatch(exp); m != nil {
        left, op, right := m[1], m[2], m[3]
        split_attempted = append(split_attempted, op)
        if !has_unbalanced_bracket(left) &&  !has_unbalanced_bracket(right) {
            converted_left, is_var := convert_exp(left, true)
            if !is_var {
                converted_left = enclose_in_brackets(converted_left)
            }

            converted_right, is_var := convert_exp(right, true)
            if !is_var {
                converted_right = enclose_in_brackets(converted_right)
            }

            converted_exp := join_exps(" ", tiller_op_to_fn[op], converted_left, converted_right)
            if no_enclosing_brackets {
                return converted_exp, false
            }
            return enclose_in_brackets(converted_exp), false
        }
    }

    if split_attempted != nil {
        log.Printf(`Can't split on %s, leaving unchanged: %s`, split_attempted, exp)
        return exp, false
    }

    if m := unary_exp_re.FindStringSubmatch(exp); m != nil {
        return convert_unary_exp(m[1], m[2], no_enclosing_brackets)
    }

    if var_exp_re.MatchString(exp) {
        return join_exps("", ".", exp), true
    }

    log.Printf(`Don't know what to do with "%s", leaving unchanged`, exp)
    return exp, false
}

func convert_unary_exp(op string, exp string, no_enclosing_brackets bool) (string, bool) {
    go_fn := tiller_op_to_fn[op]
    converted_exp, is_var := convert_exp(exp, true)
    if go_fn != "" {
        if !is_var {
            converted_exp = enclose_in_brackets(converted_exp)
        }
        converted_exp = join_exps(" ", go_fn, converted_exp)
        is_var = false
    }

    if no_enclosing_brackets || is_var{
        return converted_exp, is_var
    }
    return enclose_in_brackets(converted_exp), false
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
