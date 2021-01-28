package command

import (
    "os"
    "io"
    "testing"
    "path/filepath"

    "github.com/stretchr/testify/assert"
)

var flags = []*CommandLineFlag{
    &CommandLineFlag{
        "string",
        "s",
        "string flag",
        "some_string",
        false,
        "",
        nil,
    },
    &CommandLineFlag{
        "bool",
        "b",
        "a switch",
        "",
        false,
        false,
        nil,
    },
}
var args = &CommandLineArgs{
    []string{"arg1", "arg2"},
    "Some args at the end",
    nil,
}
const usage = `Usage:
command.test [--string|-s some_string] [--bool|-b] arg1 arg2
  -b, --bool            a switch
  -s, --string string   string flag
Some args at the end
`

func Test_ParseArgs(t *testing.T) {
    dir := t.TempDir()
    out_path := filepath.Join(dir, "stderr")
    out, err := os.Create(out_path)
    if err != nil {
        panic(err)
    }
    os.Stderr = out

    usage_fn := ParseArgs(flags, args)
    usage_fn()

    err = out.Sync()
    if err != nil {
        panic(err)
    }
    out_b := make([]byte, 512)
    n, err := out.ReadAt(out_b, 0)
    if err != io.EOF {
        if err == nil {
            panic("Buffer too small")
        }
        panic(err)
    }
    assert.Equal(t, usage, string(out_b[0:n-1]), "Usage")
}
/*
func Example_ParseArgs() {
    usage_fn := ParseArgs(flags, args)
    usage_fn()

    // Output:
    // Usage:
    // command.test [--string|-s some_string] [--bool|-b] arg1 arg2
    //  -b, --bool            a switch
    //  -s, --string string   string flag
    // Some args at the end
}
*/
