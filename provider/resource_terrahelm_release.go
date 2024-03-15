package provider

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"crypto/md5"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"gopkg.in/yaml.v3"
)

func resourceHelmRelease() *schema.Resource {
	return &schema.Resource{
		Description: "Helm chart release deployment",
		CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return resourceHelmReleaseCreateOrUpdate(ctx, d, m, false)
		},
		UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return resourceHelmReleaseCreateOrUpdate(ctx, d, m, true)
		},
		ReadContext:   resourceHelmReleaseRead,
		DeleteContext: resourceHelmReleaseDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the Helm release",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"chart_repository": {
				Description: "URL of the chart repository containing the Helm chart, Helm cli is used for downloading",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"git_repository": {
				Description: "URL of the git repository containing the Helm chart, git cli is used for downloading)",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"chart_url": {
				Description: "URL to the Helm chart, it supports advanced parameters, archives and variety of protocols: http::, file::, s3::, gcs::, hg::",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"git_reference": {
				Description: "Reference (e.g. branch, tag, commit hash) to checkout in the Git repository",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"chart_path": {
				Description: "The relative path to the Helm chart",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"insecure": {
				Description: "Disable checking certificates (not safe)",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"namespace": {
				Description: "The Kubernetes namespace where the Helm chart will be installed",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
				ForceNew:    true,
			},
			"create_namespace": {
				Description: "Whether to create the Kubernetes namespace if it does not exist",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"values": {
				Description: "A YAML string representing the values to be passed to the Helm chart",
				Type:        schema.TypeString,
				Optional:    true,
				StateFunc: func(val interface{}) string {
					safeVal, _ := sanitizeYAMLString(val.(string))
					return safeVal
				},
			},
			"values_files": {
				Description: "A list of the values file names or URLs to be passed to the Helm chart",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"chart_version": {
				Description: "The version of the Helm chart to install",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"wait": {
				Description: "Whether to wait for the Helm chart installation to complete",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					return true
				},
			},
			"atomic": {
				Description: "Whether to roll back the Helm chart installation if it fails",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					return true
				},
			},
			"debug": {
				Description: "Enable debug mode for the Helm CLI",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"timeout": {
				Description: "The maximum time to wait for the Helm chart installation to complete",
				Type:        schema.TypeString,
				Optional:    true,
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					return true
				},
			},
			"custom_args": {
				Description: "Additional arguments to pass to the Helm CLI",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},

			// Computed values for storing additional info in the state
			"release_revision": {
				Description: "The revision of the installed Helm release",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"release_chart_name": {
				Description: "The name of the installed Helm chart",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"release_chart_version": {
				Description: "The version of the installed Helm chart",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"release_values": {
				Description: "The values passed to the Helm chart at installation time",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"release_status": {
				Description: "The current status of the installed Helm release",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
			_, gitRepoOk := d.GetOk("git_repository")
			_, helmRepoOk := d.GetOk("chart_repository")
			_, chartUrlOk := d.GetOk("chart_url")
			_, gitRefOk := d.GetOk("git_reference")

			numFieldsSet := 0
			if gitRepoOk {
				numFieldsSet++
			}
			if helmRepoOk {
				numFieldsSet++
			}
			if chartUrlOk {
				numFieldsSet++
			}

			if numFieldsSet == 0 {
				return fmt.Errorf("either 'git_repository', 'chart_repository', 'chart_url' must be set")
			}
			if numFieldsSet != 1 {
				return fmt.Errorf("only one of 'git_repository', 'chart_repository', or 'chart_url' can be set")
			}
			if gitRefOk && !gitRepoOk {
				return fmt.Errorf("'git_reference' can be used only with 'git_repository'")
			}
			return nil
		},
	}
}

// resourceHelmReleaseDelete deletes Helm release
func resourceHelmReleaseDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

	config := m.(*ProviderConfig)
	cmd := config.HelmCmd("uninstall", name, "--namespace", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to uninstall Helm release: %v, Output: %s", err, output))
	}

	d.SetId("")

	return nil
}

// resourceHelmReleaseRead reads Helm release state
func resourceHelmReleaseRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

	config := m.(*ProviderConfig)

	tflog.Debug(ctx, "getting the Helm chart information")
	listCmd := config.HelmCmd("list", "-n", namespace, "-f", name, "-o", "json")
	output, err := listCmd.Output()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to retrieve Helm chart information: %s", err))
	}

	var helmList []struct {
		Name       string `json:"name"`
		Namespace  string `json:"namespace"`
		Revision   string `json:"revision"`
		Updated    string `json:"updated"`
		Status     string `json:"status"`
		Chart      string `json:"chart"`
		AppVersion string `json:"app_version"`
	}

	if err := json.Unmarshal(output, &helmList); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal Helm chart information: %s", err))
	}

	if len(helmList) == 0 {
		return diag.Errorf("failed to list Helm chart: %s", name)
	}

	helmChart := helmList[0]

	// Capture the respective values from the cluster at current time
	chartParts := strings.Split(helmChart.Chart, "-")
	chartVersion := chartParts[len(chartParts)-1]
	chartName := strings.Join(chartParts[:len(chartParts)-1], "-")
	d.Set("release_chart_name", chartName)
	d.Set("release_chart_version", chartVersion)

	d.Set("release_revision", helmChart.Revision)
	d.Set("release_status", helmChart.Status)

	tflog.Debug(ctx, "getting user Helm values")
	userValuesCmd := config.HelmCmd("get", "values", "-n", namespace, name, "-o", "yaml")
	userValuesOutput, err := userValuesCmd.Output()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to retrieve Helm values: %s", err))
	}

	safeVal, err := sanitizeYAMLString(string(userValuesOutput))
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to sanitize Helm release values: %s", err))
	}
	d.Set("values", safeVal)

	tflog.Debug(ctx, "getting release Helm values")
	valuesCmd := config.HelmCmd("get", "values", "-n", namespace, name, "-a", "-o", "json")
	valuesOutput, err := valuesCmd.Output()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to retrieve Helm release values: %s", err))
	}

	var rawValues map[string]interface{}
	if err := json.Unmarshal(valuesOutput, &rawValues); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal Helm release values: %s", err))
	}

	flatValuesMap, err := jsonMapToStringMap(rawValues)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to convert Helm release values: %s", err))
	}

	if err := d.Set("release_values", flatValuesMap); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// resourceHelmReleaseCreateOrUpdate downloads and installs or upgrades a Helm chart from a given source
func resourceHelmReleaseCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, isUpdate bool) diag.Diagnostics {
	// Retrieve input parameters from the schema
	name := d.Get("name").(string)
	chartRepository := d.Get("chart_repository").(string)
	gitRepository := d.Get("git_repository").(string)
	gitReference := d.Get("git_reference").(string)
	insecure := d.Get("insecure").(bool)
	chartPath := d.Get("chart_path").(string)
	chartURL := d.Get("chart_url").(string)
	namespace := d.Get("namespace").(string)
	createNamespace := d.Get("create_namespace").(bool)
	chartVersion := d.Get("chart_version").(string)
	values := d.Get("values").(string)
	valuesFiles := d.Get("values_files").([]interface{})
	wait := d.Get("wait").(bool)
	atomic := d.Get("atomic").(bool)
	timeout := d.Get("timeout").(string)
	debug := d.Get("debug").(bool)
	customArgs := d.Get("custom_args").([]interface{})

	// Retrieve provider config
	config := m.(*ProviderConfig)
	cacheDir := config.CacheDir

	fullChartPath := filepath.Join(chartRepository, chartPath)
	repoPath := ""

	if chartRepository == "" {
		repoPath = filepath.Join(cacheDir, "repos", name+"-"+generateHash(gitRepository+chartURL))
		fullChartPath = filepath.Join(repoPath, chartPath)

		tflog.Debug(ctx, fmt.Sprintf("Initializing repo directory: '%s'...", repoPath))

		// Remove existing repo path if it exists
		if gitRepository != "" {
			if _, err := os.Stat(repoPath); err == nil {
				if err := os.RemoveAll(repoPath); err != nil {
					return diag.FromErr(fmt.Errorf("failed to delete existing directory: %s", err))
				}
			}
		}

		// Create repo path directory
		if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
			return diag.FromErr(fmt.Errorf("failed to create the directory: %s", err))
		}

		// Clone Git repository if specified
		if gitRepository != "" {
			cloneArgs := []string{"clone", "--depth", "1", "--single-branch"}
			if insecure {
				cloneArgs = append(cloneArgs, "-c", "http.sslVerify=false")
			}
			cloneArgs = append(cloneArgs, "--branch", gitReference, gitRepository, repoPath)
			cloneCmd := exec.Command(config.GitBinPath, cloneArgs...)
			var cloneCmdStderr bytes.Buffer
			cloneCmd.Stderr = &cloneCmdStderr
			tflog.Info(ctx, fmt.Sprintf("Git Repository cloning: '%s'...", gitRepository))
			if err := cloneCmd.Run(); err != nil {
				return diag.FromErr(fmt.Errorf("failed to clone the Git repository: %s\nCommand output: %s", err, cloneCmdStderr.String()))
			}
		}

		// Download chart from URL if specified
		if chartURL != "" {
			client := &getter.Client{
				Src:      chartURL,
				Dst:      repoPath,
				Insecure: insecure,
				Mode:     getter.ClientModeAny,
			}

			tflog.Info(ctx, fmt.Sprintf("Chart URL downloading: '%s' to '%s'...", chartURL, repoPath))
			if err := client.Get(); err != nil {
				return diag.FromErr(fmt.Errorf("failed to fetch the repository: %s\nError: %s", gitRepository, err))
			}
		}

		// Build Helm dependency
		depCmd := config.HelmCmd("dependency", "build", fullChartPath)
		var helmDepStderr bytes.Buffer
		depCmd.Stderr = &helmDepStderr
		tflog.Debug(ctx, fmt.Sprintf("Building Helm dependency: '%s'...", fullChartPath))
		if err := depCmd.Run(); err != nil {
			return diag.FromErr(fmt.Errorf("failed to run 'helm dependency build': %s\nHelm output: %s", err, helmDepStderr.String()))
		}
	}

	// Install or upgrade the Helm chart
	cmd := "install"
	if isUpdate {
		cmd = "upgrade"
	}
	helmCmd := config.HelmCmd(cmd, name, fullChartPath)

	// Prepare values
	valuesPath := filepath.Join(cacheDir, "values", name)
	if values != "" || len(valuesFiles) > 0 {
		if gitReference != "" {
			valuesPath = filepath.Join(valuesPath, gitReference)
		} else if chartRepository != "" {
			valuesPath = filepath.Join(valuesPath, chartRepository)
		}

		if err := os.MkdirAll(valuesPath, os.ModePerm); err != nil {
			return diag.FromErr(fmt.Errorf("failed to create the directory for values: %s", err))
		}
	}

	// Handle values files
	if len(valuesFiles) > 0 {
		var vfPaths []string

		for _, v := range valuesFiles {
			vf := v.(string)
			if strings.HasPrefix(vf, ".") && repoPath != "" {
				vfPaths = append(vfPaths, filepath.Join(repoPath, vf))
			} else {
				vDst := path.Join(valuesPath, fmt.Sprintf("%s-%s-values.yaml", name, generateHash(vf)))
				client := &getter.Client{
					Src:      vf,
					Dst:      vDst,
					Insecure: insecure,
					Mode:     getter.ClientModeFile,
				}

				tflog.Info(ctx, fmt.Sprintf("Value File downloading: '%s' to '%s'...", vf, vDst))
				if err := client.Get(); err != nil {
					return diag.FromErr(fmt.Errorf("failed to fetch the repository: %s\nError: %s", gitRepository, err))
				}
				vfPaths = append(vfPaths, vDst)
			}
		}

		for _, v := range vfPaths {
			helmCmd.Args = append(helmCmd.Args, "-f", v)
		}
	}

	// Handle values string
	if values != "" {
		valuesPath := filepath.Join(cacheDir, "values", chartRepository)
		if gitReference != "" {
			valuesPath = filepath.Join(valuesPath, gitReference)
		} else if chartRepository != "" {
			valuesPath = filepath.Join(valuesPath, chartRepository)
		}

		if err := os.MkdirAll(valuesPath, os.ModePerm); err != nil {
			return diag.FromErr(fmt.Errorf("failed to create the directory: %s", err))
		}

		valuesFilePath := filepath.Join(valuesPath, fmt.Sprintf("%s-%s-values.yaml", name, generateHash(values)))

		if err := os.WriteFile(valuesFilePath, []byte(values), os.ModePerm); err != nil {
			return diag.FromErr(fmt.Errorf("failed to create Helm values file: %s", err))
		}

		helmCmd.Args = append(helmCmd.Args, "-f", valuesFilePath)
	}

	// Append additional Helm command arguments
	if namespace != "" {
		helmCmd.Args = append(helmCmd.Args, "--namespace", namespace)
	}
	if createNamespace {
		helmCmd.Args = append(helmCmd.Args, "--create-namespace")
	}
	if chartVersion != "" {
		helmCmd.Args = append(helmCmd.Args, "--version", chartVersion)
	}
	if wait {
		helmCmd.Args = append(helmCmd.Args, "--wait")
	}
	if atomic {
		helmCmd.Args = append(helmCmd.Args, "--atomic")
	}
	if debug {
		helmCmd.Args = append(helmCmd.Args, "--debug")
	}
	if timeout != "" {
		if _, err := strconv.Atoi(timeout); err == nil {
			timeout = timeout + "s"
		}
		helmCmd.Args = append(helmCmd.Args, "--timeout", timeout)
	}

	// Append custom arguments
	for _, arg := range customArgs {
		helmCmd.Args = append(helmCmd.Args, arg.(string))
	}

	// Execute Helm command
	var helmCmdStdout, helmCmdStderr bytes.Buffer
	helmCmd.Stderr = &helmCmdStderr
	helmCmd.Stdout = &helmCmdStdout
	helmCmdString := strings.Join(helmCmd.Args, " ")
	tflog.Info(ctx, fmt.Sprintf("\n\nRunning Helm command:\n  %s\n\n", helmCmdString))
	if err := helmCmd.Run(); err != nil {
		errMsg := fmt.Sprintf("failed to %s the Helm chart: %s\nHelm command: %s\nHelm output: %s", cmd, err, helmCmdString, helmCmdStderr.String())
		if debug {
			errMsg += fmt.Sprintf("\nHelm stdout: %s", helmCmdStdout.String())
			errMsg += fmt.Sprintf("\nHelm stderr: %s", helmCmdStderr.String())
		}
		return diag.FromErr(fmt.Errorf(errMsg))
	}

	// Set the ID for the resource
	d.SetId(fmt.Sprintf("%s/%s", namespace, name))

	log.Printf("Helm chart %s has been %s(ed) successfully. Helm output:\n%s", name, cmd, helmCmdStdout.String())

	// Read the release status to update the Terraform state
	return resourceHelmReleaseRead(ctx, d, m)
}

