package gitops

import (
	"context"
	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataGitopsMetadataCluster() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataGitopsMetadataClusterRead,
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
			"credentials": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"config": {
				Type:     schema.TypeString,
				Required: true,
			},
			"default_ingress_subdomain": {
				Type: schema.TypeString,
				Computed: true,
				Description: "The default ingress subdomain for the cluster used to build ingress/route host names",
			},
			"default_ingress_secret": {
				Type: schema.TypeString,
				Computed: true,
				Description: "The name of the secret in the cluster that holds the TLS information used to create secured ingresses",
			},
			"cluster_type": {
				Type: schema.TypeString,
				Computed: true,
				Description: "The type of cluster. Values will be 'ocp4' or 'kubernetes'",
			},
			"kube_version": {
				Type: schema.TypeString,
				Computed: true,
				Description: "The kubernetes version of the cluster",
			},
			"openshift_version": {
				Type: schema.TypeString,
				Computed: true,
				Description: "The OpenShift version of the cluster. If the cluster is not an OpenShift cluster this value will be empty",
			},
			"operator_namespace": {
				Type: schema.TypeString,
				Computed: true,
				Description: "The namespace where the cluster-wide operators are installed in the cluster",
			},
			"gitops_namespace": {
				Type: schema.TypeString,
				Computed: true,
				Description: "The namespace where the gitops instance is installed in the cluster",
			},
		},
	}
}

func dataGitopsMetadataClusterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	binDir := config.BinDir

	metadataConfig := GitopsMetadataConfig{
		Branch:         getBranchInput(d),
		ServerName:     getServerNameInput(d),
		Credentials:    getCredentialsInput(d),
		Config:         getGitopsConfigInput(d),
		CaCert:         config.GitConfig.CaCertFile,
		Debug:          config.Debug,
	}

	gitopsMetadata, err := readGitopsMetadata(ctx, binDir, metadataConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("cluster_type", gitopsMetadata.Cluster.Type)
	err = d.Set("kube_version", gitopsMetadata.Cluster.KubeVersion)
	err = d.Set("openshift_version", gitopsMetadata.Cluster.OpenShiftVersion)
	err = d.Set("default_ingress_subdomain", gitopsMetadata.Cluster.DefaultIngressSubdomain)
	err = d.Set("default_ingress_secret", gitopsMetadata.Cluster.DefaultIngressSecret)
	err = d.Set("operator_namespace", gitopsMetadata.Cluster.OperatorNamespace)
	err = d.Set("gitops_namespace", gitopsMetadata.Cluster.GitopsNamespace)

	id := uuid.New().String()
	d.SetId(id)

	return diags
}
