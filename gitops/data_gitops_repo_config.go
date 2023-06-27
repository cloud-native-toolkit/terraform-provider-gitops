package gitops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataGitopsRepoConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataGitopsRepoConfigRead,
		Schema: map[string]*schema.Schema{
			"server_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},
			"branch": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "main",
			},
			"bootstrap_url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The username of the user with access to the repository.",
				Default:     "",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "The token/password used to authenticate the user to the git server.",
				Default:     "",
			},
			"ca_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ca certificate for SSL connections.",
				Default:     "",
			},
			"ca_cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the file containing the ca certificate for SSL connections.",
				Default:     "",
			},
			"gitops_config": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The configuration of the gitops repo(s) in json format",
			},
			"git_credentials": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The git credentials for the gitops repo(s) in json format",
				Sensitive:   true,
			},
		},
	}
}

type GitopsRepoReadConfig struct {
	ServerName   string
	Branch       string
	BootstrapUrl string
	Username     string
	Token        string
	CaCert       string
	CaCertFile   string
	Debug        string
	BinDir       string
}

func dataGitopsRepoConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	binDir := config.BinDir

	repoReadConfig := GitopsRepoReadConfig{
		ServerName:   getServerNameInput(d),
		Branch:       getBranchInput(d),
		BootstrapUrl: d.Get("bootstrap_url").(string),
		Username:     d.Get("username").(string),
		Token:        d.Get("token").(string),
		CaCert:       d.Get("ca_cert").(string),
		CaCertFile:   d.Get("ca_cert_file").(string),
		BinDir:       binDir,
		Debug:        config.Debug,
	}

	result, err := lookupGitopRepoConfig(ctx, &repoReadConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	gitopsConfigJson, err := toJson(result)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("gitops_config", gitopsConfigJson)
	if err != nil {
		return diag.FromErr(err)
	}

	gitCredential := []GitCredential{{
		Url:      "*",
		Repo:     "*",
		Username: repoReadConfig.Username,
		Token:    repoReadConfig.Token,
	}}
	gitCredentialJson, err := toJson(gitCredential)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("git_credentials", gitCredentialJson)
	if err != nil {
		return diag.FromErr(err)
	}

	id := uuid.New().String()
	d.SetId(id)

	return diags
}

func lookupGitopRepoConfig(ctx context.Context, input *GitopsRepoReadConfig) (*GitopsConfigResult, error) {

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Retrieving gitops metadata: serverName=%s", input.ServerName))

	var args = []string{
		"gitops-config",
		input.BootstrapUrl,
		"--branch", input.Branch,
		"--serverName", input.ServerName,
		"--output", "jsonfile=./output.json"}

	if len(input.CaCert) > 0 {
		args = append(args, "--caCert", input.CaCert)
	}
	if len(input.Debug) > 0 {
		args = append(args, "--debug", input.Debug)
	}

	cmd := exec.Command(filepath.Join(input.BinDir, "igc"), args...)

	tflog.Debug(ctx, "Executing command: "+cmd.String())

	gitEmail := "cloudnativetoolkit@gmail.com"
	gitName := "Cloud Native Toolkit"

	updatedEnv := append(os.Environ(), "GIT_USERNAME="+input.Username)
	updatedEnv = append(updatedEnv, "GIT_TOKEN="+input.Token)
	updatedEnv = append(updatedEnv, "EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_NAME="+gitName)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_NAME="+gitName)

	tflog.Debug(ctx, fmt.Sprintf("Environment: %v", updatedEnv))

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

	tflog.Debug(ctx, fmt.Sprintf("JSON result from gitops config: %s", string(dat)))

	gitopsConfig := GitopsConfigResult{}
	err = json.Unmarshal(dat, &gitopsConfig)
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx,"Result values from gitops config")

	return &gitopsConfig, nil
}
