package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const GET_HELM_URL = "https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3"

type ProviderConfig struct {
	HelmBinPath string
	GitBinPath  string
	HelmVersion string
	CacheDir    string
	KubeAuth    KubeAuth
	HelmCmd     func(ctx context.Context, args ...string) (*exec.Cmd, error)
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
	KubeExec                  *KubeExec
}

type KubeExec struct {
	APIVersion     string
	Command        string
	Args           []string
	Env            map[string]string
	TimeoutSeconds int
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
			"git_bin_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GIT_BIN_PATH", "git"),
				Description: "Git binary path to use for git clone",
			},
			"cache_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"TF_DATA_DIR", "TH_CACHE"}, filepath.Join(".terraform", "terrahelm_cache")),
				Description: "Provider cache directory path",
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
			"kube_exec": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Exec-based authentication configuration (e.g., EKS, AKS, GKE). Supports raw token output and JSON token fields like status.token/accessToken/access_token",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_version": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "ExecCredential API version passed via KUBERNETES_EXEC_INFO",
						},
						"command": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Command to execute for retrieving the Kubernetes token",
						},
						"args": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Arguments to pass to the exec command",
						},
						"env": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Environment variables to pass to the exec command",
						},
						"timeout_seconds": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     30,
							Description: "Maximum number of seconds to wait for the exec command before failing",
						},
					},
				},
			},
			"kubeconfig": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBECONFIG", ""),
				Description: "Path to the kubeconfig file",
			},
		},

		ConfigureContextFunc: configureProvider,

		DataSourcesMap: map[string]*schema.Resource{
			"terrahelm_release": dataSourceHelmRelease(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"terrahelm_release": resourceHelmRelease(),
		},
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	helmVersion := d.Get("helm_version").(string)
	helmBinPath := d.Get("helm_bin_path").(string)
	gitGitBinPath := d.Get("git_bin_path").(string)
	cacheDir := d.Get("cache_dir").(string)

	var kubeExec *KubeExec
	if kubeExecList, ok := d.GetOk("kube_exec"); ok {
		execItems := kubeExecList.([]interface{})
		if len(execItems) > 0 && execItems[0] != nil {
			execMap := execItems[0].(map[string]interface{})
			command := execMap["command"].(string)
			apiVersion := execMap["api_version"].(string)
			timeoutSeconds := execMap["timeout_seconds"].(int)

			var args []string
			if rawArgs, ok := execMap["args"].([]interface{}); ok {
				args = make([]string, 0, len(rawArgs))
				for _, arg := range rawArgs {
					args = append(args, arg.(string))
				}
			}

			env := map[string]string{}
			if rawEnv, ok := execMap["env"].(map[string]interface{}); ok {
				for key, value := range rawEnv {
					env[key] = value.(string)
				}
			}

			kubeExec = &KubeExec{
				APIVersion:     apiVersion,
				Command:        command,
				Args:           args,
				Env:            env,
				TimeoutSeconds: timeoutSeconds,
			}
		}
	}

	tflog.Debug(ctx, "Init cache directory: "+cacheDir)
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return nil, diag.Errorf("failed to create cache directory (try to use 'cache_dir' arg): %v", err)
	}

	if helmBinPath == "" {
		var err error
		if helmBinPath, err = installHelmCLI(helmVersion, cacheDir); err != nil {
			return nil, diag.FromErr(err)
		}
		tflog.Info(ctx, "Helm version: "+helmVersion+" is installed at: "+helmBinPath)
	}
	tflog.Info(ctx, "Helm binary: "+helmBinPath)

	kubeAuth := KubeAuth{
		KubeAPIServer:             d.Get("kube_apiserver").(string),
		KubeAsGroup:               d.Get("kube_as_group").(string),
		KubeAsUser:                d.Get("kube_as_user").(string),
		KubeCAFile:                d.Get("kube_ca_file").(string),
		KubeContext:               d.Get("kube_context").(string),
		KubeInsecureSkipTLSVerify: d.Get("kube_insecure_skip_tls_verify").(bool),
		KubeTLSServerName:         d.Get("kube_tls_server_name").(string),
		KubeToken:                 d.Get("kube_token").(string),
		Kubeconfig:                d.Get("kubeconfig").(string),
		KubeExec:                  kubeExec,
	}

	helmCmdFunc := func(callCtx context.Context, args ...string) (*exec.Cmd, error) {
		helmCmd := exec.Command(helmBinPath, args...)

		if kubeAuth.KubeAPIServer != "" {
			helmCmd.Args = append(helmCmd.Args, "--kube-apiserver", kubeAuth.KubeAPIServer)
		}
		if kubeAuth.KubeAsUser != "" {
			helmCmd.Args = append(helmCmd.Args, "--kube-as-user", kubeAuth.KubeAsUser)
		}
		if kubeAuth.KubeAsGroup != "" {
			helmCmd.Args = append(helmCmd.Args, "--kube-as-group", kubeAuth.KubeAsGroup)
		}
		if kubeAuth.KubeCAFile != "" {
			helmCmd.Args = append(helmCmd.Args, "--kube-ca-file", kubeAuth.KubeCAFile)
		}
		if kubeAuth.KubeContext != "" {
			helmCmd.Args = append(helmCmd.Args, "--kube-context", kubeAuth.KubeContext)
		}
		if kubeAuth.KubeInsecureSkipTLSVerify {
			helmCmd.Args = append(helmCmd.Args, "--kube-insecure-skip-tls-verify")
		}
		if kubeAuth.KubeTLSServerName != "" {
			helmCmd.Args = append(helmCmd.Args, "--kube-tls-server-name", kubeAuth.KubeTLSServerName)
		}
		if helmCommandNeedsKubeToken(args) {
			kubeToken, err := resolveKubeToken(callCtx, kubeAuth)
			if err != nil {
				return nil, err
			}
			if kubeToken != "" {
				helmCmd.Args = append(helmCmd.Args, "--kube-token", kubeToken)
			}
		}
		if kubeAuth.Kubeconfig != "" {
			helmCmd.Args = append(helmCmd.Args, "--kubeconfig", kubeAuth.Kubeconfig)
		}

		tflog.Debug(ctx, "Helm Command:"+redactHelmArgs(helmCmd.Args))
		return helmCmd, nil
	}

	return &ProviderConfig{
		HelmBinPath: helmBinPath,
		GitBinPath:  gitGitBinPath,
		HelmVersion: helmVersion,
		CacheDir:    cacheDir,
		KubeAuth:    kubeAuth,
		HelmCmd:     helmCmdFunc,
	}, nil
}

