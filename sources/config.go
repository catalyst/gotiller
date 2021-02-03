// Infrastructure for config files Deployables. No Templates.

package sources

import (
    "github.com/catalyst/gotiller/util"
)

// Generic DeployablesSource type and SourceInterface implementation
type DeployablesSource struct {
    *Deployables
    BaseSource
}
func (d *DeployablesSource) MergeConfig(origin string, deployables interface{}) {
    logger.Debugf("Making deployables from %s\n", origin)
    deployables_d := MakeDeployables(deployables.(util.AnyMap))

    d.AddHistory(origin, deployables_d)
    if d.Deployables == nil {
        d.Deployables = deployables_d
    } else {
        d.Deployables.Merge(deployables_d)
    }
}
func (d *DeployablesSource) DeployablesForEnvironment(environment string) *Deployables {
    return d.Deployables
}

func MakeDeployablesSource() SourceInterface {
    return &DeployablesSource{nil, MakeBaseSource()}
}

// A Deployables per environment version of DeployablesSource type
type EnvironmentDeployables map[string]*Deployables
type EnvironmentsSource struct {
    EnvironmentDeployables
    BaseSource
}
func (e *EnvironmentsSource) MergeConfig(origin string, es interface{}) {
    es_m := es.(util.AnyMap)

    e.AddHistory(origin, es_m)
    for environment, deployables := range es_m {
        logger.Debugf("Making %s deployables from %s\n", environment, origin)
        if deployables == nil {
            logger.Debugln("No deployables")
            continue
        }
        deployables_d := MakeDeployables(deployables.(util.AnyMap))
        if d, exists := e.EnvironmentDeployables[environment]; exists {
            logger.Debugf("Merging %s deployables\n", environment)
            d.Merge(deployables_d)
        } else {
            e.EnvironmentDeployables[environment] = deployables_d
        }
    }
}
func (e *EnvironmentsSource) DeployablesForEnvironment(environment string) *Deployables {
    return e.EnvironmentDeployables[environment]
}
func (e *EnvironmentsSource) AllEnvironments() []string {
    var es []string
    for e, _ := range e.EnvironmentDeployables {
        es = append(es, e)
    }
    return es
}

func MakeEnvironmentsSource() SourceInterface {
    return &EnvironmentsSource{make(EnvironmentDeployables), MakeBaseSource()}
}

func init() {
    RegisterSource("defaults", MakeDeployablesSource, 20, false)
    RegisterSource("environments", MakeEnvironmentsSource, 30, false)
}
