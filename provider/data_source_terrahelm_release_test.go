package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestDataSourceHelmReleaseRead tests the dataSourceHelmReleaseRead function
func TestDataSourceHelmReleaseRead(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceHelmRelease().Schema, nil)
	d.Set("name", "test-helm-release")
	d.Set("namespace", "test-namespace")

	if diags := dataSourceHelmReleaseRead(context.Background(), d, config); diags.HasError() {
		t.Fatalf("failed to delete Helm release: %v", diags)
	}
	if id := d.Id(); id != "test-namespace/test-helm-release" {
		t.Errorf("unexpected resource ID: %s", id)
	}
	if status := d.Get("release_status"); status != "deployed" {
		t.Errorf("unexpected release status: %s", status)
	}
	if values := strings.TrimSpace(d.Get("values").(string)); values != `replicaCount: 1` {
		t.Errorf("unexpected values: %s, %v", values, []byte(values))
	}
}
