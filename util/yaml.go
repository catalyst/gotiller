package util

import (
    "gopkg.in/yaml.v3"
)

func ReadYaml(path string, target interface{}) {
    if err := yaml.Unmarshal(SlurpFile(path), target); err != nil {
        panic(err)
    }
}

func WriteYaml(path string, source interface{}) {
    bytes, err := yaml.Marshal(source)
    if err != nil {
        panic(err)
    }

    WriteFile(path, bytes);
}
