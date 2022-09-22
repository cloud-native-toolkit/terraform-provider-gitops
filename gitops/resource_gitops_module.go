package gitops

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"os"
	"os/exec"
	"path/filepath"
)

func resourceGitopsModule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsModuleCreate,
		ReadContext:   resourceGitopsModuleRead,
		UpdateContext: resourceGitopsModuleUpdate,
		DeleteContext: resourceGitopsModuleDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"content_dir": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"helm_repo_url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"helm_chart": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"helm_chart_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"server_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},
			"branch": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "main",
			},
			"layer": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"infrastructure", "services", "applications"}, false),
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "base",
				ValidateFunc: validation.StringInSlice([]string{"base", "instances", "operators"}, false),
			},
			"value_files": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"credentials": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"config": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceGitopsModuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	moduleConfig := GitopsModuleConfig{
		Name:        getNameInput(d),
		Namespace:   getNamespaceInput(d),
		Branch:      getBranchInput(d),
		ServerName:  getServerNameInput(d),
		Layer:       getLayerInput(d),
		Type:        getTypeInput(d),
		ContentDir:  getContentDirInput(d),
		ValueFiles:  getValueFilesInput(d),
		CaCert:      config.CaCertFile,
		Debug:       config.Debug,
		Credentials: getCredentialsInput(d),
		Config:      getGitopsConfigInput(d),
		HelmConfig:  helmConfigFromResourceData(d),
	}

	id, err := populateGitopsModule(ctx, config.BinDir, moduleConfig, false)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

func resourceGitopsModuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	return diags
}

func resourceGitopsModuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceGitopsModuleRead(ctx, d, m)
}

func resourceGitopsModuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	moduleConfig := GitopsModuleConfig{
		Name:        getNameInput(d),
		Namespace:   getNamespaceInput(d),
		Branch:      getBranchInput(d),
		ServerName:  getServerNameInput(d),
		Layer:       getLayerInput(d),
		Type:        getTypeInput(d),
		ContentDir:  getContentDirInput(d),
		ValueFiles:  getValueFilesInput(d),
		CaCert:      config.CaCertFile,
		Debug:       config.Debug,
		Credentials: getCredentialsInput(d),
		Config:      getGitopsConfigInput(d),
		HelmConfig:  helmConfigFromResourceData(d),
	}

	id, err := populateGitopsModule(ctx, config.BinDir, moduleConfig, true)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

func populateGitopsModule(ctx context.Context, binDir string, gitopsConfig GitopsModuleConfig, delete bool) (string, error) {

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Provisioning gitops module: name=%s, namespace=%s, serverName=%s", gitopsConfig.Name, gitopsConfig.Namespace, gitopsConfig.ServerName))

	var args = []string{
		"gitops-module",
		gitopsConfig.Name,
		"-n", gitopsConfig.Namespace,
		"--branch", gitopsConfig.Branch,
		"--serverName", gitopsConfig.ServerName,
		"--layer", gitopsConfig.Layer,
		"--type", gitopsConfig.Type}

	if delete {
		args = append(args, "--delete")
	}

	if len(gitopsConfig.ContentDir) > 0 {
		args = append(args, "--contentDir", gitopsConfig.ContentDir)
	} else if gitopsConfig.HelmConfig != nil {
		helmConfig := *gitopsConfig.HelmConfig

		args = append(args,
			"--helmRepoUrl", helmConfig.RepoUrl,
			"--helmChart", helmConfig.Chart,
			"--helmChartVersion", helmConfig.ChartVersion)
	} else {
		return "", errors.New("contentDir or helmRepoUrl, helmChart, and helmChartVersion are required")
	}

	if len(gitopsConfig.ValueFiles) > 0 {
		args = append(args, "--valueFiles", gitopsConfig.ValueFiles)
	}
	if len(gitopsConfig.CaCert) > 0 {
		args = append(args, "--caCert", gitopsConfig.CaCert)
	}
	if len(gitopsConfig.Debug) > 0 {
		args = append(args, "--debug", gitopsConfig.Debug)
	}

	cmd := exec.Command(filepath.Join(binDir, "igc"), args...)

	tflog.Debug(ctx, "Executing command: "+cmd.String())

	gitEmail := "cloudnativetoolkit@gmail.com"
	gitName := "Cloud Native Toolkit"

	updatedEnv := append(os.Environ(), "GIT_CREDENTIALS="+gitopsConfig.Credentials)
	updatedEnv = append(updatedEnv, "GITOPS_CONFIG="+gitopsConfig.Config)
	updatedEnv = append(updatedEnv, "EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_NAME="+gitName)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_NAME="+gitName)

	tflog.Debug(ctx, fmt.Sprintf("Environment: %v", updatedEnv))

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
		id = gitopsConfig.Namespace + ":" + gitopsConfig.Name + ":" + gitopsConfig.ServerName + ":" + gitopsConfig.Layer + ":" + gitopsConfig.Type
	}

	return id, nil
}
