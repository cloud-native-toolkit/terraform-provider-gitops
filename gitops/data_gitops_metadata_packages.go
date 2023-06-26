package gitops

import (
	"context"
	"github.com/google/uuid"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataGitopsMetadataPackages() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataGitopsMetadataPackagesRead,
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
			"package_name_filter": {
				Type:     schema.TypeList,
				Optional: true,
				Description: "List of package name filters to returned packages. The values can be regular expressions. Results will be returned in the order of the matching filters. If not provided or an empty list is provided, all packages will be returned.",
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"packages": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"package_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"catalog_source": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"catalog_source_namespace": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"default_channel": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"publisher": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataGitopsMetadataPackagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	packageFilter := getPackageNameFilterInput(d)

	gitopsMetadata, err := readGitopsMetadata(ctx, binDir, metadataConfig)
	if err != nil {
		return diag.FromErr(err)
	}

	packages := filterPackages(&gitopsMetadata.Packages, packageFilter)

	resultPackages := flattenPackageData(packages)
	if err := d.Set("packages", resultPackages); err != nil {
		return diag.FromErr(err)
	}

	id := uuid.New().String()
	d.SetId(id)

	return diags
}

func filterPackages(packages *[]GitopsMetadataPackage, filters []string) *[]GitopsMetadataPackage {
	if len(filters) == 0 {
		return packages
	}

	result := []GitopsMetadataPackage{}

	for _, filter := range filters {
		r, _ := regexp.Compile(filter)

		for _, testPackage := range *packages {
			if r.MatchString(testPackage.PackageName) {
				result = append(result, testPackage)
			}
		}
	}

	return &result
}

func flattenPackageData(packages *[]GitopsMetadataPackage) []interface{} {
	if packages != nil {
		result := make([]interface{}, len(*packages), len(*packages))

		for i, packageVal := range *packages {
			resultPackage := make(map[string]interface{})

			resultPackage["package_name"] = packageVal.PackageName
			resultPackage["catalog_source"] = packageVal.CatalogSource
			resultPackage["catalog_source_namespace"] = packageVal.CatalogSourceNamespace
			resultPackage["default_channel"] = packageVal.DefaultChannel
			resultPackage["publisher"] = packageVal.Publisher

			result[i] = resultPackage
		}

		return result
	}

	return make([]interface{}, 0)
}