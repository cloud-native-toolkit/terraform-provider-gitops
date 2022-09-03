package gitops

import (
	"bufio"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"os/exec"
	"path/filepath"
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

	binDir := config.BinDir
	lock := config.Lock
	debug := config.Debug
	caCert := config.CaCertFile

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	defer gitopsMutexKV.Unlock(username)

	tflog.Info(ctx, fmt.Sprintf("Provisioning gitops namespace: name=%s, serverName=%s", name, serverName))

	var args = []string{
	  "gitops-namespace",
	  name,
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

	cmd := exec.Command(filepath.Join(binDir, "igc"), args...)

    tflog.Debug(ctx, "Executing command: " + cmd.String())

	gitEmail := "cloudnativetoolkit@gmail.com"
	gitName := "Cloud Native Toolkit"

	updatedEnv := append(cmd.Env, "GIT_CREDENTIALS="+credentials)
	updatedEnv = append(updatedEnv, "GITOPS_CONFIG="+gitopsConfig)
	updatedEnv = append(updatedEnv, "EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_NAME="+gitName)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_NAME="+gitName)

	cmd.Env = updatedEnv

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return diag.FromErr(err)
    }

    // start the command after having set up the pipe
    if err := cmd.Start(); err != nil {
		return diag.FromErr(err)
    }

    // read command's stdout line by line
    in := bufio.NewScanner(stdout)

    for in.Scan() {
        if debug == "true" {
          tflog.Debug(ctx, in.Text())
        } else {
          tflog.Info(ctx, in.Text())
        }
    }

    if err := cmd.Wait(); err != nil {
        tflog.Error(ctx, "Error running command")
        return diag.FromErr(err)
    }

    if err := in.Err(); err != nil {
        tflog.Error(ctx, "Error processing stream")
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
	caCert := config.CaCertFile

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

	cmd := exec.Command(filepath.Join(binDir, "igc"), args...)

    tflog.Debug(ctx, "Executing command: " + cmd.String())

	gitEmail := "cloudnativetoolkit@gmail.com"
	gitName := "Cloud Native Toolkit"

	updatedEnv := append(cmd.Env, "GIT_CREDENTIALS="+credentials)
	updatedEnv = append(updatedEnv, "GITOPS_CONFIG="+gitopsConfig)
	updatedEnv = append(updatedEnv, "EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_AUTHOR_NAME="+gitName)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_EMAIL="+gitEmail)
	updatedEnv = append(updatedEnv, "GIT_COMMITTER_NAME="+gitName)

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return diag.FromErr(err)
    }

    // start the command after having set up the pipe
    if err := cmd.Start(); err != nil {
		return diag.FromErr(err)
    }

    // read command's stdout line by line
    in := bufio.NewScanner(stdout)

    for in.Scan() {
        if debug == "true" {
          tflog.Debug(ctx, in.Text())
        } else {
          tflog.Info(ctx, in.Text())
        }
    }

    if err := cmd.Wait(); err != nil {
        tflog.Error(ctx, "Error running command")
		stderr, _ := cmd.StderrPipe()
		errin := bufio.NewScanner(stderr)
		for errin.Scan() {
			tflog.Info(ctx, errin.Text())
		}
		return diag.FromErr(err)
    }

    if err := in.Err(); err != nil {
        tflog.Error(ctx, "Error processing stream")
        return diag.FromErr(err)
    }

	d.SetId("")

	return diags
}
