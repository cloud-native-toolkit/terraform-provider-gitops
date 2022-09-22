package gitops

import (
	context "context"
	b64 "encoding/base64"
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
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GIT_USERNAME", nil),
			},
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("GIT_TOKEN", nil),
			},
			"bin_dir": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"lock": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_LOCK", "branch"),
			},
			"debug": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_DEBUG", "false"),
			},
			"ca_cert": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_CA_CERT", ""),
			},
			"ca_cert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITOPS_CA_CERT_FILE", ""),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"gitops_namespace":       resourceGitopsNamespace(),
			"gitops_module":          resourceGitopsModule(),
			"gitops_service_account": resourceGitopsServiceAccount(),
			"gitops_seal_secrets":    resourceGitopsSealSecrets(),
			"gitops_pull_secret":     resourceGitopsPullSecret(),
		},
		DataSourcesMap:       map[string]*schema.Resource{},
		ConfigureContextFunc: providerConfigure,
	}
}

type ProviderConfig struct {
	BinDir     string
	Username   string
	Token      string
	Lock       string
	Debug      string
	CaCertFile string
}

func createCaCertFile(caCert string) (string, error) {
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

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	binDir := d.Get("bin_dir").(string)
	username := d.Get("username").(string)
	token := d.Get("token").(string)
	lock := d.Get("lock").(string)
	debug := d.Get("debug").(string)
	caCert := d.Get("ca_cert").(string)
	caCertFile := d.Get("ca_cert_file").(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	if len(caCert) > 0 && len(caCertFile) == 0 {
		newCaCertFile, err := createCaCertFile(caCert)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		caCertFile = newCaCertFile
	}

	c := &ProviderConfig{
		BinDir:     binDir,
		Username:   username,
		Token:      token,
		Lock:       lock,
		Debug:      debug,
		CaCertFile: caCertFile,
	}

	return c, diags
}
