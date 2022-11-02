package gitops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
)

func resourceGitopsRepo() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsRepoCreate,
		ReadContext:   resourceGitopsRepoRead,
		UpdateContext: resourceGitopsRepoUpdate,
		DeleteContext: resourceGitopsRepoDelete,
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The host name of the git server.",
			},
			"org": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The org/group where the git repository exists/will be provisioned.",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The project that will be used for the git repo.",
				Default:     "",
			},
			"repo": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The short name of the repository (i.e. the part after the org/group name).",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The username of the user with access to the repository.",
			},
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "The token/password used to authenticate the user to the git server.",
			},
			"branch": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The project that will be used for the git repo.",
				Default:     "main",
			},
			"server_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the cluster that will be configured via gitops.",
				Default:     "default",
			},
			"gitops_namespace": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The namespace where ArgoCD is running in the cluster.",
				Default:     "openshift-gitops",
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
			"sealed_secrets_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The certificate/public key used to encrypt the sealed secrets.",
				Default:     "",
			},
			"public": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag indicating that the repo should be public or private.",
				Default:     false,
			},
			"strict": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag indicating that an error should be thrown if the repo already exists.",
				Default:     false,
			},
			"tmp_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The temporary directory where git repo changes will be staged.",
				Default:     ".tmp/gitops-init",
			},
			"created": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Flag indicating the repo was created.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The url of the created repository.",
			},
			"repo_slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The repo slug of the created repository (i.e. url without the protocol).",
			},
			"gitops_config": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The configuration of the gitops repo(s) in json format",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"layer": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The layer for the configuration (bootstrap, infrastructure, services, or application)",
						},
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of the configuration (argocd or payload).",
						},
						"repo": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The repo slug with the git host",
						},
						"url": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The url to the git repository",
						},
						"path": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The path to the configuration in the git repository",
						},
						"project": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The project name for the ArgoCD application",
						},
					},
				},
			},
			"git_credentials": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The git credentials for the gitops repo(s) in json format",
				Sensitive:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"repo": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The repo slug with the git host",
						},
						"url": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The url to the git repository",
						},
						"username": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The username for the git repository",
						},
						"token": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The token for the git repository",
						},
					},
				},
			},
		},
	}
}

type GitopsRepoConfig struct {
	Host              string `yaml:"host"`
	Org               string `yaml:"org"`
	Project           string `yaml:"project,omitempty"`
	Repo              string `yaml:"repo"`
	Username          string `yaml:"username"`
	Token             string `yaml:"token"`
	Branch            string `yaml:"branch"`
	ServerName        string `yaml:"server_name"`
	GitopsNamespace   string `yaml:"gitops_namespace"`
	CaCertFile        string `yaml:"ca_cert_file"`
	SealedSecretsCert string `yaml:"sealed_secrets_cert"`
	Public            bool   `yaml:"public"`
	Strict            bool   `yaml:"strict"`
	TmpDir            string `yaml:"tmp_dir"`
	BinDir            string `yaml:"bin_dir"`
	Debug             bool   `yaml:"debug"`
}

type ArgocdConfig struct {
	Project string `yaml:"project" json:"project"`
	Repo    string `yaml:"repo" json:"repo"`
	Url     string `yaml:"url" json:"url"`
	Path    string `yaml:"path" json:"path"`
}

type PayloadConfig struct {
	Repo string `yaml:"repo" json:"repo"`
	Url  string `yaml:"url" json:"url"`
	Path string `yaml:"path" json:"path"`
}

type GitopsConfigEntry struct {
	Layer   string `yaml:"layer" json:"layer"`
	Type    string `yaml:"type" json:"type"`
	Project string `yaml:"project,omitempty" json:"project,omitempty"`
	Repo    string `yaml:"repo" json:"repo"`
	Url     string `yaml:"url" json:"url"`
	Path    string `yaml:"path" json:"path"`
}

type BootstrapConfig struct {
	ArgocdConfig ArgocdConfig `yaml:"argocd-config" json:"argocd-config"`
}

type LayerConfig struct {
	ArgocdConfig ArgocdConfig  `yaml:"argocd-config" json:"argocd-config"`
	Payload      PayloadConfig `yaml:"payload" json:"payload"`
}

type GitopsConfigResult struct {
	Bootstrap      BootstrapConfig `yaml:"bootstrap" json:"bootstrap"`
	Infrastructure LayerConfig     `yaml:"infrastructure" json:"infrastructure"`
	Services       LayerConfig     `yaml:"services" json:"services"`
	Applications   LayerConfig     `yaml:"applications" json:"applications"`
}

