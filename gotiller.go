// Main entry

package gotiller

import (
    "github.com/catalyst/gotiller/sources"
    "github.com/catalyst/gotiller/log"
)

var logger = log.DefaultLogger

// Process config files and templates
func Process(dir string, environment string, target_base_dir string, verbose bool) *sources.Processor {
    logger.Printf("Executing from %s\n", dir)
    if target_base_dir != "" {
        logger.Printf("Writing to %s\n", target_base_dir)
    }

    if verbose {
        logger.SetDebug(true)
    }

    processor := sources.LoadConfigsFromDir(dir)

    if environment == "" {
        if  processor.DefaultEnvironment != "" {
            environment = processor.DefaultEnvironment
            logger.Println("Executing for default environment")
        } else {
            logger.Println("Environment not specified, hope there are some defaults")
        }
    }

    logger.Printf("Executing for %s\n", environment)

    processor.RunForEnvironment(environment, target_base_dir)

    // for forensic purposes
    return processor
}
