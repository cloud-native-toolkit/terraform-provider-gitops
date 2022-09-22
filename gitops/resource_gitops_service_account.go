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

func resourceGitopsServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsServiceAccountCreate,
		ReadContext:   resourceGitopsServiceAccountRead,
		UpdateContext: resourceGitopsServiceAccountUpdate,
		DeleteContext: resourceGitopsServiceAccountDelete,
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
			"service_account_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"create_service_account": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Flag indicating the service account should be created",
			},
			"all_service_accounts": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag indicating the rbac rules should be applied to all service accounts in the namespace",
			},
			"cluster_scope": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag setting the RBAC rules at cluster scope vs namespace scope",
			},
			"rbac_namespace": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The namespace where the rbac rules should be created. If not provided defaults to the service account namespace",
				Default:     "",
			},
			"tmp_dir": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     ".tmp/service-account",
				Description: "The temporary directory where config files are written before adding to gitops repo",
			},
			"sccs": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of sccs that should be associated with the service account. Supports anyuid and/or privileged",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"rules": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_groups": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "The apiGroups for the resources in the rule",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"resources": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "The resources targeted by the rule",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"resource_names": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The names of the resources targeted by the rule",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"verbs": {
							Type:        schema.TypeList,
							Required:    true,
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
							Required:    true,
							Description: "The name of the cluster role that will be applied",
						},
					},
				},
			},
		},
	}
}

type RBACRule struct {
	ApiGroups     []string `yaml:"apiGroups"`
	Resources     []string `yaml:"resources"`
	ResourceNames []string `yaml:"resourceNames,omitempty"`
	Verbs         []string `yaml:"verbs"`
}

type RBACRole struct {
	Name string `yaml:"name"`
}

type RBACValues struct {
	Create             bool       `yaml:"create"`
	AllServiceAccounts bool       `yaml:"allServiceAccounts"`
	ClusterScope       bool       `yaml:"clusterScope"`
	Sccs               []string   `yaml:"sccs"`
	Roles              []RBACRole `yaml:"roles"`
	Rules              []RBACRule `yaml:"rules"`
	RbacNamespace      string     `yaml:"rbacNamespace"`
	Name               string     `yaml:"name,omitempty"`
}

func resourceGitopsServiceAccountCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	name := getNameInput(d)
	namespace := getNamespaceInput(d)
	serverName := getServerNameInput(d)
	branch := getBranchInput(d)
	credentials := getCredentialsInput(d)
	gitopsConfig := getGitopsConfigInput(d)
	layer := "infrastructure"
	moduleType := "base"

	tmpDir := d.Get("tmp_dir").(string)
	createServiceAccount := d.Get("create_service_account").(bool)
	allServiceAccounts := d.Get("all_service_accounts").(bool)
	clusterScope := d.Get("cluster_scope").(bool)
	rbacNamespace := d.Get("rbac_namespace").(string)
	servieAccountName := d.Get("service_account_name").(string)
	sccs := interfacesToString(d.Get("sccs").([]interface{}))
	rules := getRBACRules(d, "rules")
	roles := getRBACRoles(d, "roles")

	rbacValues := RBACValues{
		Create:             createServiceAccount,
		AllServiceAccounts: allServiceAccounts,
		RbacNamespace:      rbacNamespace,
		Sccs:               sccs,
		ClusterScope:       clusterScope,
		Roles:              roles,
		Rules:              rules,
		Name:               servieAccountName,
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

	err = os.MkdirAll(valuesPath, os.ModePerm)
	if err != nil {
		return diag.FromErr(err)
	}
	err = os.WriteFile(valuesFile, valueData, 0644)
	if err != nil {
		return diag.FromErr(err)
	}

	var args = []string{
		"gitops-module",
		name,
		"-n", namespace,
		"--branch", branch,
		"--serverName", serverName,
		"--layer", layer,
		"--type", moduleType,
		"--helmRepoUrl", "https://charts.cloudnativetoolkit.dev",
		"--helmChart", "service-account",
		"--helmChartVersion", "1.0.0",
		"--valueFiles", valuesFile}

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

	d.SetId(scope + ":" + name + ":" + serverName + ":service-account")

	return diags
}

func resourceGitopsServiceAccountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	return diags
}

func resourceGitopsServiceAccountUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO implement update...
	return resourceGitopsModuleRead(ctx, d, m)
}

func resourceGitopsServiceAccountDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	binDir := config.BinDir
	lock := config.Lock
	debug := config.Debug
	caCert := config.CaCertFile

	name := getNameInput(d)
	namespace := getNamespaceInput(d)
	serverName := getServerNameInput(d)
	branch := getBranchInput(d)
	credentials := getCredentialsInput(d)
	gitopsConfig := getGitopsConfigInput(d)
	layer := "infrastructure"
	moduleType := "base"

	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Destroying gitops module: name=%s, namespace=%s, serverName=%s", name, namespace, serverName))

	var args = []string{
		"gitops-module",
		name,
		"--delete",
		"-n", namespace,
		"--branch", branch,
		"--serverName", serverName,
		"--layer", layer,
		"--type", moduleType,
		"--helmRepoUrl", "https://charts.cloudnativetoolkit.dev",
		"--helmChart", "service-account",
		"--helmChartVersion", "1.0.0"}

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

	d.SetId("")

	return diags
}

func getRBACRules(d *schema.ResourceData, name string) []RBACRule {
	rawRules := d.Get(name).([]interface{})

	rules := []RBACRule{}
	for _, item := range rawRules {
		i := item.(map[string]interface{})

		rule := RBACRule{
			ApiGroups:     interfacesToString(i["api_groups"].([]interface{})),
			Resources:     interfacesToString(i["resources"].([]interface{})),
			ResourceNames: interfacesToString(i["resource_names"].([]interface{})),
			Verbs:         interfacesToString(i["verbs"].([]interface{})),
		}

		rules = append(rules, rule)
	}

	return rules
}

func getRBACRoles(d *schema.ResourceData, name string) []RBACRole {
	rawRoles := d.Get(name).([]interface{})

	roles := []RBACRole{}
	for _, item := range rawRoles {
		i := item.(map[string]interface{})

		role := RBACRole{
			Name: i["name"].(string),
		}

		roles = append(roles, role)
	}
	return roles
}
