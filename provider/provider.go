package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type providerConfig struct {
	HelmPath string
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"helm_version": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_VERSION", "latest"),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"terrahelm_release": resourceHelmGitChart(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	helmVersion := d.Get("helm_version").(string)

	helmPath, err := installHelmCLI(helmVersion)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	config := &providerConfig{
		HelmPath: helmPath,
	}

	return config, nil
}

func installHelmCLI(helmVersion string) (helmPath string, err error) {
	cacheDir := filepath.Join(os.TempDir(), "terrahelm_cache")
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %v", err)
	}
	fmt.Printf("[DEBUG] using cacheDir: %v\n", cacheDir)

	helmDir := filepath.Join(cacheDir, helmVersion)
	helmPath = filepath.Join(helmDir, "helm")
	if _, err := os.Stat(helmPath); err == nil {
		log.Printf("Using cached Helm binary: %s", helmPath)
		return helmPath, nil
	}

	if err := os.MkdirAll(helmDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create Helm directory: %v", err)
	}

	installCmd := exec.Command("curl", "-fsSL", "-o", filepath.Join(helmDir, "get_helm.sh"), "https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3")
	if err := installCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download Helm installation script: %v", err)
	}

	chmodCmd := exec.Command("chmod", "700", filepath.Join(helmDir, "get_helm.sh"))
	if err := chmodCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set execute permission on Helm installation script: %v", err)
	}

	installHelmCmd := exec.Command(filepath.Join(helmDir, "get_helm.sh"), "--version", "v"+helmVersion)
	installHelmCmd.Env = append(os.Environ(),
		"HELM_INSTALL_DIR="+helmDir,
		"USE_SUDO=false",
	)
	output, err := installHelmCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to install Helm: %v\nOutput: %s", err, output)
	}

	// helmPath = filepath.Join(helmDir, "helm")

	log.Printf("Helm version: %s is installed at: %s", helmVersion, helmPath)

	return helmPath, nil
}
