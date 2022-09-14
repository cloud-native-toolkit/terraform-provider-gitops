package gitops

import (
	"bufio"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
)

func resourceGitopsRBAC() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsRBACCreate,
		ReadContext:   resourceGitopsRBACRead,
		UpdateContext: resourceGitopsRBACUpdate,
		DeleteContext: resourceGitopsRBACDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"infrastructure", "services", "applications"}, false),
				Default:      "infrastructure",
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
			"cluster_scope": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag setting the RBAC rules at cluster scope vs namespace scope",
			},
			"service_account_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the service account that should be bound to the role. If not provided defaults to all service accounts in the namespace",
				Default:     "",
			},
			"service_account_namespace": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The namespace where the service account has been created. If not provided defaults to namespace",
				Default:     "",
			},
			"tmp_dir": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     ".tmp/rbac",
				Description: "The temporary directory where config files are written before adding to gitops repo",
			},
			"rules": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"apiGroups": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The apiGroups for the resources in the rule",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"resources": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The resources targeted by the rule",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"resourceNames": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The names of the resources targeted by the rule",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"verbs": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The verbs or actions that can be performed on the resources",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"roles": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of cluster roles that should be applied to the service account",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the cluster role that will be applied",
						},
					},
				},
			},
		},
	}
}

type RBACServiceAccount struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type RBACRule struct {
	ApiGroups     []string `yaml:"apiGroups"`
	Resources     []string `yaml:"resources"`
	ResourceNames []string `yaml:"resourceNames"`
	Verbs         []string `yaml:"verbs"`
}

type RBACRole struct {
	Name string `yaml:"name"`
}

type RBACValues struct {
	ServiceAccount RBACServiceAccount `yaml:"serviceAccount"`
	ClusterScope   bool               `yaml:"clusterScope"`
	Roles          []RBACRole         `yaml:"roles"`
	Rules          []RBACRule         `yaml:"rules"`
}

func resourceGitopsRBACCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	serverName := d.Get("server_name").(string)
	branch := d.Get("branch").(string)
	credentials := d.Get("credentials").(string)
	gitopsConfig := d.Get("config").(string)
	layer := "infrastructure"
	moduleType := "base"

	tmpDir := d.Get("tmp_dir").(string)
	clusterScope := d.Get("cluster_scope").(bool)
	rules := d.Get("roles").([]RBACRule)
	roles := d.Get("roles").([]RBACRole)

	serviceAccountName := d.Get("service_account_name").(string)
	serviceAccountNamespace := d.Get("service_account_namespace").(string)

	rbacValues := RBACValues{
		ServiceAccount: RBACServiceAccount{
			Name:      serviceAccountName,
			Namespace: serviceAccountNamespace,
		},
		ClusterScope: clusterScope,
		Roles:        roles,
		Rules:        rules,
	}

	valueData, err := yaml.Marshal(&rbacValues)
	if err != nil {
		return diag.FromErr(err)
	}

	var scope string
	if clusterScope {
		scope = "cluster"
	} else {
		scope = namespace
	}
	valuesPath := fmt.Sprintf("%s/%s/%s", tmpDir, scope, name)
	valuesFile := fmt.Sprintf("%s/values.yaml", valuesPath)

	binDir := config.BinDir
	lock := config.Lock
	debug := config.Debug
	caCert := config.CaCertFile

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Provisioning gitops rbac: name=%s, namespace=%s, serverName=%s", name, namespace, serverName))

	err = os.Mkdir(valuesPath, os.ModePerm)
	if err != nil {
		diag.FromErr(err)
	}
	err = os.WriteFile(valuesFile, valueData, 0644)
	if err != nil {
		diag.FromErr(err)
	}

	var args = []string{
		"gitops-module",
		name,
		"-n", namespace,
		"--branch", branch,
		"--serverName", serverName,
		"--layer", layer,
		"--type", moduleType}

	args = append(args,
		"--helmRepoUrl", "https://charts.cloudnativetoolkit.dev",
		"--helmChart", "rbac",
		"--helmChartVersion", "0.2.0",
		"--valueFiles", valuesFile)

	if len(lock) > 0 {
		args = append(args, "--lock", lock)
	}
	if len(caCert) > 0 {
		args = append(args, "--caCert", caCert)
	}
	if len(debug) > 0 {
		args = append(args, "--debug", debug)
	}

	cmd := exec.Command(filepath.Join(binDir, "igc"), args...)

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

	tflog.Debug(ctx, fmt.Sprintf("Environment: %v", updatedEnv))

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

	d.SetId(scope + ":" + name + ":" + serverName + ":rbac")

	return diags
}

func resourceGitopsRBACRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	return diags
}

func resourceGitopsRBACUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceGitopsModuleRead(ctx, d, m)
}

func resourceGitopsRBACDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	binDir := config.BinDir
	lock := config.Lock
	debug := config.Debug
	caCert := config.CaCertFile

	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	contentDir := d.Get("content_dir").(string)
	serverName := d.Get("server_name").(string)
	layer := d.Get("layer").(string)
	branch := d.Get("branch").(string)
	moduleType := d.Get("type").(string)
	valueFiles := d.Get("value_files").(string)
	credentials := d.Get("credentials").(string)
	gitopsConfig := d.Get("config").(string)

	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Destroying gitops module: name=%s, namespace=%s, serverName=%s", name, namespace, serverName))

	var args = []string{
		"gitops-module",
		name,
		"--delete",
		"-n", namespace,
		"--contentDir", contentDir,
		"--branch", branch,
		"--serverName", serverName,
		"--layer", layer,
		"--type", moduleType}

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

	cmd := exec.Command(filepath.Join(binDir, "igc"), args...)

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

	tflog.Debug(ctx, fmt.Sprintf("Environment: %v", updatedEnv))

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
