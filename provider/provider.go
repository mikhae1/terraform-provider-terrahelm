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
	KubeAuth    KubeAuth
}

type KubeAuth struct {
	KubeAPIServer             string
	KubeAsGroup               string
	KubeAsUser                string
	KubeCAFile                string
	KubeContext               string
	KubeInsecureSkipTLSVerify bool
	KubeTLSServerName         string
	KubeToken                 string
	Kubeconfig                string
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"helm_version": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_VERSION", "latest"),
				Description: "Helm binary version to install",
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
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"TF_DATA_DIR", "TH_CACHE"}, filepath.Join(".terraform", "terrahelm_cache")),
				Description: "Cache directory path",
			},
			"kube_apiserver": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBEAPISERVER", ""),
				Description: "Address and the port for the Kubernetes API server",
			},
			"kube_as_group": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBEASGROUPS", ""),
				Description: "Group to impersonate for the operation, this flag can be repeated to specify multiple groups",
			},
			"kube_as_user": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBEASUSER", ""),
				Description: "Username to impersonate for the operation",
			},
			"kube_ca_file": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBECAFILE", ""),
				Description: "Certificate authority file for the Kubernetes API server connection",
			},
			"kube_context": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBECONTEXT", ""),
				Description: "Name of the kubeconfig context to use",
			},
			"kube_insecure_skip_tls_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBEINSECURE_SKIP_TLS_VERIFY", false),
				Description: "If true, the Kubernetes API server's certificate will not be checked for validity. This will make your HTTPS connections insecure",
			},
			"kube_tls_server_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBETLS_SERVER_NAMEs", ""),
				Description: "Server name to use for Kubernetes API server certificate validation. If it is not provided, the hostname used to contact the server is used",
			},
			"kube_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBETOKEN", ""),
				Description: "Bearer token used for authentication",
			},
			"kubeconfig": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBECONFIG", ""),
				Description: "Path to the kubeconfig file",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"terrahelm_release": resourceHelmRelease(),
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

		KubeAuth: KubeAuth{
			KubeAPIServer:             d.Get("kube_apiserver").(string),
			KubeAsGroup:               d.Get("kube_as_group").(string),
			KubeAsUser:                d.Get("kube_as_user").(string),
			KubeCAFile:                d.Get("kube_ca_file").(string),
			KubeContext:               d.Get("kube_context").(string),
			KubeInsecureSkipTLSVerify: d.Get("kube_insecure_skip_tls_verify").(bool),
			KubeTLSServerName:         d.Get("kube_tls_server_name").(string),
			KubeToken:                 d.Get("kube_token").(string),
			Kubeconfig:                d.Get("kubeconfig").(string),
		},
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
