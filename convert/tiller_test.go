package convert

import (
    /*
    "os"
    "io/ioutil"
    "log"
    "fmt"
    "strings"
    "github.com/catalyst/gotiller/util"
    */
    "path/filepath"
    "runtime"

    "testing"
    "github.com/stretchr/testify/assert"

    "gopkg.in/yaml.v3"

    "github.com/catalyst/gotiller"
    "github.com/catalyst/gotiller/util"
)

const c1_tiller = `
data_sources: [ "file" , "environment" ]
template_sources: [ "file" ]

defaults:
  global:
    env_db_hostname: localhost
  db.erb:
    target: db.ini

environments:
  development:
  production:
    db.erb:
      config:
        env_db_hostname: db.prod.example.com
  staging:
    db.erb:
      config:
        env_db_hostname: db.staging.example.com

blah:
`
var c1 = AnyMap {
    "default_vars": AnyMap {
        "db_hostname": "localhost",
    },
    "defaults": AnyMap {
        "db.ini": AnyMap {
            "target": "db.ini",
        },
    },
    "environments": AnyMap {
        "development": nil,
        "production": AnyMap {
            "db.ini": AnyMap {
                "vars": AnyMap {
                    "db_hostname": "db.prod.example.com",
                },
            },
        },
        "staging": AnyMap {
            "db.ini": AnyMap {
                "vars": AnyMap {
                    "db_hostname": "db.staging.example.com",
                },
            },
        },
    },
    "blah": nil,
}
func Test_convert_config(t *testing.T) {
    source_dir := t.TempDir()
    source_config_path := filepath.Join(source_dir, gotiller.ConfigFname)
    util.WriteFile(source_config_path, []byte(c1_tiller))

    config := make(AnyMap)
    if err := yaml.Unmarshal([]byte(c1_tiller), config); err != nil {
        panic(err)
    }

    converter := NewConverter(source_dir, "/tmp", "env_")
    converter.convert_config(config)

    assert.Equal(t, c1, config, "convert_config()")
}

type TemplateConversion struct {
    Tiller   string
    GoTiller string
}
var template_conversion_tests = []TemplateConversion {
    {
        Tiller: `
<%- if defined?(x) && defined?(y) -%>
appname = "<%= app %>"
<%- else -%>
appname = "blah"
<%- end -%>
`,
        GoTiller: `
{{- if and .x .y -}}
appname = "{{ .app }}"
{{- else -}}
appname = "blah"
{{- end -}}
`,
    },
}
func Test_convert_template(t *testing.T) {
    for _, ct := range template_conversion_tests {
        assert.Equal(t, ct.GoTiller, convert_template(ct.Tiller), "convert_template()")
    }
}

var (
    _, b, _, _ = runtime.Caller(0)
    base_dir   = filepath.Dir(b)
)

func Test_FromTiller(t *testing.T) {
}
