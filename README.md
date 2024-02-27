# TerraHelm Provider

**TerraHelm** is a third-party [Terraform](https://www.terraform.io/) provider designed for managing [Helm](https://helm.sh/) releases via Helm CLI.

Looking to integrate Helm releases seamlessly into your existing Terraform infrastructure while keeping an eye on future GitOps adoption? Look no further. TerraHelm streamlines the process by downloading and installing the Helm binary and providing essential configuration options for the Helm release management. While Terraform itself might not be the ideal tool for the full orchestration of Helm deployments, incorporating the Helm CLI directly through Terraform allows for efficient debugging and simplifies other Helm-related tasks. TerraHelm offers a flexible approach to adopting IaC principles, empowering you to enhance your infrastructure management now while effortlessly laying the groundwork for a full GitOps transition in the future.

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

### When Terraform is not good for Helm Release Management:

- **Longer Release Times**: Compared to using the Helm CLI directly, Terraform introduces additional processing overhead, leading to longer deployment times. This becomes significant as deployment numbers or complexity increases. 
- **Terraform State Lock**: Sometimes it requires manual intervention in situations where the lock has not been properly released.
- **Security Concerns**: Sensitive information, such as passwords or API keys, stored in Terraform state or value templates can introduce security risks if not managed properly.
- **Configuration Drift**: Resources in Kubernetes are dynamic and often change throughout their lifecycle. This leads to configuration drift, where the Terraform state becomes misaligned with the actual state of the resources in the cluster.
- **Limited Functionality**: While the Helm provider offers certain functionalities, it might not provide the full spectrum of features and capabilities available in dedicated Helm CLI for advanced deployment orchestration scenarios.
