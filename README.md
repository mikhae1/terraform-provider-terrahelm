# TerraHelm Provider

**TerraHelm** is a third-party [Terraform](https://www.terraform.io/) provider designed for managing [Helm](https://helm.sh/) releases via Helm CLI.

If you're currently leveraging Terraform for your infrastructure, contemplating the adoption of an advanced GitOps solution in the future, and in need of an seamless tool for integrating Helm with Terraform, look no further. TerraHelm streamlines the process by downloading and installing the Helm binary and providing essential configuration options for the Helm release management. While Terraform might not be the go-to tool for Helm deployment orchestration, incorporating the Helm CLI directly through the Terraform proves to be a highly effective strategy that simplifies the debugging of release installations and other Helm-related tasks.

## Features

- **Binary Management**: Terrahelm downloads and installs specified Helm binary to the `cache_dir` directory (`.terraform/terrahelm_cache/`). This enables seamless switching between different Helm versions with minimal hassle.
- **Git Repository Integration**: Terrahelm supports direct downloads of charts from Git repositories, streamlining the integration of custom Helm charts into your configuration.
- **Debugging**: Simplifying the troubleshooting process, TerraHelm empowers users to manually execute the same Helm CLI commands utilized by the provider.

## Installation

To install the TerraHelm Provider, include a provider requirements section in your Terraform code for automatic installation and management:

```hcl
terraform {
  required_providers {
    terrahelm = {
      source  = "mikhae1/terrahelm"
      version = ">= 1.0.0"
    }
  }
}
```

### Manual binary installation

1. Download the appropriate binary for your platform from the [Releases](https://github.com/mikhae1/terrahelm/releases/latest) page.
2. Unzip the downloaded binary and move it to the Terraform plugins directory (e.g., `~/.terraform.d/plugins/github.com/mikhae1/terrahelm/1.0.0/linux_amd64/`).

### Build binary from Source

To build and install the `terraform-provider-terrahelm` provider from source, ensure you have Go binary installed and run the following command:

```sh
$ make install
```

## Usage

TerraHelm manages Helm releases through the `terrahelm_release` resource and data source, representing a Helm release installed on a Kubernetes cluster.

### Provider configuration

```hcl
provider "terrahelm" {
  helm_version = "v3.9.4"
  kube_context = "kind"
}
```

### Git Repository Chart release

```hcl
resource "terrahelm_release" "nginx" {
  name             = "nginx"
  git_repository   = "https://github.com/bitnami/charts.git"
  git_reference    = "main"
  chart_path       = "bitnami/nginx"
  namespace        = "nginx"
  create_namespace = true

  values = <<EOF
  replicaCount: 1
  EOF
}
```

### Helm Repository Chart release

```hcl
resource "terrahelm_release" "mysql" {
  name             = "mysql"
  helm_repository  = "bitnami"
  chart_path       = "mysql"

  values = [data.template_file.values.rendered]
}

data "template_file" "values" {
  template = file("values.yaml")
  vars = {
    username = var.username
    password = var.password
  }
}
```

### Data Source

```hcl
data "terrahelm_release" "nginx" {
  name      = "nginx"
  namespace = "nginx"
}
```

Refer to the examples [here](./examples).

## Documentation

- [Provider Docs](./docs/index.md)

## Troubleshooting

If you encounter issues with Helm release, utilize the Helm CLI for debugging. Set the `TF_LOG=INFO` environment variable to view Helm commands in the provider's logs:

```sh
$ TF_LOG=INFO terraform apply
...
terrahelm_release.nginx: Still creating... [2m50s elapsed]
terrahelm_release.nginx: Still creating... [3m0s elapsed]2023-04-01T18:46:53.636+0300 [INFO]  provider.terraform-provider-terrahelm:
  Running helm command:
  .terraform/terrahelm_cache/helm/v3.7.1/helm install nginx .terraform/terrahelm_cache/repos/charts.git/main/bitnami/nginx --kube-context my-cluster --namespace nginx --create-namespace --version 13.2.1 -f .terraform/terrahelm_cache/values/charts.git/main/nginx-f6749b77d453441e-values.yaml --logtostderr
```

You can now invoke helm commands directly from the command line using the same helm binary and values:

```sh
$ .terraform/terrahelm_cache/helm/v3.7.1/helm ...
```
