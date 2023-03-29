package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceHelmGitChart() *schema.Resource {
	return &schema.Resource{
		CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return resourceHelmGitChartCreateOrUpdate(ctx, d, m, false)
		},
		UpdateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return resourceHelmGitChartCreateOrUpdate(ctx, d, m, true)
		},
		ReadContext:   resourceHelmGitChartRead,
		DeleteContext: resourceHelmGitChartDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the Helm release",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"git_repository": {
				Description: "URL of the Git repository containing the Helm chart",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"git_reference": {
				Description: "Reference (e.g. branch, tag, commit hash) to checkout in the Git repository",
				Type:        schema.TypeString,
				Required:    true,
			},
			"chart_path": {
				Description: "The path within the Git repository where the Helm chart is located",
				Type:        schema.TypeString,
				Required:    true,
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
				Default:     true,
			},
			"values": {
				Description: "A YAML string representing the values to be passed to the Helm chart",
				Type:        schema.TypeString,
				Optional:    true,
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
			},
			"atomic": {
				Description: "Whether to roll back the Helm chart installation if it fails",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"timeout": {
				Description: "The maximum time to wait for the Helm chart installation to complete",
				Type:        schema.TypeString,
				Optional:    true,
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
	}
}

func resourceHelmGitChartDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

	config := m.(*ProviderConfig)
	helmBinPath := config.HelmBinPath

	cmd := exec.Command(helmBinPath, "uninstall", name, "--namespace", namespace)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to uninstall Helm release: %v, Output: %s", err, output))
	}

	d.SetId("")

	return nil
}

func resourceHelmGitChartRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

	config := m.(*ProviderConfig)
	helmBinPath := config.HelmBinPath

	// Retrieve the Helm chart information
	listCmd := exec.Command(helmBinPath, "list", "-n", namespace, "-f", fmt.Sprintf("^%s$", name), "-o", "json")
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

	// Retrieve the Helm release values
	valuesCmd := exec.Command(helmBinPath, "get", "values", "-n", namespace, name, "-a", "-o", "json")
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

// resourceHelmGitChartCreate installs a Helm chart from a Git repository
func resourceHelmGitChartCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, isUpdate bool) diag.Diagnostics {
	name := d.Get("name").(string)
	gitRepository := d.Get("git_repository").(string)
	gitReference := d.Get("git_reference").(string)
	chartPath := d.Get("chart_path").(string)
	namespace := d.Get("namespace").(string)
	create_namespace := d.Get("create_namespace").(bool)
	chart_version := d.Get("chart_version").(string)
	values := d.Get("values").(string)
	wait := d.Get("wait").(bool)
	atomic := d.Get("atomic").(bool)
	timeout := d.Get("timeout").(string)

	config := m.(*ProviderConfig)
	helmBinPath := config.HelmBinPath
	cacheDir := config.CacheDir

	// Clone the Git repository or use the cache
	repoName := gitRepository[strings.LastIndex(gitRepository, "/")+1:]
	repoPath := filepath.Join(cacheDir, "repos", repoName, gitReference)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
			return diag.FromErr(fmt.Errorf("failed to create the directory: %s", err))
		}

		cloneCmd := exec.Command("git", "clone", "--branch", gitReference, gitRepository, repoPath)
		if err := cloneCmd.Run(); err != nil {
			return diag.FromErr(fmt.Errorf("failed to clone the Git repository: %s", err))
		}
	}

	fullChartPath := filepath.Join(repoPath, chartPath)
	depCmd := exec.Command(helmBinPath, "dependency", "build", "--logtostderr", fullChartPath)
	var helmDepStderr bytes.Buffer
	depCmd.Stderr = &helmDepStderr
	if err := depCmd.Run(); err != nil {
		return diag.FromErr(fmt.Errorf("failed to run 'helm dependency build': %s\nHelm output: %s", err, helmDepStderr.String()))
	}

	// Install the Helm chart
	cmd := "install"
	if isUpdate {
		cmd = "upgrade"
	}

	helmCmd := exec.Command(helmBinPath, cmd, name, fullChartPath)

	if namespace != "" {
		helmCmd.Args = append(helmCmd.Args, "--namespace", namespace)
	}
	if create_namespace {
		helmCmd.Args = append(helmCmd.Args, "--create-namespace")
	}
	if chart_version != "" {
		helmCmd.Args = append(helmCmd.Args, "--version", chart_version)
	}
	if values != "" {
		// Create a YAML file from the "values" string
		hash := sha256.Sum256([]byte(values))
		hashStr := hex.EncodeToString(hash[:8])
		valuesPath := filepath.Join(cacheDir, "values", repoName, gitReference)
		if err := os.MkdirAll(valuesPath, os.ModePerm); err != nil {
			return diag.FromErr(fmt.Errorf("failed to create the directory: %s", err))
		}

		valuesFilePath := filepath.Join(valuesPath, fmt.Sprintf("%s-%s-values.yaml", name, hashStr))

		if err := ioutil.WriteFile(valuesFilePath, []byte(values), os.ModePerm); err != nil {
			return diag.FromErr(fmt.Errorf("failed to create Helm values file: %s", err))
		}

		helmCmd.Args = append(helmCmd.Args, "-f", valuesFilePath)
	}
	if wait {
		helmCmd.Args = append(helmCmd.Args, "--wait")
	}
	if atomic {
		helmCmd.Args = append(helmCmd.Args, "--atomic")
	}
	if timeout != "" {
		if _, err := strconv.Atoi(timeout); err == nil {
			timeout = timeout + "s"
		}
		helmCmd.Args = append(helmCmd.Args, "--timeout", timeout)
	}
	helmCmd.Args = append(helmCmd.Args, "--logtostderr")

	var helmCmdStdout, helmCmdStderr bytes.Buffer
	helmCmd.Stderr = &helmCmdStderr
	helmCmd.Stdout = &helmCmdStdout
	helmCmdString := strings.Join(helmCmd.Args, " ")

	tflog.Info(ctx, "Running Helm command: "+helmCmdString)
	if err := helmCmd.Run(); err != nil {
		return diag.FromErr(fmt.Errorf("failed to install the Helm chart: %s\nHelm output: %s", err, helmCmdStderr.String()))
	}

	// Set the ID for the resource
	d.SetId(fmt.Sprintf("%s/%s", namespace, name))

	log.Printf("Helm chart %s has been installed successfully.\nHelm output: %s", name, helmCmdStdout.String())

	// Read the release status to update the Terraform state
	return resourceHelmGitChartRead(ctx, d, m)
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
