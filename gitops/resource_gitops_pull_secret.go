package gitops

import (
	"bufio"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

func resourceGitopsPullSecret() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsPullSecretCreate,
		ReadContext:   resourceGitopsPullSecretRead,
		UpdateContext: resourceGitopsPullSecretUpdate,
		DeleteContext: resourceGitopsPullSecretDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": {
				Type:     schema.TypeString,
				Required: true,
			},
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
			"layer": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"infrastructure", "services", "applications"}, false),
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "base",
				ValidateFunc: validation.StringInSlice([]string{"base", "instances", "operators"}, false),
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
			"kubeseal_cert": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The certificate that will be used to encrypt the secret with kubeseal",
			},
			"registry_server": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The host name of the server that will be stored in the pull secret",
			},
			"registry_username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The username to the container registry that will be stored in the pull secret",
			},
			"registry_password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "The password to the container registry that will be stored in the pull secret",
			},
			"secret_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The name of the secret that will be created. If not provided the module name will be used",
			},
			"tmp_dir": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
		},
	}
}

type PullSecretConfig struct {
	Name      string
	Namespace string
	Server    string
	Username  string
	Password  string
}

func resourceGitopsPullSecretCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	namespace := getNamespaceInput(d)
	name := getNameInput(d)

	cert := d.Get("kubeseal_cert").(string)
	secretName := d.Get("secret_name").(string)
	tmpDir := d.Get("tmp_dir").(string)
	if len(tmpDir) == 0 {
		tmpDir = fmt.Sprintf(".tmp/pull_secret/%s/%s", namespace, name)
	}

	binDir := config.BinDir

	secretDir := path.Join(tmpDir, name, "secrets")
	contentDir := path.Join(tmpDir, name, "sealed-secrets")

	var pullSecretName string
	if len(secretName) > 0 {
		pullSecretName = secretName
	} else {
		pullSecretName = name
	}

	pullSecretConfig := PullSecretConfig{
		Name:      pullSecretName,
		Namespace: namespace,
		Server:    d.Get("registry_server").(string),
		Username:  d.Get("registry_username").(string),
		Password:  d.Get("registry_password").(string),
	}

	// create secret in secretDir
	secretFile, err := createSecret(ctx, binDir, secretDir, "pull-secret.yaml", pullSecretConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = encryptWithCert(ctx, binDir, tmpDir, secretDir, contentDir, secretFile, cert)
	if err != nil {
		return diag.FromErr(err)
	}

	moduleConfig := GitopsModuleConfig{
		Name:        name,
		Namespace:   namespace,
		Branch:      getBranchInput(d),
		ServerName:  getServerNameInput(d),
		Layer:       getLayerInput(d),
		Type:        getTypeInput(d),
		ContentDir:  contentDir,
		CaCert:      config.GitConfig.CaCertFile,
		Debug:       config.Debug,
		Credentials: getCredentialsInput(d),
		Config:      getGitopsConfigInput(d),
	}

	id, err := populateGitopsModule(ctx, binDir, moduleConfig, false)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

func resourceGitopsPullSecretRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	return diags
}

func resourceGitopsPullSecretUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceGitopsPullSecretRead(ctx, d, m)
}

func resourceGitopsPullSecretDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	name := getNameInput(d)
	namespace := getNamespaceInput(d)

	cert := d.Get("kubeseal_cert").(string)
	secretName := d.Get("secret_name").(string)
	tmpDir := d.Get("tmp_dir").(string)
	if len(tmpDir) == 0 {
		tmpDir = fmt.Sprintf(".tmp/pull_secret/%s/%s", namespace, name)
	}

	binDir := config.BinDir

	secretDir := path.Join(tmpDir, name, "secrets")
	contentDir := path.Join(tmpDir, name, "sealed-secrets")

	var pullSecretName string
	if len(secretName) > 0 {
		pullSecretName = secretName
	} else {
		pullSecretName = name
	}

	pullSecretConfig := PullSecretConfig{
		Name:      pullSecretName,
		Namespace: namespace,
		Server:    d.Get("registry_server").(string),
		Username:  d.Get("registry_username").(string),
		Password:  d.Get("registry_password").(string),
	}

	// create secret in secretDir
	secretFile, err := createSecret(ctx, binDir, secretDir, "pull-secret.yaml", pullSecretConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = encryptWithCert(ctx, binDir, tmpDir, secretDir, contentDir, secretFile, cert)
	if err != nil {
		return diag.FromErr(err)
	}

	moduleConfig := GitopsModuleConfig{
		Name:        name,
		Namespace:   namespace,
		Branch:      getBranchInput(d),
		ServerName:  getServerNameInput(d),
		Layer:       getLayerInput(d),
		Type:        getTypeInput(d),
		ContentDir:  contentDir,
		CaCert:      config.GitConfig.CaCertFile,
		Debug:       config.Debug,
		Credentials: getCredentialsInput(d),
		Config:      getGitopsConfigInput(d),
	}

	id, err := populateGitopsModule(ctx, binDir, moduleConfig, true)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

func createSecret(ctx context.Context, binDir string, destDir string, fileName string, secretData PullSecretConfig) (string, error) {

	args := []string{
		"create",
		"secret",
		"docker-registry",
		secretData.Name,
		"--namespace", secretData.Namespace,
		"--docker-server=" + secretData.Server,
		"--docker-username=" + secretData.Username,
		"--docker-password=" + secretData.Password,
		"--dry-run=client",
		"--output=json"}

	cmd := exec.Command(filepath.Join(binDir, "kubectl"), args...)
	tflog.Debug(ctx, "Executing command: "+cmd.String())

	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return "", err
	}
	destFile := path.Join(destDir, fileName)

	outfilePipeIn, err := os.Create(destFile)
	if err != nil {
		return "", err
	}
	defer func() {
		if tmpErr := outfilePipeIn.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()

	cmd.Stdout = outfilePipeIn

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	inErr := bufio.NewScanner(stderr)
	for inErr.Scan() {
		tflog.Error(ctx, inErr.Text())
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	return fileName, err
}
