package gitops

import (
	context "context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mutexkv "terraform-provider-gitops/mutex"
	"os"
	"path/filepath"
	b64 "encoding/base64"
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
			"gitops_namespace": resourceGitopsNamespace(),
			"gitops_module":    resourceGitopsModule(),
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
	    // write contents of caCert to caCertFile
   	    basePath, err := os.Getwd()
        if err != nil {
            return diag.FromErr(err)
        }

	    caCertFile = filepath.Join(basePath, "git-ca.crt")

        decodedCaCert, err := b64.StdEncoding.DecodeString(caCert)
        if err != nil {
            return diag.FromErr(err)
        }

        d1 := []byte(decodedCaCert)
        err = os.WriteFile(caCertFile, d1, 0666)
        if err != nil {
            return diag.FromErr(err)
        }
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
