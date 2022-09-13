package gitops

import (
    "bufio"
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"os"
	"os/exec"
	"path/filepath"
	"fmt"
)

func resourceGitopsModule() *schema.Resource {
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
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{"infrastructure", "services", "applications"}, false),
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
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},
		},
	}
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

	binDir := config.BinDir
	lock := config.Lock
	debug := config.Debug
	caCert := config.CaCertFile

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Provisioning gitops rbac: name=%s, namespace=%s, serverName=%s", name, namespace, serverName))

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
      "--helmChartVersion", "0.1.0")

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

    tflog.Debug(ctx, "Executing command: " + cmd.String())

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

	d.SetId(namespace + ":" + name + ":" + serverName + ":" + contentDir)

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

    tflog.Debug(ctx, "Executing command: " + cmd.String())

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
