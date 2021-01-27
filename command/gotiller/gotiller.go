package main

import (
    "fmt"
    "os"
    "log"

    "github.com/catalyst/gotiller"
    "github.com/catalyst/gotiller/sources"
    "github.com/spf13/pflag"
)

const ConfigEtcPath = "/etc/gotiller"

func main() {
    dir_p := pflag.StringP("config-dir",  "d", "", fmt.Sprintf("gotiller config dir (default . then %s)", ConfigEtcPath))
    target_base_dir_p := pflag.StringP("output-base-dir", "o", "", "root dir for generate files (usually not needed)")
    env_p := pflag.StringP("environment", "e", "", "environment")
    verbose_p := pflag.BoolP("verbose", "v", false, "verbose")
    pflag.Usage = func() {
        fmt.Println("Usage:")
        fmt.Println(os.Args[0] + " [--config-dir|-d path] [--output-base-dir|-o path] [--verbose|v] --environment|-e environment")
        pflag.PrintDefaults()
        fmt.Println()
    }
    pflag.Parse()

    defer func() {
        if r := recover(); r != nil {
            pflag.Usage()
            log.Fatal(r)
        }
    }()

    if *dir_p == "" {
        if _, err := os.Stat(sources.ConfigFname); err == nil {
            *dir_p = "."
        } else {
            *dir_p = ConfigEtcPath
        }
    }

    gotiller.Execute(*dir_p, *env_p, *target_base_dir_p, *verbose_p)
}
