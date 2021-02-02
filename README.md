GoTiller
========

Ruby Tiller replacement implemented in Go.

It is not a drop-in replacement, but it is similar enough. Some basic
conversion is provided.

**An important convention:**

GoTiller has a concept of **environment** inherited from Tiller to
separate sets of rules for different scenarios/purposes, eg *prod*,
*uat*, *test* etc.

At the same time we have the (shell-ish) *environment* in which
`gotiller` runs, and sources variables from. In order to minimise the
confusion we will refer to that *environment* as **env**.

If You have used Tiller
-----------------------

### What's the same

-   Config files are Yaml
-   Config directory structure/file naming
-   The intention is to keep the YAML structure as similar as possible

### What's not

#### Sources and variables

-   `data_sources:` and `template_sources:` do not exist - plugins are
    implied with inclusion of the corresponding sections; "File" plugin
    is assumed
-   `config:` is replaced with `vars:`
-   `defaults: global:` is replaced with `default_vars:`

Rationale: `defaults: global:` is equivalent to
`defaults: <any_template>: config:`, thus breaking the structure of
`defaults: <template>:`. Besides, it means that you cannot have template
named `global`. So it had to be renamed. I find `global_config:` to be
confusing, so I went with `default_vars:`.

Once `default_vars:` was there, `config:` had to become `vars:`.

#### Environment plugin

This is a big change.

-   The `environment:` section is replaced with a single entry
    `env_vars_prefix:`
-   `env_vars_prefix` is not implied, ie if it is missing it is not
    assumed that we want env vars staring with `env_`
-   \"\" is not a valid value, ie you cannot have `env_vars_prefix: ""`
    to slurp all environment - it is an equivalent of not having
    `env_vars_prefix:`
-   `lowercase:` is not supported, ie env vars will not be lowercased
    before checking the prefix
-   `env_vars_prefix` is stripped down from matching env vars - an
    example:

<!-- -->

    in your template:    "host={{ .db_host }}"
    in your config:      env_vars_prefix: env_
    in your running env: env_db_host=myhost

Conversion will strip `env_vars_prefix` from vars in templates.

#### Plugins

At the moment, only `filesystem` and `environment` sources are implemented.

#### ERB vs Go templates