type GitopsRepoResult struct {
	Url          string             `yaml:"url" json:"url"`
	Repo         string             `yaml:"repo" json:"repo"`
	Created      bool               `yaml:"created" json:"created"`
	Initialized  bool               `yaml:"initialized" json:"initialized"`
	GitopsConfig GitopsConfigResult `yaml:"gitops_config" json:"gitops_config"`
	KubesealCert string             `yaml:"kubeseal_cert" json:"kubeseal_cert"`
}

type GitCredential struct {
	Repo     string `yaml:"repo" json:"repo"`
	Url      string `yaml:"url" json:"url"`
	Username string `yaml:"username" json:"username"`
	Token    string `yaml:"token" json:"token"`
}

func resourceGitopsRepoCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	tflog.Info(ctx, "Creating gitops repo")

	config := m.(*ProviderConfig)

	caCertFile := d.Get("ca_cert_file").(string)
	if len(caCertFile) == 0 {
		caCertFile = config.CaCertFile
	}

	gitopsRepoConfig := GitopsRepoConfig{
		Host:              d.Get("host").(string),
		Org:               d.Get("org").(string),
		Project:           d.Get("project").(string),
		Repo:              d.Get("repo").(string),
		Username:          d.Get("username").(string),
		Token:             d.Get("token").(string),
		Branch:            d.Get("branch").(string),
		ServerName:        d.Get("server_name").(string),
		GitopsNamespace:   d.Get("gitops_namespace").(string),
		CaCertFile:        caCertFile,
		SealedSecretsCert: d.Get("sealed_secrets_cert").(string),
		Public:            d.Get("public").(bool),
		Strict:            d.Get("strict").(bool),
		TmpDir:            d.Get("tmp_dir").(string),
		BinDir:            config.BinDir,
	}

	result, err := processGitopsRepo(ctx, gitopsRepoConfig, false)
	if err != nil {
		return diag.FromErr(err)
	}

	suffix := randStringBytes(16)

	var id string
	if len(gitopsRepoConfig.Project) > 0 {
		id = fmt.Sprintf("%s/%s/%s/%s:%s", gitopsRepoConfig.Host, gitopsRepoConfig.Org, gitopsRepoConfig.Project, gitopsRepoConfig.Repo, suffix)
	} else {
		id = fmt.Sprintf("%s/%s/%s:%s", gitopsRepoConfig.Host, gitopsRepoConfig.Org, gitopsRepoConfig.Repo, suffix)
	}

	tflog.Debug(ctx, fmt.Sprintf("Create result: %t, %s", result.Created, result.Url))

	err = d.Set("created", result.Created)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("url", result.Url)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("repo_slug", result.Repo)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("sealed_secrets_cert", result.KubesealCert)
	if err != nil {
		return diag.FromErr(err)
	}

	gitopsConfigEntries := gitopsConfigToConfigEntries(result.GitopsConfig)

	gitopsConfigEntriesJson, err := toJson(gitopsConfigEntries)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("gitops_config", gitopsConfigEntriesJson)
	if err != nil {
		return diag.FromErr(err)
	}

	gitCredential := []GitCredential{{
		Url:      result.Url,
		Repo:     result.Repo,
		Username: gitopsRepoConfig.Username,
		Token:    gitopsRepoConfig.Token,
	}}
	gitCredentialJson, err := toJson(gitCredential)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("git_credentials", gitCredentialJson)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

func resourceGitopsRepoRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	tflog.Info(ctx, "Reading gitops-repo")

	return diags
}

func resourceGitopsRepoUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	tflog.Info(ctx, "Updating gitops-repo")

	return resourceGitopsRepoRead(ctx, d, m)
}

func resourceGitopsRepoDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	created := d.Get("created").(bool)
	if !created {
		tflog.Info(ctx, "Repository not created by this resource. Skipping delete")

		d.SetId("")
		return diags
	}

	tflog.Info(ctx, "Deleting gitops repo")

	caCertFile := d.Get("ca_cert_file").(string)
	if len(caCertFile) == 0 {
		caCertFile = config.CaCertFile
	}

	gitopsRepoConfig := GitopsRepoConfig{
		Host:              d.Get("host").(string),
		Org:               d.Get("org").(string),
		Project:           d.Get("project").(string),
		Repo:              d.Get("repo").(string),
		Username:          d.Get("username").(string),
		Token:             d.Get("token").(string),
		Branch:            d.Get("branch").(string),
		ServerName:        d.Get("server_name").(string),
		CaCertFile:        caCertFile,
		SealedSecretsCert: d.Get("sealed_secrets_cert").(string),
		Public:            d.Get("public").(bool),
		Strict:            d.Get("strict").(bool),
		TmpDir:            d.Get("tmp_dir").(string),
		BinDir:            config.BinDir,
	}

	_, err := processGitopsRepo(ctx, gitopsRepoConfig, true)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func processGitopsRepo(ctx context.Context, config GitopsRepoConfig, delete bool) (*GitopsRepoResult, error) {

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Provisioning gitops repo: host=%s, org=%s, project=%s, repo=%s", config.Host, config.Org, config.Project, config.Repo))

	var args = []string{
		"gitops-init",
		config.Repo,
		"--output", "json",
		"--host", config.Host,
		"--org", config.Org,
		"--branch", config.Branch,
		"--serverName", config.ServerName,
		"--tmpDir", config.TmpDir,
	}

	if len(config.Project) > 0 {
		args = append(args, "--project", config.Project)
	}

	if len(config.CaCertFile) > 0 {
		args = append(args, "--caCertFile", config.CaCertFile)
	}

	if config.Public {
		args = append(args, "--public", "true")
	}

	if config.Strict {
		args = append(args, "--strict", "true")
	}

	if delete {
		args = append(args, "--delete")
	}

	cmd := exec.Command(filepath.Join(config.BinDir, "igc"), args...)

	tflog.Debug(ctx, "Executing command: "+cmd.String())

	envNames := []string{"GIT_USERNAME", "GIT_TOKEN"}

	updatedEnv := append(os.Environ(), "GIT_USERNAME="+config.Username)
	updatedEnv = append(updatedEnv, "GIT_TOKEN="+config.Token)
	if len(config.SealedSecretsCert) > 0 {
		envNames = append(envNames, "KUBESEAL_CERT")
		updatedEnv = append(updatedEnv, "KUBESEAL_CERT="+config.SealedSecretsCert)
	}

	tflog.Debug(ctx, fmt.Sprintf("Environment variables set: %s", envNames))
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

	errText := errb.String()
	if len(errText) > 0 {
		tflog.Error(ctx, fmt.Sprintf("Command error log: %s", errText))
	}

	repoResult := GitopsRepoResult{}
	err := json.Unmarshal(outb.Bytes(), &repoResult)
	if err != nil {
		return nil, err
	}

	return &repoResult, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)

	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

func toJson(value interface{}) (string, error) {
	result, err := json.Marshal(value)

	return string(result), err
}

func gitopsConfigToConfigEntries(value GitopsConfigResult) []GitopsConfigEntry {
	result := make([]GitopsConfigEntry, 7)

	result[0] = GitopsConfigEntry{
		Layer:   "bootstrap",
		Type:    "argocd",
		Project: value.Bootstrap.ArgocdConfig.Project,
		Repo:    value.Bootstrap.ArgocdConfig.Repo,
		Url:     value.Bootstrap.ArgocdConfig.Url,
		Path:    value.Bootstrap.ArgocdConfig.Path,
	}
	result[1] = GitopsConfigEntry{
		Layer:   "infrastucture",
		Type:    "argocd",
		Project: value.Infrastructure.ArgocdConfig.Project,
		Repo:    value.Infrastructure.ArgocdConfig.Repo,
		Url:     value.Infrastructure.ArgocdConfig.Url,
		Path:    value.Infrastructure.ArgocdConfig.Path,
	}
	result[2] = GitopsConfigEntry{
		Layer: "infrastucture",
		Type:  "payload",
		Repo:  value.Infrastructure.Payload.Repo,
		Url:   value.Infrastructure.Payload.Url,
		Path:  value.Infrastructure.Payload.Path,
	}
	result[3] = GitopsConfigEntry{
		Layer:   "services",
		Type:    "argocd",
		Project: value.Services.ArgocdConfig.Project,
		Repo:    value.Services.ArgocdConfig.Repo,
		Url:     value.Services.ArgocdConfig.Url,
		Path:    value.Services.ArgocdConfig.Path,
	}
	result[4] = GitopsConfigEntry{
		Layer: "services",
		Type:  "payload",
		Repo:  value.Services.Payload.Repo,
		Url:   value.Services.Payload.Url,
		Path:  value.Services.Payload.Path,
	}
	result[5] = GitopsConfigEntry{
		Layer:   "applications",
		Type:    "argocd",
		Project: value.Applications.ArgocdConfig.Project,
		Repo:    value.Applications.ArgocdConfig.Repo,
		Url:     value.Applications.ArgocdConfig.Url,
		Path:    value.Applications.ArgocdConfig.Path,
	}
	result[6] = GitopsConfigEntry{
		Layer: "applications",
		Type:  "payload",
		Repo:  value.Applications.Payload.Repo,
		Url:   value.Applications.Payload.Url,
		Path:  value.Applications.Payload.Path,
	}

	return result
}