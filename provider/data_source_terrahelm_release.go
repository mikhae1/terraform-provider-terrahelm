package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHelmRelease() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceHelmReleaseRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the Helm release",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"namespace": {
				Description: "The Kubernetes namespace where the Helm chart will be installed",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
				ForceNew:    true,
			},
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

func dataSourceHelmReleaseRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

	d.SetId(fmt.Sprintf("%s/%s", namespace, name))

	return resourceHelmReleaseRead(ctx, d, m)
}
