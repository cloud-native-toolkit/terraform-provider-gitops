package gitops

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func getNameInput(d *schema.ResourceData) string {
	return d.Get("name").(string)
}

func getNamespaceInput(d *schema.ResourceData) string {
	return d.Get("namespace").(string)
}

func getServerNameInput(d *schema.ResourceData) string {
	return d.Get("server_name").(string)
}

func getBranchInput(d *schema.ResourceData) string {
	return d.Get("branch").(string)
}

func getCredentialsInput(d *schema.ResourceData) string {
	return d.Get("credentials").(string)
}

func getGitopsConfigInput(d *schema.ResourceData) string {
	return d.Get("config").(string)
}

func getLayerInput(d *schema.ResourceData) string {
	return d.Get("layer").(string)
}

func getTypeInput(d *schema.ResourceData) string {
	return d.Get("type").(string)
}

func getContentDirInput(d *schema.ResourceData) string {
	return d.Get("content_dir").(string)
}

func getValueFilesInput(d *schema.ResourceData) string {
	return d.Get("value_files").(string)
}

func getHelmRepoUrlInput(d *schema.ResourceData) string {
	return d.Get("helm_repo_url").(string)
}

func getHelmChartInput(d *schema.ResourceData) string {
	return d.Get("helm_chart").(string)
}

func getHelmChartVersionInput(d *schema.ResourceData) string {
	return d.Get("helm_chart_version").(string)
}

type HelmConfig struct {
	RepoUrl      string
	Chart        string
	ChartVersion string
}

func helmConfigFromResourceData(d *schema.ResourceData) *HelmConfig {
	helmRepoUrl := getHelmRepoUrlInput(d)
	helmChart := getHelmChartInput(d)
	helmChartVersion := getHelmChartVersionInput(d)

	var helmConfig *HelmConfig
	if len(helmRepoUrl) > 0 && len(helmChart) > 0 && len(helmChartVersion) > 0 {
		helmConfig = &HelmConfig{
			RepoUrl:      helmRepoUrl,
			Chart:        helmChart,
			ChartVersion: helmChartVersion,
		}
	}

	return helmConfig
}

type GitopsModuleConfig struct {
	Name        string
	Namespace   string
	Branch      string
	ServerName  string
	Layer       string
	Type        string
	ContentDir  string
	HelmConfig  *HelmConfig
	ValueFiles  string
	CaCert      string
	Debug       string
	Credentials string
	Config      string
}