func resolveKubeToken(ctx context.Context, kubeAuth KubeAuth) (string, error) {
	if kubeAuth.KubeToken != "" {
		return kubeAuth.KubeToken, nil
	}
	if kubeAuth.KubeExec == nil {
		return "", nil
	}
	return kubeAuth.KubeExec.token(ctx)
}

type execInfo struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Spec       struct {
		Interactive bool `json:"interactive"`
	} `json:"spec"`
}

func (kubeExec *KubeExec) token(ctx context.Context) (string, error) {
	if kubeExec == nil {
		return "", nil
	}
	if kubeExec.Command == "" {
		return "", fmt.Errorf("kube_exec command is required")
	}

	execCtx := ctx
	var cancel context.CancelFunc
	if kubeExec.TimeoutSeconds > 0 {
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(kubeExec.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, kubeExec.Command, kubeExec.Args...)
	cmd.Env = kubeExecEnv(kubeExec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("kube_exec command timed out after %d seconds", kubeExec.TimeoutSeconds)
		}
		return "", fmt.Errorf("failed to run kube_exec command: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	return parseKubeExecToken(strings.TrimSpace(stdout.String()))
}

func kubeExecEnv(kubeExec *KubeExec) []string {
	env := append([]string{}, os.Environ()...)
	for key, value := range kubeExec.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	_, hasExecInfo := kubeExec.Env["KUBERNETES_EXEC_INFO"]
	if kubeExec.APIVersion != "" && !hasExecInfo {
		info := execInfo{
			Kind:       "ExecCredential",
			APIVersion: kubeExec.APIVersion,
		}
		info.Spec.Interactive = false
		if infoBytes, err := json.Marshal(info); err == nil {
			env = append(env, "KUBERNETES_EXEC_INFO="+string(infoBytes))
		}
	}

	return env
}

func parseKubeExecToken(output string) (string, error) {
	if output == "" {
		return "", fmt.Errorf("kube_exec command returned empty output")
	}

	if !looksLikeJSON(output) {
		return output, nil
	}

	var value interface{}
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		return "", fmt.Errorf("kube_exec output is not valid JSON: %w", err)
	}

	if token, ok := value.(string); ok {
		token = strings.TrimSpace(token)
		if token == "" {
			return "", fmt.Errorf("kube_exec output JSON missing token field")
		}
		return token, nil
	}

	token := extractTokenFromJSONValue(value)
	if token == "" {
		return "", fmt.Errorf("kube_exec output JSON missing token field")
	}

	return token, nil
}

func looksLikeJSON(output string) bool {
	if output == "" {
		return false
	}
	switch output[0] {
	case '{', '[', '"':
		return true
	default:
		return false
	}
}

func extractTokenFromJSONValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			token := extractTokenFromJSONValue(item)
			if token != "" {
				return token
			}
		}
	case map[string]interface{}:
		preferredPaths := [][]string{
			{"status", "token"},
			{"status", "accessToken"},
			{"status", "access_token"},
			{"status", "idToken"},
			{"status", "id_token"},
			{"token"},
			{"accessToken"},
			{"access_token"},
			{"idToken"},
			{"id_token"},
		}
		for _, path := range preferredPaths {
			token := getStringByPath(v, path)
			if token != "" {
				return token
			}
		}
		for key, nestedValue := range v {
			if isTokenKey(key) {
				token, ok := nestedValue.(string)
				if ok && strings.TrimSpace(token) != "" {
					return strings.TrimSpace(token)
				}
			}
			token := extractTokenFromJSONValue(nestedValue)
			if token != "" {
				return token
			}
		}
	}

	return ""
}

