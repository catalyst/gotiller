package main

import (
    "fmt"
    "os"

    "github.com/catalyst/gotiller/convert"
    "github.com/spf13/pflag"
)

func main() {
    in_dir_p := pflag.StringP("tiller-config-dir",  "t", "", fmt.Sprintf("tiller config dir (default . then %s)", convert.ConfigEtcPath))
    strip_var_prefix_p := pflag.StringP("strip-var-prefix",  "s", "", "strip prefix from vars")
    pflag.Usage = func() {
        fmt.Println("Usage:")
        fmt.Println(os.Args[0] + " [--tiller-config-dir|-t path] output-config-dir-path")
        pflag.PrintDefaults()
        fmt.Println()
    }
    pflag.Parse()

    out_dir_p := pflag.Arg(0)

    defer func() {
        if r := recover(); r != nil {
            pflag.Usage()
            panic(r)
        }
    }()

    convert.Convert(*in_dir_p, out_dir_p, *strip_var_prefix_p)
}
