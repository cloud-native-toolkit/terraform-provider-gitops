package gitops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	"os/exec"
)

func readGitopsMetadata(ctx context.Context, binDir string, gitopsConfig GitopsMetadataConfig) (*GitopsMetadata, error) {

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Retrieving gitops metadata: serverName=%s", gitopsConfig.ServerName))

	var args = []string{
		"gitops-metadata-get",
		"--branch", gitopsConfig.Branch,
		"--serverName", gitopsConfig.ServerName,
		"--output", "jsonfile=./output.json"}

	if len(gitopsConfig.CaCert) > 0 {
		args = append(args, "--caCert", gitopsConfig.CaCert)
	}
	if len(gitopsConfig.Debug) > 0 {
		args = append(args, "--debug", gitopsConfig.Debug)
	}

	cmd := exec.Command("igc", args...)
	cmd.Path = pathWithBinDir(binDir)

	tflog.Debug(ctx, "Executing command: "+cmd.String())

	gitEmail := "cloudnativetoolkit@gmail.com"
	gitName := "Cloud Native Toolkit"

	updatedEnv := append(os.Environ(), "GIT_CREDENTIALS="+gitopsConfig.Credentials)
	updatedEnv = append(updatedEnv, "GITOPS_CONFIG="+gitopsConfig.Config)
	updatedEnv = append(updatedEnv, "KUBECONFIG="+gitopsConfig.KubeConfigPath)
	updatedEnv = append(updatedEnv, "EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_NAME="+gitName)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_NAME="+gitName)

	logEnvironment(ctx, &updatedEnv)

	cmd.Env = updatedEnv

	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	// start the command after having set up the pipe
	if err := cmd.Start(); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Error starting command: %s", fmt.Sprintln(err)))
		tflog.Error(ctx, fmt.Sprintf("Command error log: %s", errb.String()))
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Error running command: %s", fmt.Sprintln(err)))
		tflog.Error(ctx, fmt.Sprintf("Command error log: %s", errb.String()))
		return nil, err
	}

	outText := outb.String()
	if len(outText) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("Command standard log: %s", outText))
	}

	errText := errb.String()
	if len(errText) > 0 {
		tflog.Error(ctx, fmt.Sprintf("Command error log: %s", errText))
	}

	dat, err := os.ReadFile("./output.json")
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, fmt.Sprintf("JSON result from gitops metadata: %s", string(dat)))

	gitopsMetadata := GitopsMetadata{}
	err = json.Unmarshal(dat, &gitopsMetadata)
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, fmt.Sprintf("Result values from gitops metadata: %s", gitopsMetadata.Cluster.Type))

	return &gitopsMetadata, nil
}
