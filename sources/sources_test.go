package sources

import (
    "os"
    "os/user"
    "syscall"
    // "io/ioutil"
    "path/filepath"
    "fmt"

    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/catalyst/gotiller/util"
)

var t1 = Deployables {
    "deployable1": &Spec {
        Target: "target1",
        User  : "user1",
        Vars  : Vars {
            "var1": "val1",
            "varX": "val1X",
        },
    },
    "deployableX": &Spec {
        Target: "targetX1",
        User  : "user1",
        Group : "group1",
        Vars  : Vars {
            "var1": "val1",
            "varX": "val1X",
        },
    },
}
var t2 = Deployables {
    "deployable2": &Spec {
        Target: "target2",
        User  : "user2",
        Vars  : Vars {
            "var1": "val2",
            "varX": "val2X",
        },
    },
    "deployableX": &Spec {
        Target: "targetX2",
        User  : "user2",
        Perms : os.FileMode(0644),
        Vars  : Vars {
            "var2": "val2",
            "varX": "val2X",
        },
    },
}
var t1_t2 = Deployables {
    "deployable1": &Spec {
        Target: "target1",
        User  : "user1",
        Vars  : Vars {
            "var1": "val1",
            "varX": "val1X",
        },
    },
    "deployable2": &Spec {
        Target: "target2",
        User  : "user2",
        Vars  : Vars {
            "var1": "val2",
            "varX": "val2X",
        },
    },
    "deployableX": &Spec {
        Target: "targetX2",
        User  : "user2",
        Group : "group1",
        Perms : os.FileMode(0644),
        Vars  : Vars {
            "var1": "val1",
            "var2": "val2",
            "varX": "val2X",
        },
    },
}
func Test_merge_deployables(t *testing.T) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    tr := make(Deployables)
    tr.Merge(t1)
    tr.Merge(t2)

    assert.Equal(t, t1_t2, tr, "merge_deployables()")
}

const perms = os.FileMode(0600)
const os_user = "nobody"
const os_group = "nogroup"
var spec = Spec {
    Target: "/etc/dummy/target.conf",
    Perms : perms,
    User: os_user,
    Group: os_group,
    Vars  : Vars {
        "varT": "valT",
        "varA": "valTA",
        "varB": "valTB",
        "varC": "valTC",
    },
}
var vars = Vars {
    "varV": "valV",
    "varA": "valVA",
    "varB": "valVB",
    "varD": "valVD",
}
var default_vars = Vars {
    "varDV": "valDV",
    "varA": "valDVA",
    "varC": "valDVC",
    "varD": "valDVD",
}
var merged_vars = Vars {  // default_vars + spec.Vars + vars
    "varT": "valT",
    "varV": "valV",
    "varDV": "valDV",
    "varA": "valVA",
    "varB": "valVB",
    "varC": "valTC",
    "varD": "valVD",
}
const deployable_in string = `
{{.varA}}
{{.varB}}
{{.varC}}
{{.varD}}
{{.varT}}
{{.varV}}
{{.varDV}}
`
const content_out string = `
valVA
valVB
valTC
valVD
valT
valV
valDV
`
func Test_merge_vars(t *testing.T) {
    vr := make(Vars)
    vr.Merge(default_vars, spec.Vars, vars)

    assert.Equal(t, merged_vars, vr, "merge_vars()")
}

func Test_chown(t *testing.T) {
    t.Cleanup(util.SupressLogForTest(t, logger))

    u, err := user.Current()
    if err != nil {
        panic(err)
    }

    if u.Username != "root" {
        t.Skipf("Running as %s (not root)", u.Username)
    }

    dir := t.TempDir()
    templates_dir := filepath.Join(dir, TemplatesSubdir)
    util.Mkdir(templates_dir)

    tpl := "target.conf"
    tpl_path := filepath.Join(templates_dir, tpl)
    util.Touch(tpl_path)

    spec.Deploy(tpl, "")

    stat, err := os.Stat(spec.Target)
    if err != nil {
        panic(err)
    }
    assert.Equal(t, perms, stat.Mode(), "generated file mode (permissions)")

    sys_stat := stat.Sys().(*syscall.Stat_t)

    sys_user, err := user.LookupId(fmt.Sprint(sys_stat.Uid))
    if err != nil {
        panic(err)
    }
    assert.Equal(t, os_user, sys_user.Username, "generated file owner")

    sys_group, err := user.LookupGroupId(fmt.Sprint(sys_stat.Gid))
    if err != nil {
        panic(err)
    }
    assert.Equal(t, os_group, sys_group.Name, "generated file group")
}