func sanitizeYAMLString(yamlString string) (string, error) {
	if strings.TrimSpace(yamlString) == "" {
		return "", nil
	}

	var parsedYAML interface{}
	err := yaml.Unmarshal([]byte(yamlString), &parsedYAML)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	output, err := yaml.Marshal(parsedYAML)
	if err != nil {
		return "", fmt.Errorf("failed to re-serialize YAML: %w", err)
	}

	return string(output), nil
}

func jsonMapToStringMap(rawValues map[string]interface{}) (map[string]string, error) {
	converted := make(map[string]string)

	var traverse func(parentKey string, value interface{}) error

	traverse = func(parentKey string, value interface{}) error {
		switch v := value.(type) {
		case map[string]interface{}:
			for key, value := range v {
				err := traverse(parentKey+"."+key, value)
				if err != nil {
					return err
				}
			}
		default:
			converted[parentKey] = fmt.Sprintf("%v", value)
		}
		return nil
	}

	for key, value := range rawValues {
		err := traverse(key, value)
		if err != nil {
			return nil, err
		}
	}

	return converted, nil
}

func generateHash(input string) string {
	const hashLen = 8

	hash := md5.Sum([]byte(input))
	hashStr := hex.EncodeToString(hash[:])
	if hashLen > 0 && hashLen < len(hashStr) {
		hashStr = hashStr[:hashLen]
	}

	return hashStr
}
