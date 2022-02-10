package gitops

import (
	"bytes"
	"context"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"log"
	"os/exec"
)

func resourceGitopsModule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsModuleCreate,
		ReadContext:   resourceGitopsModuleRead,
		UpdateContext: resourceGitopsModuleUpdate,
		DeleteContext: resourceGitopsModuleDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": &schema.Schema{
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
			"layer": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "base",
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

func resourceGitopsModuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	contentDir := d.Get("content_dir").(string)
	serverName := d.Get("server_name").(string)
	layer := d.Get("layer").(string)
	branch := d.Get("branch").(string)
	moduleType := d.Get("type").(string)
	credentials := d.Get("credentials").(string)
	gitopsConfig := d.Get("config").(string)

	binDir := config.BinDir
	lock := config.Lock

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	tflog.Info(ctx, "Provisioning gitops module: name=%s, namespace=%s, serverName=%s", name, namespace, serverName)

	defer gitopsMutexKV.Unlock(username)

	cmd := exec.Command(
		binDir+"/igc",
		"gitops-module",
		name,
		"-n", namespace,
		"--lock", lock,
		"--contentDir", contentDir,
		"--serverName", serverName,
		"--layer", layer,
		"--branch", branch,
		"--type", moduleType,
		"--debug")

	updatedEnv := append(cmd.Env, "GIT_CREDENTIALS="+credentials)
	updatedEnv = append(updatedEnv, "GITOPS_CONFIG="+gitopsConfig)

	cmd.Env = updatedEnv

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(log.Writer(), &stdoutBuf)
	cmd.Stderr = io.MultiWriter(log.Writer(), &stderrBuf)

	err := cmd.Run()
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(namespace + ":" + name + ":" + serverName + ":" + contentDir)

	return diags
}

func resourceGitopsModuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	return diags
}

func resourceGitopsModuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceGitopsModuleRead(ctx, d, m)
}

func resourceGitopsModuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	binDir := config.BinDir
	lock := config.Lock

	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	serverName := d.Get("server_name").(string)
	layer := d.Get("layer").(string)
	branch := d.Get("branch").(string)
	moduleType := d.Get("type").(string)
	credentials := d.Get("credentials").(string)
	gitopsConfig := d.Get("config").(string)

	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	tflog.Info(ctx, "Destroying gitops module: name=%s, namespace=%s, serverName=%s", name, namespace, serverName)

	defer gitopsMutexKV.Unlock(username)

	cmd := exec.Command(
		binDir+"/igc",
		"gitops-module",
		name,
		"-n", namespace,
		"--delete",
		"--lock", lock,
		"--serverName", serverName,
		"--layer", layer,
		"--branch", branch,
		"--type", moduleType,
		"--debug")

	updatedEnv := append(cmd.Env, "GIT_CREDENTIALS="+credentials)
	updatedEnv = append(updatedEnv, "GITOPS_CONFIG="+gitopsConfig)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(log.Writer(), &stdoutBuf)
	cmd.Stderr = io.MultiWriter(log.Writer(), &stderrBuf)

	err := cmd.Run()
	if err != nil {
		return diag.FromErr(err)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}
