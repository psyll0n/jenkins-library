package cmd

import (
	"bytes"
	"fmt"
	"github.com/SAP/jenkins-library/pkg/command"
	"github.com/SAP/jenkins-library/pkg/log"
	"github.com/SAP/jenkins-library/pkg/piperutils"
	"github.com/SAP/jenkins-library/pkg/telemetry"
	"github.com/SAP/jenkins-library/pkg/terraform"
)

type terraformExecuteUtils interface {
	command.ExecRunner

	FileExists(filename string) (bool, error)
}

type terraformExecuteUtilsBundle struct {
	*command.Command
	*piperutils.Files
}

func newTerraformExecuteUtils() terraformExecuteUtils {
	utils := terraformExecuteUtilsBundle{
		Command: &command.Command{},
		Files:   &piperutils.Files{},
	}
	// Reroute command output to logging framework
	utils.Stdout(log.Writer())
	utils.Stderr(log.Writer())
	return &utils
}

func terraformExecute(config terraformExecuteOptions, telemetryData *telemetry.CustomData, commonPipelineEnvironment *terraformExecuteCommonPipelineEnvironment) {
	utils := newTerraformExecuteUtils()

	err := runTerraformExecute(&config, telemetryData, utils, commonPipelineEnvironment)
	if err != nil {
		log.Entry().WithError(err).Fatal("step execution failed")
	}
}

func runTerraformExecute(config *terraformExecuteOptions, telemetryData *telemetry.CustomData, utils terraformExecuteUtils, commonPipelineEnvironment *terraformExecuteCommonPipelineEnvironment) error {
	if len(config.CliConfigFile) > 0 {
		utils.AppendEnv([]string{fmt.Sprintf("TF_CLI_CONFIG_FILE=%s", config.CliConfigFile)})
	}

	args := []string{}

	if config.Command == "apply" {
		args = append(args, "-auto-approve")
	}

	if (config.Command == "apply" || config.Command == "plan") && config.TerraformSecrets != "" {
		args = append(args, fmt.Sprintf("-var-file=%s", config.TerraformSecrets))
	}

	if config.AdditionalArgs != nil {
		args = append(args, config.AdditionalArgs...)
	}

	if config.Init {
		err := runTerraform(utils, "init", []string{}, config.GlobalOptions)

		if err != nil {
			return err
		}
	}

	err := runTerraform(utils, config.Command, args, config.GlobalOptions)

	if err != nil {
		return err
	}

	var outputBuffer bytes.Buffer
	utils.Stdout(&outputBuffer)

	err = runTerraform(utils, "output", []string{"-json"}, config.GlobalOptions)

	if err != nil {
		return err
	}

	commonPipelineEnvironment.custom.terraformOutputs, err = terraform.ReadOutputs(outputBuffer.String())

	return err
}

func runTerraform(utils terraformExecuteUtils, command string, additionalArgs []string, globalOptions []string) error {
	args := []string{}

	if len(globalOptions) > 0 {
		args = append(args, globalOptions...)
	}

	args = append(args, command)

	if len(additionalArgs) > 0 {
		args = append(args, additionalArgs...)
	}

	return utils.RunExecutable("terraform", args...)
}
