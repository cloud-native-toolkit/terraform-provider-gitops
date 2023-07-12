package gitops

import (
	"bufio"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/exec"
)

func resourceGitopsNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsNamespaceCreate,
		ReadContext:   resourceGitopsNamespaceRead,
		UpdateContext: resourceGitopsNamespaceUpdate,
		DeleteContext: resourceGitopsNamespaceDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"content_dir": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"server_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},
			"create_operator_group": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"argocd_namespace": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "openshift-gitops",
			},
			"dev_namespace": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"tmp_dir": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  ".tmp/namespace",
			},
			"branch": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "main",
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

type GitopsConfigValues struct {
	Create              bool   `yaml:"create"`
	ApplicationBasePath string `yaml:"applicationBasePath"`
	Host                string `yaml:"host"`
	Org                 string `yaml:"org"`
	Repo                string `yaml:"repo"`
	Branch              string `yaml:"branch"`
}

type NamespaceValues struct {
	CreateOperatorGroup bool               `yaml:"createOperatorGroup"`
	ArgocdNamespace     string             `yaml:"argocdNamespace"`
	GitopsConfig        GitopsConfigValues `yaml:"gitopsConfig"`
}

func resourceGitopsNamespaceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	fmt.Printf("Creating gitops namespace")

	config := m.(*ProviderConfig)

	name := d.Get("name").(string)
	contentDir := d.Get("content_dir").(string)
	serverName := d.Get("server_name").(string)
	branch := d.Get("branch").(string)
	valueFiles := d.Get("value_files").(string)
	credentials := d.Get("credentials").(string)
	gitopsConfig := d.Get("config").(string)

	createOperatorGroup := d.Get("create_operator_group").(bool)
	argocdNamespace := d.Get("argocd_namespace").(string)
	//devNamespace := d.Get("dev_namespace").(bool)
	tmpDir := d.Get("tmp_dir").(string)

	namespaceValues := NamespaceValues{
		CreateOperatorGroup: createOperatorGroup,
		ArgocdNamespace:     argocdNamespace,
		GitopsConfig: GitopsConfigValues{
			Create: false,
		},
	}

	valueData, err := yaml.Marshal(&namespaceValues)
	if err != nil {
		return diag.FromErr(err)
	}

	valuesPath := fmt.Sprintf("%s/namespace/%s", tmpDir, name)
	valuesFile := fmt.Sprintf("%s/values.yaml", valuesPath)

	binDir := config.BinDir
	lock := config.Lock
	debug := config.Debug
	caCert := config.GitConfig.CaCertFile

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Provisioning gitops namespace: name=%s, serverName=%s", name, serverName))

	err = os.MkdirAll(valuesPath, os.ModePerm)
	if err != nil {
		return diag.FromErr(err)
	}
	err = os.WriteFile(valuesFile, valueData, 0644)
	if err != nil {
		return diag.FromErr(err)
	}

	var args = []string{
		"gitops-namespace",
		name,
		"--branch", branch,
		"--serverName", serverName}

	if len(contentDir) > 0 {
		args = append(args, "--contentDir", contentDir)

		if len(valueFiles) > 0 {
			args = append(args, "--valueFiles", valueFiles)
		}
	} else {
		args = append(args,
			"--helmRepoUrl", "https://charts.cloudnativetoolkit.dev",
			"--helmChart", "namespace",
			"--helmChartVersion", "0.2.0",
			"--valueFiles", valuesFile)
	}

	if len(lock) > 0 {
		args = append(args, "--lock", lock)
	}
	if len(caCert) > 0 {
		args = append(args, "--caCert", caCert)
	}
	if len(debug) > 0 {
		args = append(args, "--debug", debug)
	}

	cmd := exec.Command("igc", args...)
	cmd.Path = pathWithBinDir(binDir)

	tflog.Debug(ctx, "Executing command: "+cmd.String())

	gitEmail := "cloudnativetoolkit@gmail.com"
	gitName := "Cloud Native Toolkit"

	updatedEnv := append(os.Environ(), "GIT_CREDENTIALS="+credentials)
	updatedEnv = append(updatedEnv, "GITOPS_CONFIG="+gitopsConfig)
	updatedEnv = append(updatedEnv, "EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_NAME="+gitName)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_NAME="+gitName)

	logEnvironment(ctx, &updatedEnv)

	cmd.Env = updatedEnv

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return diag.FromErr(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return diag.FromErr(err)
	}

	// start the command after having set up the pipe
	if err := cmd.Start(); err != nil {
		return diag.FromErr(err)
	}

	// read command's stdout line by line
	in := bufio.NewScanner(stdout)
	inErr := bufio.NewScanner(stderr)

	for in.Scan() {
		if debug == "true" {
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
		return diag.FromErr(err)
	}

	if err := in.Err(); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Error processing stream: %s", fmt.Sprintln(err)))
		return diag.FromErr(err)
	}

	d.SetId(name + ":" + serverName + ":" + contentDir)

	return diags
}

func resourceGitopsNamespaceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	log.Printf("Reading gitops-namespace")

	return diags
}

func resourceGitopsNamespaceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	log.Printf("Updating gitops-namespace")

	return resourceGitopsNamespaceRead(ctx, d, m)
}

func resourceGitopsNamespaceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	binDir := config.BinDir
	lock := config.Lock
	debug := config.Debug
	caCert := config.GitConfig.CaCertFile

	username := "cloudnativetoolkit"

	name := d.Get("name").(string)
	serverName := d.Get("server_name").(string)
	contentDir := d.Get("content_dir").(string)
	branch := d.Get("branch").(string)
	valueFiles := d.Get("value_files").(string)
	credentials := d.Get("credentials").(string)
	gitopsConfig := d.Get("config").(string)

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Destroying gitops namespace: name=%s, serverName=%s", name, serverName))

	var args = []string{
		"gitops-namespace",
		name,
		"--delete",
		"--contentDir", contentDir,
		"--branch", branch,
		"--serverName", serverName}

	if len(lock) > 0 {
		args = append(args, "--lock", lock)
	}
	if len(valueFiles) > 0 {
		args = append(args, "--valueFiles", valueFiles)
	}
	if len(caCert) > 0 {
		args = append(args, "--caCert", caCert)
	}
	if len(debug) > 0 {
		args = append(args, "--debug", debug)
	}

	cmd := exec.Command("igc", args...)
	cmd.Path = pathWithBinDir(binDir)

	tflog.Debug(ctx, "Executing command: "+cmd.String())

	gitEmail := "cloudnativetoolkit@gmail.com"
	gitName := "Cloud Native Toolkit"

	updatedEnv := append(os.Environ(), "GIT_CREDENTIALS="+credentials)
	updatedEnv = append(updatedEnv, "GITOPS_CONFIG="+gitopsConfig)
	updatedEnv = append(updatedEnv, "EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_NAME="+gitName)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_NAME="+gitName)

	logEnvironment(ctx, &updatedEnv)

	cmd.Env = updatedEnv

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return diag.FromErr(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return diag.FromErr(err)
	}

	// start the command after having set up the pipe
	if err := cmd.Start(); err != nil {
		return diag.FromErr(err)
	}

	// read command's stdout line by line
	in := bufio.NewScanner(stdout)
	inErr := bufio.NewScanner(stderr)

	for in.Scan() {
		if debug == "true" {
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
		return diag.FromErr(err)
	}

	if err := in.Err(); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Error processing stream: %s", fmt.Sprintln(err)))
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}