func getStringByPath(raw map[string]interface{}, path []string) string {
	var current interface{} = raw
	for _, key := range path {
		object, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		next, ok := getByNormalizedKey(object, key)
		if !ok {
			return ""
		}
		current = next
	}
	value, ok := current.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func getByNormalizedKey(raw map[string]interface{}, lookup string) (interface{}, bool) {
	lookupNorm := normalizeTokenKey(lookup)
	for key, value := range raw {
		if normalizeTokenKey(key) == lookupNorm {
			return value, true
		}
	}
	return nil, false
}

func normalizeTokenKey(key string) string {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	return key
}

func isTokenKey(key string) bool {
	switch normalizeTokenKey(key) {
	case "token", "accesstoken", "idtoken":
		return true
	default:
		return false
	}
}

func redactHelmArgs(args []string) string {
	redacted := make([]string, len(args))
	copy(redacted, args)

	for i := 0; i < len(redacted); i++ {
		if strings.HasPrefix(redacted[i], "--kube-token=") {
			redacted[i] = "--kube-token=REDACTED"
			continue
		}
		if redacted[i] == "--kube-token" && i+1 < len(redacted) {
			redacted[i+1] = "REDACTED"
			i++
		}
	}

	return strings.Join(redacted, " ")
}

func helmCommandNeedsKubeToken(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "dependency":
		return false
	default:
		return true
	}
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

	installScriptPath := filepath.Join(helmDir, "get_helm.sh")

	if err := downloadFile(GET_HELM_URL, installScriptPath); err != nil {
		return "", fmt.Errorf("failed to download Helm installation script: %v", err)
	}

	chmodCmd := exec.Command("chmod", "700", installScriptPath)
	if err := chmodCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set execute permission on Helm installation script: %v", err)
	}

	installHelmCmd := exec.Command(installScriptPath, "--version", helmVersion)
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

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP status %v", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return nil
}
