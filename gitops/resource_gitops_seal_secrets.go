package gitops

import (
	"bufio"
	"bytes"
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

	certFile, err := writeCertFile(ctx, tmpDir, cert)
	if err != nil {
		return diag.FromErr(err)
	}

	var baseArgs = []string{
		"--cert", certFile,
		"--format", "yaml"}

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
			tflog.Debug(ctx, "Skipping file because it is not a yaml file: "+file.Name())
			continue
		}

		tflog.Info(ctx, "Encrypting file: "+file.Name())

		if annotations == nil {
			tflog.Debug(ctx, "Encrypting file without annotations")

			result, err := encryptFile(ctx, baseArgs, binDir, sourceDir, destDir, file.Name())
			if err != nil {
				return diag.FromErr(err)
			}

			tflog.Debug(ctx, "Sealed file written to: "+result)
		} else {
			tflog.Debug(ctx, "Encrypting file with annotations")

			result, err := encryptFileWithAnnotations(ctx, baseArgs, binDir, sourceDir, destDir, file.Name(), annotations)
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

func encryptFile(ctx context.Context, args []string, binDir string, sourceDir string, destDir string, fileName string) (string, error) {
	sourceFile := fmt.Sprintf("%s/%s", sourceDir, fileName)
	tflog.Debug(ctx, "Reading file contents: "+sourceFile)

	sourceContents, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", err
	}

	destFile := fmt.Sprintf("%s/%s", destDir, fileName)
	tflog.Debug(ctx, "Encrypted secret destination file: "+destFile)

	cmd := exec.Command(filepath.Join(binDir, "kubeseal"), args...)
	tflog.Debug(ctx, "Executing command: "+cmd.String())

	outfilePipeIn, err := os.Create(destFile)
	if err != nil {
		return "", err
	}
	defer outfilePipeIn.Close()

	cmd.Stdin = bytes.NewReader(sourceContents)
	cmd.Stdout = outfilePipeIn

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	tflog.Debug(ctx, "Writing input yaml to pipe: "+string(sourceContents))

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	inErr := bufio.NewScanner(stderr)
	for inErr.Scan() {
		tflog.Error(ctx, inErr.Text())
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	return destFile, nil
}

func encryptFileWithAnnotations(ctx context.Context, args []string, binDir string, sourceDir string, destDir string, fileName string, annotations []string) (string, error) {
	sourceFile := fmt.Sprintf("%s/%s", sourceDir, fileName)
	tflog.Debug(ctx, "Reading file contents: "+sourceFile)

	sourceContents, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(filepath.Join(binDir, "kubeseal"), args...)

	destFile := fmt.Sprintf("%s/%s", destDir, fileName)

	annotationArgs := []string{
		"annotate",
		"-f", "-",
		"--local=true",
		"--dry-run=client",
		"--output=yaml"}

	annotationArgs = append(annotationArgs, annotations...)

	cmd2 := exec.Command(filepath.Join(binDir, "kubectl"), annotationArgs...)

	outfilePipeIn, err := os.Create(destFile)
	if err != nil {
		return "", err
	}
	defer outfilePipeIn.Close()

	sealedSecretPipeOut, sealedSecretPipeIn := io.Pipe()

	cmd.Stdin = bytes.NewReader(sourceContents)
	cmd.Stdout = sealedSecretPipeIn
	cmd2.Stdin = sealedSecretPipeOut
	cmd2.Stdout = outfilePipeIn

	err = cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd2.Start()
	if err != nil {
		return "", err
	}

	err = func() error {
		defer sealedSecretPipeIn.Close()

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

func writeCertFile(ctx context.Context, dirName string, cert string) (string, error) {

	err := os.MkdirAll(dirName, os.ModePerm)
	if err != nil {
		return "", err
	}

	certFile := fmt.Sprintf("%s/kubeseal.crt", dirName)

	tflog.Info(ctx, "Writing cert to file: "+certFile)
	tflog.Debug(ctx, "cert: "+cert)

	err = os.WriteFile(certFile, []byte(cert), 0644)
	if err != nil {
		return "", err
	}

	return certFile, nil
}
