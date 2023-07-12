package gitops

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"os"
	"path"
	"regexp"
	"strings"
)

func getNameInput(d *schema.ResourceData) string {
	return d.Get("name").(string)
}

func getKubeConfigPath(d *schema.ResourceData) string {
	return d.Get("kube_config_path").(string)
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

func getGitopsNamespaceInput(d *schema.ResourceData) string {
	return d.Get("gitops_namespace").(string)
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

func getIgnoreDiffInput(d *schema.ResourceData) string {
	return d.Get("ignore_diff").(string)
}

func getPackageNameFilterInput(d *schema.ResourceData) []string {
	return interfacesToStrings(d.Get("package_name_filter").([]interface{}))
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
	IgnoreDiff  string
}

type GitopsMetadataConfig struct {
	Branch          string
	ServerName      string
	CaCert          string
	Debug           string
	Credentials     string
	Config          string
	KubeConfigPath  string
	GitopsNamespace string
}

type GitopsMetadataCluster struct {
	DefaultIngressSubdomain string
	DefaultIngressSecret    string
	KubeVersion             string
	OpenShiftVersion        string
	Type                    string
	OperatorNamespace       string
	GitopsNamespace         string
}

type GitopsMetadataPackage struct {
	PackageName            string
	CatalogSource          string
	CatalogSourceNamespace string
	DefaultChannel         string
	Publisher              string
}

type GitopsMetadata struct {
	Cluster  GitopsMetadataCluster
	Packages []GitopsMetadataPackage
}

func interfacesToStrings(list []interface{}) []string {
	if list == nil {
		return nil
	}

	result := make([]string, len(list))
	for i, item := range list {
		if item == nil {
			result[i] = ""
		} else {
			result[i] = item.(string)
		}
	}

	return result
}

func stringsToInterfaces(sVal *[]string) []interface{} {
	if sVal != nil {
		result := make([]interface{}, len(*sVal), len(*sVal))

		for i, val := range *sVal {
			result[i] = val
		}

		return result
	}

	return make([]interface{}, 0)
}

func logEnvironment(ctx context.Context, env *[]string) {
	newEnv := *removeItem(env, "^GIT_CREDENTIALS")

	if len(*env) != len(newEnv) {
		newEnv = append(newEnv, "GIT_CREDENTIALS=**redacted**")
	}

	tflog.Debug(ctx, fmt.Sprintf("Environment: %v", newEnv))
}

func findItem(env *[]string, match string) int {
	r, _ := regexp.Compile(match)
	for i := 0; i < len(*env); i++ {
		if r.MatchString((*env)[i]) {
			return i
		}
	}

	return -1
}

func removeItem(env *[]string, match string) *[]string {
	pos := findItem(env, match)

	if pos == -1 {
		return env
	}

	result := make([]string, len(*env) - 1)

	for i := 0; i < len(*env); i++ {
		if i != pos {
			result = append(result, (*env)[i])
		}
	}

	return &result
}

func pathWithBinDir(binDir string) string {
	currentPath := os.Getenv("PATH")

	if len(binDir) == 0 {
		return currentPath
	}

	if !strings.HasPrefix(binDir, "/") {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}

		binDir = path.Join(cwd, binDir)
	}

	if strings.HasPrefix(currentPath, binDir) {
		return currentPath
	}

	return fmt.Sprintf("%s:%s", binDir, currentPath)
}