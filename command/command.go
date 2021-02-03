// Command line helpers

package command

import (
    "fmt"
    "os"
    "path/filepath"
    "log"

    "github.com/spf13/pflag"
)

// Types for defining command line flags and arguments
type CommandLineFlag struct {
    Long       string
    Short      string
    Usage      string
    UsageParam string
    Mandatory  bool
    DefaultValue interface{}
    ValueP       interface{}
}
type CommandLineArgs struct {
    Names      []string
    Usage      string
    Values     []string
}

// Marshal command line into CommandLine* types
func ParseArgs(flags []*CommandLineFlag, args *CommandLineArgs) func() {
    out := os.Stderr
    pflag.CommandLine.SetOutput(out)

    usage_line := filepath.Base(os.Args[0])

    for _, f := range flags {
        u := "--" + f.Long
        if  f.Short != "" {
            u = u + "|-" + f.Short
        }

        switch t := f.DefaultValue.(type) {
            case string:
                f.ValueP = pflag.StringP(f.Long, f.Short, f.DefaultValue.(string), f.Usage)
                u = u + " " + f.UsageParam
            case bool:
                f.ValueP = pflag.BoolP  (f.Long, f.Short, f.DefaultValue.(bool),   f.Usage)
            // case int, float:
            default:
                log.Panicf("Command line flag type %s not supported", t)
        }

        if !f.Mandatory {
            u = "[" + u + "]"
        }

        usage_line = usage_line + " " + u
    }

    if args != nil && args.Names != nil {
        for _, n := range args.Names {
            usage_line = usage_line + " " + n
        }
    }

    pflag.Usage = func() {
        fmt.Fprintln(out, "Usage:")
        fmt.Fprintln(out, usage_line)
        pflag.PrintDefaults()
        if args.Usage != "" {
            fmt.Fprintln(out, args.Usage)
        }
        fmt.Fprintln(out, "")
    }

    pflag.Parse()
    if args != nil {
        args.Values = pflag.Args()
    }

    return pflag.Usage
}

// A thin wrapper intended for main() funcs. Calls ParseArgs() and sets panic() handler
func Run(flags []*CommandLineFlag, args *CommandLineArgs, main_fn func()) {
    usage_fn := ParseArgs(flags, args)

    defer func() {
        if r := recover(); r != nil {
            usage_fn()
            log.Fatal(r)
        }
    }()

    main_fn()
}
