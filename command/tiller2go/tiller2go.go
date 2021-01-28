package main

import (
    "fmt"

    "github.com/catalyst/gotiller/convert"
    "github.com/catalyst/gotiller/command"
)

var command_line_flags = []*command.CommandLineFlag{
    &command.CommandLineFlag{
        "tiller-config-dir",
        "t",
        fmt.Sprintf("tiller config dir (default . then %s)", convert.ConfigEtcPath),
        "path",
        false,
        "",
        nil,
    },
}
var command_line_args = &command.CommandLineArgs{
    []string{"output-config-dir-path"},
    "",
    nil,
}

func main() {

    command.Run(
        command_line_flags,
        command_line_args,
        func() {
            in_dir  := *command_line_flags[0].ValueP.(*string)
            if len(command_line_args.Values) == 0 {
                panic("output-config-dir-path must be specified")
            }
            out_dir := command_line_args.Values[0]

            convert.Convert(in_dir, out_dir)
        },
    )
}
