package sources

import (
    "github.com/catalyst/gotiller/util"
)

type VarsSource struct {
    Vars
    BaseSource
}
func (v *VarsSource) MergeConfig(origin string, vars interface{}) {
    logger.Debugf("Merging vars from %s\n", origin)
    vars_v := MakeVars(vars.(util.AnyMap))

    v.AddHistory(origin, vars_v)
    v.Vars.Merge(vars_v)
}
func (v *VarsSource) DefaultVars() Vars {
    return v.Vars
}

func MakeVarsSource() SourceInterface {
    return &VarsSource{make(Vars), MakeBaseSource()}
}

type DeployablesSource struct {
    Deployables
    BaseSource
}
func (d *DeployablesSource) MergeConfig(origin string, deployables interface{}) {
    logger.Debugf("Making default deployables from %s\n", origin)
    deployables_d := MakeDeployables(deployables.(util.AnyMap))

    d.AddHistory(origin, deployables_d)
    d.Deployables.Merge(deployables_d)
}
func (d *DeployablesSource) DeployablesForEnvironment(environment string) Deployables {
    return d.Deployables
}

func MakeDeployablesSource() SourceInterface {
    return &DeployablesSource{make(Deployables), MakeBaseSource()}
}

type EnvironmentDeployables map[string]Deployables
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
            d.Merge(deployables_d)
        } else {
            logger.Debugf("Setting %s deployables\n", environment)
            e.EnvironmentDeployables[environment] = deployables_d
        }
    }
}
func (e *EnvironmentsSource) DeployablesForEnvironment(environment string) Deployables {
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
    RegisterSource("default_vars", MakeVarsSource, 10, false)
    RegisterSource("defaults", MakeDeployablesSource, 20, false)
    RegisterSource("environments", MakeEnvironmentsSource, 30, false)
}
