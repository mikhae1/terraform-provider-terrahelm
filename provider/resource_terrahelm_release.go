package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func resourceHelmGitChart() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHelmGitChartCreate,
		ReadContext:   resourceHelmGitChartRead,
		UpdateContext: resourceHelmGitChartUpdate,
		DeleteContext: resourceHelmGitChartDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"git_repository": {
				Type:     schema.TypeString,
				Required: true,
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
			},
			"values": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"release_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// TODO:
func resourceHelmGitChartUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Diagnostics{}
}

// TODO:
func resourceHelmGitChartDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Diagnostics{}
}

// resourceHelmGitChartRead reads the status of the installed Helm chart
func resourceHelmGitChartRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

	// Run 'helm status' to get the release status
	cmd := exec.Command("helm", "status", name, "--namespace", namespace, "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get the status of the Helm release: %s", err))
	}

	// Parse the JSON output to get the release status
	var releaseStatus struct {
		Info struct {
			Status string `json:"status"`
		} `json:"info"`
	}
	if err := json.Unmarshal(output, &releaseStatus); err != nil {
		return diag.FromErr(fmt.Errorf("failed to parse the Helm release status JSON: %s", err))
	}

	// Set the 'release_status' attribute in the Terraform state
	if err := d.Set("release_status", strings.ToLower(releaseStatus.Info.Status)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set release_status: %s", err))
	}

	// Set the resource ID to the release name, which is used to identify the resource in Terraform state
	d.SetId(name)

	return diags
}

// resourceHelmGitChartCreate installs a Helm chart from a Git repository
func resourceHelmGitChartCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	gitRepository := d.Get("git_repository").(string)
	gitReference := d.Get("git_reference").(string)
	chartPath := d.Get("chart_path").(string)
	namespace := d.Get("namespace").(string)
	values := d.Get("values").(string)

	// Clone the Git repository
	tempDir := os.TempDir()
	repoPath := filepath.Join(tempDir, "helm-git-repo")
	if err := os.RemoveAll(repoPath); err != nil {
		return diag.FromErr(fmt.Errorf("failed to clean up the temporary Git repository directory: %s", err))
	}
	defer os.RemoveAll(repoPath)

	cmd := exec.Command("git", "clone", "--branch", gitReference, gitRepository, repoPath)
	if err := cmd.Run(); err != nil {
		return diag.FromErr(fmt.Errorf("failed to clone the Git repository: %s", err))
	}

	// Install the Helm chart
	fullChartPath := filepath.Join(repoPath, chartPath)
	installCmd := exec.Command("helm", "upgrade", "--install", name, fullChartPath, "--namespace", namespace)
	if values != "" {
		installCmd.Args = append(installCmd.Args, "--set", values)
	}

	if err := installCmd.Run(); err != nil {
		return diag.FromErr(fmt.Errorf("failed to install the Helm chart: %s", err))
	}

	// Set the resource ID to the release name, which is used to identify the resource in Terraform state
	d.SetId(name)

	// Read the release status to update the Terraform state
	return resourceHelmGitChartRead(ctx, d, m)
}
