package gitops

import (
	context "context"
	b64 "encoding/base64"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"os"
	"path/filepath"
	mutexkv "terraform-provider-gitops/mutex"
)

var gitopsMutexKV = mutexkv.NewMutexKV()

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"bin_dir": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The directory containing the igc binary used to interact with the gitops repo.",
			},
			"repo": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the repository in the org on the git server.",
				DefaultFunc: schema.EnvDefaultFunc("GIT_REPO", nil),
			},
			"branch": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the branch in the gitops repository where the config will be stored.",
				DefaultFunc: schema.EnvDefaultFunc("GIT_BRANCH", "main"),
			},
			"server_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the server the configuration with which the configuration will be associated.",
				Default:     "default",
			},
			"public": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag indicating that the repo should be public or private.",
				Default:     false,
			},
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The host name of the gitops repository (GitHub, Github Enterprise, Gitlab, Bitbucket, Azure DevOps, and Gitea servers are supported).",
				DefaultFunc: schema.EnvDefaultFunc("GIT_HOST", nil),
			},
			"org": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The organization on the git server where the repsitory will be located. If not provided the org will default to the username.",
				DefaultFunc: schema.EnvDefaultFunc("GIT_ORG", nil),
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Azure DevOps project in the git server. This value is only applied for Azure DevOps servers.",
				DefaultFunc: schema.EnvDefaultFunc("GIT_PROJECT", nil),
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The username used to access the git server.",
				DefaultFunc: schema.EnvDefaultFunc("GIT_USERNAME", nil),
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The token used to access the git server.",
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("GIT_TOKEN", nil),
			},
			"ca_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ca certificate used to sign the self-signed certificate used by the git server, if applicable.",
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_CA_CERT", ""),
			},
			"ca_cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The file containing the ca certificate used to sign the self-signed certificate used by the git server, if applicable.",
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_CA_CERT_FILE", ""),
			},
			"default_host": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default/fallback host name of the gitops repository (GitHub, Github Enterprise, Gitlab, Bitbucket, Azure DevOps, and Gitea servers are supported).",
				DefaultFunc: schema.EnvDefaultFunc("GIT_HOST", nil),
			},
			"default_org": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default/fallback organization on the git server where the repsitory will be located. If not provided the org will default to the username.",
				DefaultFunc: schema.EnvDefaultFunc("GIT_ORG", nil),
			},
			"default_project": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default/fallback Azure DevOps project in the git server. This value is only applied for Azure DevOps servers.",
				DefaultFunc: schema.EnvDefaultFunc("GIT_PROJECT", nil),
			},
			"default_username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default/fallback username used to access the git server.",
				DefaultFunc: schema.EnvDefaultFunc("GIT_USERNAME", nil),
			},
			"default_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default/fallback  token used to access the git server.",
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("GIT_TOKEN", nil),
			},
			"default_ca_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default/fallback ca certificate used to sign the self-signed certificate used by the git server, if applicable.",
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_CA_CERT", ""),
			},
			"default_ca_cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The default/fallback file containing the ca certificate used to sign the self-signed certificate used by the git server, if applicable.",
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_CA_CERT_FILE", ""),
			},
			"lock": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_LOCK", "branch"),
			},
			"debug": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_DEBUG", "false"),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"gitops_repo":            resourceGitopsRepo(),
			"gitops_namespace":       resourceGitopsNamespace(),
			"gitops_module":          resourceGitopsModule(),
			"gitops_service_account": resourceGitopsServiceAccount(),
			"gitops_seal_secrets":    resourceGitopsSealSecrets(),
			"gitops_pull_secret":     resourceGitopsPullSecret(),
			"gitops_metadata":        resourceGitopsMetadata(),
		},
		DataSourcesMap:       map[string]*schema.Resource{
			"gitops_repo_config": dataGitopsRepoConfig(),
			"gitops_metadata_cluster":  dataGitopsMetadataCluster(),
			"gitops_metadata_packages": dataGitopsMetadataPackages(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

type GitConfigValues struct {
	Host       string
	Org        string
	Project    string
	Username   string
	Token      string
	CaCertFile string
}

type ProviderConfig struct {
	BinDir     string
	GitConfig  *GitConfigValues
	Repo       string
	Branch     string
	ServerName string
	Public     bool
	Lock       string
	Debug      string
}

func createCaCertFile(caCert string, prefix string) (string, error) {
	// write contents of caCert to caCertFile
	basePath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	caCertFile := filepath.Join(basePath, "git-ca.crt")

	decodedCaCert, err := b64.StdEncoding.DecodeString(caCert)
	if err != nil {
		return "", err
	}

	d1 := []byte(decodedCaCert)
	err = os.WriteFile(caCertFile, d1, 0666)
	if err != nil {
		return "", err
	}

	return caCertFile, nil
}

func loadGitConfigValues(ctx context.Context, d *schema.ResourceData, prefix string) (*GitConfigValues, error) {
	host := d.Get(fmt.Sprintf("%shost", prefix)).(string)
	org := d.Get(fmt.Sprintf("%sorg", prefix)).(string)
	project := d.Get(fmt.Sprintf("%sproject", prefix)).(string)
	username := d.Get(fmt.Sprintf("%susername", prefix)).(string)
	token := d.Get(fmt.Sprintf("%stoken", prefix)).(string)
	caCert := d.Get(fmt.Sprintf("%sca_cert", prefix)).(string)
	caCertFile := d.Get(fmt.Sprintf("%sca_cert_file", prefix)).(string)

	if len(org) == 0 {
		org = username
	}

	if len(caCert) > 0 && len(caCertFile) == 0 {
		newCaCertFile, err := createCaCertFile(caCert, prefix)
		if err != nil {
			return nil, err
		}

		caCertFile = newCaCertFile
	}

	c := &GitConfigValues{
		Host:       host,
		Org:        org,
		Project:    project,
		Username:   username,
		Token:      token,
		CaCertFile: caCertFile,
	}

	return c, nil
}

func isValidGitConfig(config *GitConfigValues) bool {
	return len(config.Host) > 0 && len(config.Username) > 0 && len(config.Token) > 0
}

func getResourceValue(d *schema.ResourceData, name string, defaultValue string) string {
	value := d.Get(name).(string)

	if len(value) == 0 {
		return defaultValue
	}

	return value
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {

	tflog.Info(ctx, "Configuring GitOps provider")

	binDir := d.Get("bin_dir").(string)
	lock := d.Get("lock").(string)
	debug := d.Get("debug").(string)

	repo := getResourceValue(d, "repo", "")
	branch := getResourceValue(d, "branch", "main")
	serverName := getResourceValue(d, "server_name", "default")
	public := d.Get("public").(bool)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	gitConfig, err := loadGitConfigValues(ctx, d, "")
	if err != nil {
		tflog.Error(ctx, "Error loading config values", err)
		return nil, diag.FromErr(err)
	}

	if !isValidGitConfig(gitConfig) {
		gitConfig, err = loadGitConfigValues(ctx, d, "default_")
		if err != nil {
			tflog.Error(ctx, "Error loading default config values", err)
			return nil, diag.FromErr(err)
		}
	}

	ctx = tflog.With(ctx, "gitops_binDir", binDir)
	ctx = tflog.With(ctx, "gitops_repo", repo)
	ctx = tflog.With(ctx, "gitops_branch", branch)
	ctx = tflog.With(ctx, "gitops_serverName", serverName)
	ctx = tflog.With(ctx, "gitops_gitConfig", gitConfig)

	tflog.Debug(ctx, "Creating GitOps provider config")

	c := &ProviderConfig{
		BinDir:     binDir,
		GitConfig:  gitConfig,
		Repo:       repo,
		Branch:     branch,
		ServerName: serverName,
		Public:     public,
		Lock:       lock,
		Debug:      debug,
	}

	tflog.Info(ctx, "Configured Gitops provider", map[string]any{"success": true, "config": c})

	return c, diags
}
