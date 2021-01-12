# GoTiller
Ruby Tiller replacement implemented in Go.

It is not a drop-in replacement, but it is similar enough. Some basic conversion is provided.

## If You have used Tiller

### What's the same
* Config files are Yaml
* Config directory structure/file naming
* The intention is to keep the YAML structure as similar as possible

### What's not
#### Sources and variables
* `data_sources:` and `template_sources:` do not exist - plugins are implied with inclusion of
the corresponding sections; "File" plugin is assumed
* `config:` is replaced with `vars:`
* `defaults: globals:` is replaced with `default_vars:`

Rationale: `defaults: globals:` is equivalent to `defaults: <any_template>: config:`,
thus breaking the structure of `defaults: <template>:`. Besides, it means that you cannot have
template named `globals`. So it had to be renamed. I find `global_config:` to be confusing,
so I went with `global_vars:`.

Once `global_vars:` was there, `config:` had to become `vars:`.

#### Environment plugin
This is a big change.

* The `environment:` section is replaced with a single entry `env_vars_prefix:`
* `env_vars_prefix` is not implied, ie if it is missing it is not assumed that we want
env vars staring with `env_`
* "" is not a valid value, ie you cannot have `env_vars_prefix: ""` to slurp all environment - it
is an equivalent of not having `env_vars_prefix:`
* `lowercase:` is not supported, ie env vars will not be lowercased before checking the prefix
* `env_vars_prefix` is stripped down from matching env vars - an example:
```
in your template:    "host={{ .db_host }}"
in your config:      env_vars_prefix: env_
in your running env: env_db_host=myhost
```

Conversion will strip `env_vars_prefix` from vars in templates.

#### ERB vs Go templates
Apart from the trivial embedding tag difference (`<% %>` pair in ERB vs `{{ }}` in Go templates)
there are notable differences. Let's see what is missing. For what new is availabe have a look
at the [Go text template package docs](https://golang.org/pkg/text/template/).

##### No logical operators
Go template control structures ("actions") only do functions.
So if you had `"if a && b"` that turns into `"if and a b"`

##### No code embedding
You cannot just run (evaluate) any random code. You can only do:

* acions (`if`, `range` (for loops) etc)
* function calls: `fn_name arg1 arg2...`

Templating processor is preloaded with a small set of functions, mainly ones that do
logical operations and text formatting. However it is possible to make functions available
to the processor. We will make some popular functions available in our `gotiller` executable,
and there's always an option to build a custom one.

### Conversion
TBD

## If you haven't used Tiller
[Tiller](https://tiller.readthedocs.io) is an one-off puppet. It was created to alleviate
pain of configuring containers that come fom the same image but run in different environments.

## Config directory and YAML structure
TBD

## CLI
TBD