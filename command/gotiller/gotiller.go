package main

import (
    "fmt"
    "os"

    "github.com/catalyst/gotiller"
    "github.com/catalyst/gotiller/sources"
    "github.com/catalyst/gotiller/command"
)

const ConfigEtcPath = "/etc/gotiller"

var command_line_flags = []*command.CommandLineFlag{
    &command.CommandLineFlag{
        "config-dir",
        "d",
        fmt.Sprintf("gotiller config dir (default . then %s)", ConfigEtcPath),
        "path",
        false,
        "",
        nil,
    },
    &command.CommandLineFlag{
        "output-base-dir",
        "o",
        "root dir for generate files (usually not needed)",
        "path",
        false,
        "",
        nil,
    },
    &command.CommandLineFlag{
        "verbose",
        "v",
        "",
        "",
        false,
        false,
        nil,
    },
}
var command_line_args = &command.CommandLineArgs{
    []string{"[environment]"},
    "If environment is not specified, default_environment from config is assumed",
    nil,
}
func main() {
    command.Run(
        command_line_flags,
        command_line_args,
        func() {
            dir             := *command_line_flags[0].ValueP.(*string)
            target_base_dir := *command_line_flags[1].ValueP.(*string)
            verbose         := *command_line_flags[2].ValueP.(*bool)
            env             := ""

            if len(command_line_args.Values) > 0 {
                env = command_line_args.Values[0]
            }

            if dir == "" {
                if _, err := os.Stat(sources.ConfigFname); err == nil {
                    dir = "."
                } else {
                    dir = ConfigEtcPath
                }
            }

            gotiller.Execute(dir, env, target_base_dir, verbose)
        },
    )
}
