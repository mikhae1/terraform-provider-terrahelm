package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"git_repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"git_reference": {
				Type:     schema.TypeString,
				Required: true,
			},
			"chart_path": {
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
				ForceNew: true,
			},
			"create_namespace": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"values": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"chart_version": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"wait": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"atomic": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"timeout": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Computed values for storing additional info in the state
			"release_revision": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"release_chart_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"release_chart_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"release_values": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"release_status": {
				Type:     schema.TypeString,
				Computed: true,
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

	// Clone the Git repository
	tempDir := os.TempDir()
	repoPath := filepath.Join(tempDir, "helm-git-repo")
	defer os.RemoveAll(repoPath)

	cloneCmd := exec.Command("git", "clone", "--branch", gitReference, gitRepository, repoPath)
	if err := cloneCmd.Run(); err != nil {
		return diag.FromErr(fmt.Errorf("failed to clone the Git repository: %s", err))
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
		helmCmd.Args = append(helmCmd.Args, "--set", values)
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

	log.Printf("Running Helm command: %s", helmCmdString)
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
