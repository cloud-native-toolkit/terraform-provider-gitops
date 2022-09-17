package gitops

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func resourceGitopsSealSecrets() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGitopsSealSecretsCreate,
		ReadContext:   resourceGitopsSealSecretsRead,
		UpdateContext: resourceGitopsSealSecretsUpdate,
		DeleteContext: resourceGitopsSealSecretsDelete,
		Schema: map[string]*schema.Schema{
			"source_dir": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"dest_dir": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"kubeseal_cert": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"annotations": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of annotations that should be added to the generated Sealed Secrets. Expected format of each annotation is a string if 'key=value'",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"tmp_dir": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     ".tmp/sealed-secrets",
				Description: "The temporary directory where the cert will be written",
			},
		},
	}
}

func resourceGitopsSealSecretsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	config := m.(*ProviderConfig)

	sourceDir := d.Get("source_dir").(string)
	destDir := d.Get("dest_dir").(string)
	cert := d.Get("kubeseal_cert").(string)
	annotations := interfacesToString(d.Get("annotations").([]interface{}))
	tmpDir := d.Get("tmp_dir").(string)

	binDir := config.BinDir

	certFile, err := writeCertFile(tmpDir, cert)
	if err != nil {
		return diag.FromErr(err)
	}

	var baseArgs = []string{
		"--cert", certFile,
		"--format", "yaml",
		"--raw"}

	err = os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return diag.FromErr(err)
	}

	files, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), "yaml") {
			continue
		}

		tflog.Info(ctx, "Sealing file: "+file.Name())

		if annotations == nil {
			result, err := encryptFile(baseArgs, binDir, sourceDir, destDir, file.Name())
			if err != nil {
				return diag.FromErr(err)
			}

			tflog.Debug(ctx, "Sealed file written to: "+result)
		} else {
			result, err := encryptFileWithAnnotations(baseArgs, binDir, sourceDir, destDir, file.Name(), annotations)
			if err != nil {
				return diag.FromErr(err)
			}

			tflog.Debug(ctx, "Sealed file written to: "+result)
		}
	}

	d.SetId("sealCert:" + sourceDir + ":" + destDir)

	return diags
}

func resourceGitopsSealSecretsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	return diags
}

func resourceGitopsSealSecretsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO implement update...
	return resourceGitopsModuleRead(ctx, d, m)
}

func resourceGitopsSealSecretsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	d.SetId("")

	return diags
}

func encryptFile(args []string, binDir string, sourceDir string, destDir string, fileName string) (string, error) {
	sourceContents, err := os.ReadFile(fmt.Sprintf("%s/%s", sourceDir, fileName))
	if err != nil {
		return "", err
	}

	cmd := exec.Command(filepath.Join(binDir, "kubeseal"), args...)

	destFile := fmt.Sprintf("%s/%s", destDir, fileName)

	outfile, err := os.Create(destFile)
	if err != nil {
		return "", err
	}
	defer outfile.Close()

	pr, pw := io.Pipe()
	cmd.Stdin = pr
	cmd.Stdout = outfile
	_, err = pw.Write(sourceContents)
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	return destFile, nil
}

func encryptFileWithAnnotations(baseArgs []string, binDir string, sourceDir string, destDir string, fileName string, annotations []string) (string, error) {
	args := append(baseArgs, "--from-file", fmt.Sprintf("%s/%s", sourceDir, fileName))

	cmd := exec.Command(filepath.Join(binDir, "kubeseal"), args...)

	destFile := fmt.Sprintf("%s/%s", destDir, fileName)

	outfile, err := os.Create(destFile)
	if err != nil {
		return "", err
	}
	defer outfile.Close()

	pr, pw := io.Pipe()
	var cmd2 *exec.Cmd
	if len(annotations) == 0 {
		cmd2 = exec.Command("echo", "")

		cmd.Stdout = outfile
	} else {
		annotationArgs := []string{
			"annotate",
			"-f", "-",
			"--local=true",
			"--dry-run=client",
			"--output=yaml"}

		annotationArgs = append(annotationArgs, annotations...)

		cmd2 = exec.Command(filepath.Join(binDir, "kubectl"), annotationArgs...)

		cmd.Stdout = pw
		cmd2.Stdin = pr

		cmd2.Stdout = outfile
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd2.Start()
	if err != nil {
		return "", err
	}

	err = func() error {
		defer pw.Close()

		return cmd.Wait()
	}()
	if err != nil {
		return "", err
	}

	err = cmd2.Wait()
	if err != nil {
		return "", err
	}

	return destFile, nil
}

func writeCertFile(dirName string, cert string) (string, error) {

	err := os.MkdirAll(dirName, os.ModePerm)
	if err != nil {
		return "", err
	}

	certFile := fmt.Sprintf("%s/kubeseal.crt", dirName)
	err = os.WriteFile(certFile, []byte(cert), 0644)
	if err != nil {
		return "", err
	}

	return certFile, nil
}
