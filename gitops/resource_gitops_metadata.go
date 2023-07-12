package gitops

import (
	"bufio"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"os"
	"os/exec"
)

func resourceGitopsMetadata() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsMetadataCreate,
		ReadContext:   resourceGitopsMetadataRead,
		UpdateContext: resourceGitopsMetadataUpdate,
		DeleteContext: resourceGitopsMetadataDelete,
		Schema: map[string]*schema.Schema{
			"server_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},
			"gitops_namespace": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"branch": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "main",
			},
			"credentials": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"config": {
				Type:     schema.TypeString,
				Required: true,
			},
			"kube_config_path": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceGitopsMetadataCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	metadataConfig := GitopsMetadataConfig{
		KubeConfigPath: getKubeConfigPath(d),
		Branch:         getBranchInput(d),
		ServerName:     getServerNameInput(d),
		Credentials:    getCredentialsInput(d),
		Config:         getGitopsConfigInput(d),
		GitopsNamespace: getGitopsNamespaceInput(d),
		CaCert:         config.GitConfig.CaCertFile,
		Debug:          config.Debug,
	}

	id, err := populateGitopsMetadata(ctx, config.BinDir, metadataConfig, false)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

func resourceGitopsMetadataRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	return diags
}

func resourceGitopsMetadataUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceGitopsMetadataRead(ctx, d, m)
}

func resourceGitopsMetadataDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	metadataConfig := GitopsMetadataConfig{
		KubeConfigPath: getKubeConfigPath(d),
		Branch:         getBranchInput(d),
		ServerName:     getServerNameInput(d),
		Credentials:    getCredentialsInput(d),
		Config:         getGitopsConfigInput(d),
		CaCert:         config.GitConfig.CaCertFile,
		Debug:          config.Debug,
	}

	id, err := populateGitopsMetadata(ctx, config.BinDir, metadataConfig, true)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

func populateGitopsMetadata(ctx context.Context, binDir string, gitopsConfig GitopsMetadataConfig, delete bool) (string, error) {

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Provisioning gitops metadata: serverName=%s", gitopsConfig.ServerName))

	var args = []string{
		"gitops-metadata-update",
		"--branch", gitopsConfig.Branch,
		"--serverName", gitopsConfig.ServerName}

	if delete {
		return "", nil
		//args = append(args, "--delete")
	}

	if len(gitopsConfig.CaCert) > 0 {
		args = append(args, "--caCert", gitopsConfig.CaCert)
	}
	if len(gitopsConfig.GitopsNamespace) > 0 {
		args = append(args, "--gitopsNamespace", gitopsConfig.GitopsNamespace)
	}
	if len(gitopsConfig.Debug) > 0 {
		args = append(args, "--debug", gitopsConfig.Debug)
	}

	cmd := exec.Command("igc", args...)
	cmd.Path = pathWithBinDir(binDir)

	tflog.Debug(ctx, "Executing command: "+cmd.String())
	tflog.Debug(ctx, "  Command path: "+cmd.Path)

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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	// start the command after having set up the pipe
	if err := cmd.Start(); err != nil {
		return "", err
	}

	// read command's stdout line by line
	in := bufio.NewScanner(stdout)
	inErr := bufio.NewScanner(stderr)

	for in.Scan() {
		if gitopsConfig.Debug == "true" {
			tflog.Debug(ctx, in.Text())
		} else {
			tflog.Info(ctx, in.Text())
		}
	}

	for inErr.Scan() {
		tflog.Error(ctx, inErr.Text())
	}

	if err := cmd.Wait(); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Error running command: %s", fmt.Sprintln(err)))
		return "", err
	}

	if err := in.Err(); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Error processing stream: %s", fmt.Sprintln(err)))
		return "", err
	}

	var id string
	if delete {
		id = ""
	} else {
		id = uuid.New().String()
	}

	return id, nil
}
