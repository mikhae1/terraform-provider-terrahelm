package provider

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// MockProviderConfig returns a mock ProviderConfig for testing
func MockProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		CacheDir: os.TempDir(),
		HelmCmd: func(args ...string) *exec.Cmd {
			output := ""
			switch cmd := args[0]; cmd {
			case "list":
				output = `[{"name":"test-helm-release","namespace":"test-namespace","revision":"3","updated":"1999-03-31 09:34:27.199247 +0300 +03","status":"deployed","chart":"nginx-13.2.32","app_version":"1.23.4"}]`
			case "get":
				output = `{"replicaCount":1}`
			default:
				output = "unknown: " + cmd
			}
			return exec.Command("echo", output)
		},
	}
}

var config = MockProviderConfig()

// TestResourceHelmReleaseCreateOrUpdate tests the resourceHelmReleaseCreateOrUpdate function
func TestResourceHelmReleaseCreateOrUpdate(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceHelmRelease().Schema, nil)
	d.Set("name", "test-helm-release")
	d.Set("namespace", "test-namespace")
	d.Set("git_repository", "https://github.com/helm/charts.git")
	d.Set("git_reference", "master")
	d.Set("chart_path", "stable/nginx")

	if diags := resourceHelmReleaseCreateOrUpdate(context.Background(), d, config, true); diags.HasError() {
		t.Fatalf("resourceHelmReleaseCreateOrUpdate update failed: %v", diags)
	}

	if diags := resourceHelmReleaseCreateOrUpdate(context.Background(), d, config, false); diags.HasError() {
		t.Fatalf("resourceHelmReleaseCreateOrUpdate create failed: %v", diags)
	}

	if id := d.Id(); id != "test-namespace/test-helm-release" {
		t.Errorf("unexpected resource ID: %s", id)
	}
}

// TestResourceHelmReleaseRead tests the resourceHelmReleaseRead function
func TestResourceHelmReleaseRead(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceHelmRelease().Schema, nil)
	d.SetId("test-namespace/test-helm-release")
	d.Set("name", "test-helm-release")
	d.Set("namespace", "test-namespace")

	if diags := resourceHelmReleaseRead(context.Background(), d, config); diags.HasError() {
		t.Fatalf("resourceHelmReleaseRead failed: %v", diags)
	}

	if status := d.Get("release_status"); status != "deployed" {
		t.Errorf("unexpected release status: %s", status)
	}
}

// TestJsonMapToStringMap tests the jsonMapToStringMap function
func TestJsonMapToStringMap(t *testing.T) {
	rawValues := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": 42,
			"baz": "hello",
		},
		"qux": true,
	}

	expected := map[string]string{
		"foo.bar": "42",
		"foo.baz": "hello",
		"qux":     "true",
	}

	converted, err := jsonMapToStringMap(rawValues)
	if err != nil {
		t.Fatalf("jsonMapToStringMap failed: %v", err)
	}

	for key, value := range expected {
		if converted[key] != value {
			t.Errorf("unexpected value for key %s: %s", key, converted[key])
		}
	}
}

// TestResourceHelmReleaseDelete tests the resourceHelmReleaseDelete function
func TestResourceHelmReleaseDelete(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceHelmRelease().Schema, nil)
	d.SetId("test-namespace/test-helm-release")
	d.Set("name", "test-helm-release")
	d.Set("namespace", "test-namespace")

	if diags := resourceHelmReleaseDelete(context.Background(), d, config); diags.HasError() {
		t.Fatalf("failed to delete Helm release: %v", diags)
	}

	if id := d.Id(); id != "" {
		t.Errorf("unexpected resource ID: %s", id)
	}
}
