package sources

import (
    "os"
    "io/ioutil"
    "strings"
    "path/filepath"

    "github.com/catalyst/gotiller/util"
)

const (
    ConfigSuffix       = ".yaml"
    ConfigFname        = "common" + ConfigSuffix
    ConfigD            = "config.d"
    EnvironmentsSubdir = "environments"
    TemplatesSubdir    = "templates"
)

type FileSystemSource struct {
    EnvironmentsSource
    Templates
}
func (f *FileSystemSource) MergeConfig(origin string, d interface{}) {
    d_m := d.(map[string]string)

    dir := d_m["dir"]
    suffix := d_m["suffix"]

    environment_pattern := filepath.Join(dir, EnvironmentsSubdir, "*" + suffix)
    if matches, _ := filepath.Glob(environment_pattern); matches != nil {
        logger.Debugf("Entering %s\n", EnvironmentsSubdir)
        es := make(util.AnyMap)
        for _, m := range matches {
            envinment := strings.TrimSuffix(filepath.Base(m), suffix)

            logger.Debugf("Loading %s\n", envinment)
            config := make(util.AnyMap)

            util.ReadYaml(m, config)

            es[envinment] = config
        }
        f.EnvironmentsSource.MergeConfig(environment_pattern, es)
    }

    template_dir_path := filepath.Join(dir, TemplatesSubdir)
    if dir_entries, err := ioutil.ReadDir(template_dir_path); err == nil {
        for _, entry := range dir_entries {
            t := entry.Name()

            f.Templates[t] = &Template{filepath.Join(template_dir_path, t), ""}
        }
    }
}
func (f *FileSystemSource) Template(name string) string {
    t, exists := f.Templates[name]
    if !exists {
        return ""
    }

    if t.Content == "" {
        t.Content = string(util.SlurpFile(t.Path))
    }

    return t.Content
}
func (f *FileSystemSource) AllTemplates() Templates {
    return f.Templates
}

func MakeFileSystemSource() SourceInterface {
    es := MakeEnvironmentsSource()
    return &FileSystemSource{*es.(*EnvironmentsSource), make(map[string]*Template)}
}

func init() {
    RegisterSource("filesystem", MakeFileSystemSource, 50, false)
}

func LoadConfigsFromDir(dir string) *Processor {
    processor := NewProcessor()

    config_path := filepath.Join(dir, ConfigFname)
    if _, err := os.Stat(config_path); err == nil {
        logger.Debugf("Reading main config %s\n", ConfigFname)
        processor.MergeConfig(config_path, LoadConfigFile(config_path))
    } else {
        logger.Debugf("No main config %s\n", ConfigFname)
    }

    config_pattern := filepath.Join(dir, ConfigD, "*" + ConfigSuffix)
    if matches, _ := filepath.Glob(config_pattern); matches != nil {
        logger.Debugf("Entering %s\n", ConfigD)
        for _, m := range matches {
            processor.MergeConfig(m, LoadConfigFile(m))
        }
    }

    processor.Get("filesystem").MergeConfig(dir, map[string]string{"dir": dir, "suffix": ConfigSuffix})

    return processor
}

func LoadConfigFile(path string) util.AnyMap {
    config := make(util.AnyMap)

    util.ReadYaml(path, config)

    return config
}
