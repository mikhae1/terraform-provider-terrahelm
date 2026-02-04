package provider

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseKubeExecToken(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		expected  string
		wantError string
	}{
		{
			name:     "raw token",
			output:   "raw-token",
			expected: "raw-token",
		},
		{
			name:     "quoted json token",
			output:   `"quoted-token"`,
			expected: "quoted-token",
		},
		{
			name: "exec credential token",
			output: `{
				"apiVersion":"client.authentication.k8s.io/v1beta1",
				"kind":"ExecCredential",
				"status":{"token":"exec-token"}
			}`,
			expected: "exec-token",
		},
		{
			name: "azure cli token",
			output: `{
				"accessToken":"azure-token"
			}`,
			expected: "azure-token",
		},
		{
			name: "gcloud access token",
			output: `{
				"access_token":"gcp-token"
			}`,
			expected: "gcp-token",
		},
		{
			name:      "invalid json",
			output:    "{invalid}",
			wantError: "not valid JSON",
		},
		{
			name:      "json missing token",
			output:    `{"kind":"ExecCredential"}`,
			wantError: "missing token field",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			token, err := parseKubeExecToken(tc.output)
			if tc.wantError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantError)
				}
				if !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected error containing %q, got %q", tc.wantError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token != tc.expected {
				t.Fatalf("unexpected token: got %q, want %q", token, tc.expected)
			}
		})
	}
}

func TestKubeExecEnvExecInfo(t *testing.T) {
	kubeExec := &KubeExec{
		APIVersion: "client.authentication.k8s.io/v1beta1",
		Env:        map[string]string{"SOME_ENV": "value"},
	}

	env := kubeExecEnv(kubeExec)
	execInfoRaw := envValue(env, "KUBERNETES_EXEC_INFO")
	if execInfoRaw == "" {
		t.Fatal("expected KUBERNETES_EXEC_INFO to be set")
	}

	var info execInfo
	if err := json.Unmarshal([]byte(execInfoRaw), &info); err != nil {
		t.Fatalf("failed to decode KUBERNETES_EXEC_INFO JSON: %v", err)
	}
	if info.Kind != "ExecCredential" {
		t.Fatalf("unexpected kind: %q", info.Kind)
	}
	if info.APIVersion != kubeExec.APIVersion {
		t.Fatalf("unexpected apiVersion: %q", info.APIVersion)
	}
	if !strings.Contains(strings.Join(env, "\n"), "SOME_ENV=value") {
		t.Fatal("expected custom environment variable to be included")
	}
}

func TestKubeExecEnvPreservesProvidedExecInfo(t *testing.T) {
	customInfo := `{"custom":"value"}`
	kubeExec := &KubeExec{
		APIVersion: "client.authentication.k8s.io/v1",
		Env: map[string]string{
			"KUBERNETES_EXEC_INFO": customInfo,
		},
	}

	env := kubeExecEnv(kubeExec)
	value := envValue(env, "KUBERNETES_EXEC_INFO")
	if value != customInfo {
		t.Fatalf("unexpected KUBERNETES_EXEC_INFO: got %q, want %q", value, customInfo)
	}
}

func TestKubeExecTokenResilientFormats(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{
			name:     "raw token from stdout",
			mode:     "raw",
			expected: "raw-token",
		},
		{
			name:     "exec credential json",
			mode:     "exec",
			expected: "exec-token",
		},
		{
			name:     "azure access token json",
			mode:     "azure",
			expected: "azure-token",
		},
		{
			name:     "gcp access token json",
			mode:     "gcp",
			expected: "gcp-token",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			kubeExec := &KubeExec{
				Command: os.Args[0],
				Args: []string{
					"-test.run=TestHelperProcessKubeExec",
					"--",
					tc.mode,
				},
				Env: map[string]string{
					"GO_WANT_HELPER_PROCESS": "1",
				},
				TimeoutSeconds: 5,
			}

			token, err := kubeExec.token(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token != tc.expected {
				t.Fatalf("unexpected token: got %q, want %q", token, tc.expected)
			}
		})
	}
}

func TestKubeExecTokenTimeout(t *testing.T) {
	kubeExec := &KubeExec{
		Command: os.Args[0],
		Args: []string{
			"-test.run=TestHelperProcessKubeExec",
			"--",
			"sleep",
		},
		Env: map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
		},
		TimeoutSeconds: 1,
	}

	start := time.Now()
	_, err := kubeExec.token(context.Background())
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Fatalf("timeout took too long: %s", elapsed)
	}
}

func TestHelperProcessKubeExec(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	sep := -1
	for i, arg := range os.Args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep == -1 || sep+1 >= len(os.Args) {
		os.Exit(2)
	}

	switch os.Args[sep+1] {
	case "sleep":
		time.Sleep(3 * time.Second)
	case "raw":
		os.Stdout.WriteString("raw-token")
	case "exec":
		os.Stdout.WriteString(`{"apiVersion":"client.authentication.k8s.io/v1beta1","kind":"ExecCredential","status":{"token":"exec-token"}}`)
	case "azure":
		os.Stdout.WriteString(`{"accessToken":"azure-token","expiresOn":"2026-01-01 00:00:00.000000"}`)
	case "gcp":
		os.Stdout.WriteString(`{"access_token":"gcp-token","token_type":"Bearer"}`)
	default:
		os.Exit(2)
	}
	os.Exit(0)
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}