Apart from the trivial embedding tag difference (`<% %>` pair in ERB vs
`{{ }}` in Go templates) there are notable differences. Let's see what
is missing. For what new is availabe have a look at the [Go text
template package docs](https://golang.org/pkg/text/template/).

##### No logical operators

Go template control structures ("actions") only do functions. So if you
had `"if a && b"` that turns into `"if and a b"`

##### No code embedding

You cannot just run (evaluate) any random code. You can only do:

-   acions (`if`, `range` (for loops) etc)
-   function calls: `fn_name arg1 arg2...`

Templating processor is preloaded with a small set of functions, mainly
ones that do logical operations and text formatting. However it is
possible to make functions available to the processor. We will make some
popular functions available in our `gotiller` executable, and there's
always an option to build a custom one.

### Conversion

TBD

If you haven't used Tiller
--------------------------

[Tiller](https://tiller.readthedocs.io) is an one-off puppet. It was
created to alleviate pain of configuring containers that come fom the
same image but run in different environments.

Config directory and YAML structure
-----------------------------------

    |- common.yaml
    |- config.d
    |   |- conf1.yaml
    |   |- conf2.yaml
    |   └- ...
    |
    |- environments
    |   |- prod.yaml
    |   |- uat.yaml
    |   |- test.taml
    |   └- ...
    |
    └- templates
        |- some.conf
        |- another.ini
        └- ...

### Config files

Base config file is `common.yaml`. Files from `config.d` overlay config
options/rules on top, and are taken in alphabetical order.

In the following structures, any of the entries/stanzas could be
missing.

#### Config structure

Config files understand following keys:

-   default\_vars - default template vars substitutions, applicable to
    all environments/templates
-   defaults: - base templates structure (see below) applicable to all
    environments
-   environments - environments structure (see below)
-   default\_environment - environment to assume if no environment is
    specified
-   env\_vars\_prefix - prefix of the env vars (see convention at the
    top) to apply; if missing or empty no vars are taken from env

    default_vars:
      var1: val1
      ...

    defaults: {Templates structure}

    environments:
      prod: {Templates structure}
      test: {Templates structure}
      environmentX: {Templates structure}
      ...

    env_vars_prefix: env_

#### Templates structure

Templates are keyed on template filenames, as stored in `templates`
directory:

    tpl1: {Target structure}
    tpl2: {Target structure}
    ...

#### Target structure

`user:` and `group:` entries are optional, default to running process
username/group.

    target: /path/on/the/filesystem/where/to/write/processed/template

    user: os-username
    group: os-group

    vars:
      var1: val1

### Environment files

File`some_enironment.yaml` in `environments` directory hold *Template*
structure, an equivalent of `environment: -> some_environment:`

Target parameter rules
----------------------

When `gotiller` runs it

-   forms a working *Templates* structure
-   for each template from the working structure forms a set of vars to
    apply and processes the template

### Working Config structure formation

-   Take `common.yaml`, otherwise an empty structure
-   Overlay files from `config.d` in alphabetical order

### Working Templates structure formation for environment

Let's name the specified environment *eX*

-   From the working Config structure:
    -   Take `defaults:`
    -   Overlay with `environments: -> eX:`
-   Overlay with `environments/eX.yaml`

`environments/eX.yaml` trumps `environments: -> eX:` from the working
config, trumps `defaults:` from the working config

### Variable set formation for template

Working *Target* is the Target structure for the working *template*
taken from the Working Templates

-   From the working Config structure take `default_vars:`
-   Overlay with working Target `vars:`
-   Overlay with *env vars*

*env vars* trump Target `vars:` trump `default_vars:`

### Utility functions

Functions that are available in templates to make life easier:

#### `sequence start length`

To be used with `range` to create incremented loops:

    {{range sequence 0 3}}
    something
    {{end}}

will give `something0 something1 something2`

#### `hash string`

Gives CRC32 hashed value (8 hex digits) of a string.

#### `isfile path`

Returns boolean whether the file specified with path exists. In case of a dir throws an exception.

### A full blown example
#### Config

**`common.conf`**
```
default_vars:
    v1: v_default_1
    v2: v_default_2
    v3: v_default_3

defaults:
    tpl.conf:
        target: /app/a.conf
        vars:
            v_true_1: "A string"
            v_true_2: 1
            v_false_1: ""
            v1: v_template_default_1
            v2: v_template_default_2

env_vars_prefix: env_

default_environment: e1

environments:
    e1:
        tpl.conf:
            vars:
                v2: v_common_e1_2
                x: v_common_e1_x
                y: v_common_e1_y
    e2:
        tpl.conf:
            vars:
                v1: v_common_e2_1
                v3: v_common_e2_3
                x: v_common_e2_x
                z: v_common_e2_z

```

**`config.d/xyz.yaml`**
```
default_vars:
    x: v_default_x
    y: v_default_y
    z: v_default_z

```

**`environments/e1.yaml`**
```
tpl.conf:
    vars:
        x: v_env1_x

```

**Running env**
```
env_y=v_env_y

```

#### Template
```
# Config for app

{{if and .v_true_1 (not .v_false_1) -}}

param_1_that_shows="{{.v1}} from common.yaml.defaults"

  {{- if and .v_true_1 .v_false_1}}

some_param="This won't show"

  {{- end}}
  {{- if or (and .v_true_1 .v_false_1) (not (and .v_true_2 .v_false_2))}}

param_2_that_shows="{{.v2}} from common.yaml environments e1"

  {{- end}}

{{- /* This is a comment.
       On multiple lines. */ -}}
{{- end}}

param_3="{{.v3}} from common.yaml default_vars"

param_x="{{.x}} from environments/e1"

param_y="{{.y}} from env"

param_z="{{.z}} from config.d/xyz.yaml defaults"

{{/* This demonstrates saving current level when changing ".".  If there's no "-" before comment, there can be no space before "/*" */ -}}
{{- $save := . -}}
{{- /* range is loop */ -}}
{{- range sequence 0 3 -}}
{{$save.z}}
{{end -}}

```

#### Result
```
# Config for app

param_1_that_shows="v_template_default_1 from common.yaml.defaults"

param_2_that_shows="v_common_e1_2 from common.yaml environments e1"

param_3="v_default_3 from common.yaml default_vars"

param_x="v_env1_x from environments/e1"

param_y="v_common_e1_y from env"

param_z="v_default_z from config.d/xyz.yaml defaults"

v_default_z0
v_default_z1
v_default_z2

```

Makefile
--------

`make` builds statically compiled executables for specified arhitectures. If no
architecture is specified it will build for default architecture (`amd64`). For example:

    make ARCH="amd64 arm"

Binaries are created in the `bin` directory.

CLI
---

TBD
