package convert

import (
    "path/filepath"
    "runtime"

    "testing"
    "github.com/stretchr/testify/assert"

    "gopkg.in/yaml.v3"

    "github.com/catalyst/gotiller/sources"
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
var c1_templates = []string{"db.erb"}
var c1 = util.AnyMap {
    "default_vars": util.AnyMap {
        "db_hostname": "localhost",
    },
    "defaults": util.AnyMap {
        "db.ini": util.AnyMap {
            "target": "db.ini",
        },
    },
    "environments": util.AnyMap {
        "development": nil,
        "production": util.AnyMap {
            "db.ini": util.AnyMap {
                "vars": util.AnyMap {
                    "db_hostname": "db.prod.example.com",
                },
            },
        },
        "staging": util.AnyMap {
            "db.ini": util.AnyMap {
                "vars": util.AnyMap {
                    "db_hostname": "db.staging.example.com",
                },
            },
        },
    },
    "blah": nil,
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

# Start pgbouncer, if required
<%- if !defined?(env_disable_pgbouncer) -%>
/etc/init.d/pgbouncer start
<%- end -%>
`,
        GoTiller: `
{{- if and .x .y -}}
appname = "{{ .app }}"
{{- else -}}
appname = "blah"
{{- end -}}

# Start pgbouncer, if required
{{- if not .disable_pgbouncer -}}
/etc/init.d/pgbouncer start
{{- end -}}
`,
    },
}
func Test_Converter(t *testing.T) {
    source_dir := t.TempDir()

    source_config_path := filepath.Join(source_dir, sources.ConfigFname)
    util.WriteFile(source_config_path, []byte(c1_tiller))

    source_templates_path := filepath.Join(source_dir, sources.TemplatesSubdir)
    util.Mkdir(source_templates_path)
    for _, t := range c1_templates {
        util.Touch(filepath.Join(source_templates_path, t))
    }

    config := make(util.AnyMap)
    if err := yaml.Unmarshal([]byte(c1_tiller), config); err != nil {
        panic(err)
    }

    converter := NewConverter(source_dir, "/tmp", "env_")
    converter.convert_config(config)

    assert.Equal(t, c1, config, "convert_config()")

    for _, ct := range template_conversion_tests {
        assert.Equal(t, ct.GoTiller, converter.convert_template(ct.Tiller), "convert_template()")
    }
}

var (
    _, b, _, _ = runtime.Caller(0)
    base_dir   = filepath.Dir(b)
)

func Test_Convert(t *testing.T) {
}
