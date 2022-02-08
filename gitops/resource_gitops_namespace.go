package gitops

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"log"
	"os/exec"
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
				Required: true,
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
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
	credentials := d.Get("credentials").(string)
	gitopsConfig := d.Get("config").(string)

	binDir := config.BinDir
	lock := config.Lock

	// this should be replaced with the actual git user
	username := "cloudnativetoolkit"

	gitopsMutexKV.Lock(username)

	log.Printf("Provisioning gitops namespace: %s, %s, ", name, serverName)

	defer gitopsMutexKV.Unlock(username)

	cmd := exec.Command(
		binDir+"/igc",
		"gitops-namespace",
		name,
		"--contentDir", contentDir,
		"--lock", lock,
		"--serverName", serverName,
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

	d.SetId(name + ":" + serverName + ":" + contentDir)
	err = d.Set("username", username)

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

	log.Printf("Deleting gitops namespace")

	config := m.(*ProviderConfig)

	binDir := config.BinDir
	lock := config.Lock

	username := "cloudnativetoolkit"

	name := d.Get("name").(string)
	serverName := d.Get("server_name").(string)
	credentials := d.Get("credentials").(string)
	gitopsConfig := d.Get("config").(string)

	gitopsMutexKV.Lock(username)

	log.Printf("Destroying gitops namespace: %s, %s, ", name, serverName)

	defer gitopsMutexKV.Unlock(username)

	cmd := exec.Command(
		binDir+"/igc",
		"gitops-namespace",
		name,
		"--delete",
		"--lock", lock,
		"--serverName", serverName,
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

	d.SetId("")

	return diags
}
