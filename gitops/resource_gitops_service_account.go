package gitops

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"gopkg.in/yaml.v3"
	"os"
)

func resourceGitopsServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsServiceAccountCreate,
		ReadContext:   resourceGitopsServiceAccountRead,
		UpdateContext: resourceGitopsServiceAccountUpdate,
		DeleteContext: resourceGitopsServiceAccountDelete,
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
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"infrastructure", "services", "applications"}, false),
				Default:      "infrastructure",
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
			"service_account_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The name of the service account that will be created. If not specified the value will default to the module name",
			},
			"create_service_account": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Flag indicating the service account should be created",
			},
			"all_service_accounts": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag indicating the rbac rules should be applied to all service accounts in the namespace",
			},
			"cluster_scope": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag setting the RBAC rules at cluster scope vs namespace scope",
			},
			"rbac_namespace": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The namespace where the rbac rules should be created. If not provided defaults to the service account namespace",
				Default:     "",
			},
			"tmp_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     ".tmp/service-account",
				Description: "The temporary directory where config files are written before adding to gitops repo",
			},
			"sccs": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of sccs that should be associated with the service account. Supports anyuid and/or privileged",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"rules": {
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
			"roles": {
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
	layer := "infrastructure"
	moduleType := "base"

	tmpDir := d.Get("tmp_dir").(string)
	createServiceAccount := d.Get("create_service_account").(bool)
	allServiceAccounts := d.Get("all_service_accounts").(bool)
	clusterScope := d.Get("cluster_scope").(bool)
	rbacNamespace := d.Get("rbac_namespace").(string)
	serviceAccountName := d.Get("service_account_name").(string)
	sccs := interfacesToString(d.Get("sccs").([]interface{}))
	rules := getRBACRules(d, "rules")
	roles := getRBACRoles(d, "roles")

	if len(serviceAccountName) == 0 {
		serviceAccountName = name
	}

	name = name + "-sa"

	rbacValues := RBACValues{
		Create:             createServiceAccount,
		AllServiceAccounts: allServiceAccounts,
		RbacNamespace:      rbacNamespace,
		Sccs:               sccs,
		ClusterScope:       clusterScope,
		Roles:              roles,
		Rules:              rules,
		Name:               serviceAccountName,
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

	err = os.MkdirAll(valuesPath, os.ModePerm)
	if err != nil {
		return diag.FromErr(err)
	}
	err = os.WriteFile(valuesFile, valueData, 0644)
	if err != nil {
		return diag.FromErr(err)
	}

	moduleConfig := GitopsModuleConfig{
		Name:        name,
		Namespace:   namespace,
		Branch:      getBranchInput(d),
		ServerName:  getServerNameInput(d),
		Layer:       layer,
		Type:        moduleType,
		ValueFiles:  valuesFile,
		CaCert:      config.CaCertFile,
		Debug:       config.Debug,
		Credentials: getCredentialsInput(d),
		Config:      getGitopsConfigInput(d),
		HelmConfig: &HelmConfig{
			RepoUrl:      "https://charts.cloudnativetoolkit.dev",
			Chart:        "service-account",
			ChartVersion: "1.1.0",
		},
	}

	id, err := populateGitopsModule(ctx, config.BinDir, moduleConfig, false)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

	return diags
}

func resourceGitopsServiceAccountRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
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

	name := getNameInput(d)
	namespace := getNamespaceInput(d)
	layer := "infrastructure"
	moduleType := "base"

	name = name + "-sa"

	moduleConfig := GitopsModuleConfig{
		Name:        name,
		Namespace:   namespace,
		Branch:      getBranchInput(d),
		ServerName:  getServerNameInput(d),
		Layer:       layer,
		Type:        moduleType,
		ValueFiles:  "values.yaml",
		CaCert:      config.CaCertFile,
		Debug:       config.Debug,
		Credentials: getCredentialsInput(d),
		Config:      getGitopsConfigInput(d),
		HelmConfig: &HelmConfig{
			RepoUrl:      "https://charts.cloudnativetoolkit.dev",
			Chart:        "service-account",
			ChartVersion: "1.1.0",
		},
	}

	id, err := populateGitopsModule(ctx, config.BinDir, moduleConfig, true)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id)

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
