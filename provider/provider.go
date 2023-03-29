package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const GET_HELM_URL = "https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3"

type ProviderConfig struct {
	HelmBinPath string
	HelmVersion string
	CacheDir    string
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"helm_version": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_VERSION", "latest"),
			},
			"helm_bin_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_BIN_PATH", ""),
				Description: "If provided it will be used instead for installing Helm binary",
			},
			"cache_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_DATA_DIR", filepath.Join(".terraform", "terrahelm_cache")),
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
	helmBinPath := d.Get("helm_bin_path").(string)
	cacheDir := d.Get("cache_dir").(string)

	tflog.Debug(ctx, "Init cache directory: "+cacheDir)
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return nil, diag.Errorf("failed to create cache directory: %v", err)
	}

	if helmBinPath == "" {
		var err error
		if helmBinPath, err = installHelmCLI(helmVersion, cacheDir); err != nil {
			return nil, diag.FromErr(err)
		}
		tflog.Info(ctx, "Helm version: "+helmVersion+" is installed at: "+helmBinPath)
	}

	tflog.Info(ctx, "Helm binary path: "+helmBinPath)

	return &ProviderConfig{
		HelmBinPath: helmBinPath,
		HelmVersion: helmVersion,
		CacheDir:    cacheDir,
	}, nil
}

func installHelmCLI(helmVersion string, cacheDir string) (helmBinPath string, err error) {
	helmDir := filepath.Join(cacheDir, "helm", helmVersion)
	helmBinPath = filepath.Join(helmDir, "helm")
	if _, err := os.Stat(helmBinPath); err == nil {
		log.Printf("Using cached Helm binary: %s", helmBinPath)
		return helmBinPath, nil
	}

	if err := os.MkdirAll(helmDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create Helm directory: %v", err)
	}

	installCmd := exec.Command("curl", "-fsSL", "-o", filepath.Join(helmDir, "get_helm.sh"), GET_HELM_URL)
	if err := installCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download Helm installation script: %v", err)
	}

	chmodCmd := exec.Command("chmod", "700", filepath.Join(helmDir, "get_helm.sh"))
	if err := chmodCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set execute permission on Helm installation script: %v", err)
	}

	installHelmCmd := exec.Command(filepath.Join(helmDir, "get_helm.sh"), "--version", helmVersion)
	installHelmCmd.Env = append(os.Environ(),
		"HELM_INSTALL_DIR="+helmDir,
		"USE_SUDO=false",
	)
	output, err := installHelmCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to install Helm: %v\nOutput: %s", err, output)
	}

	return helmBinPath, nil
}
