# TerraHelm Provider

**TerraHelm** is a third-party [Terraform](https://www.terraform.io/) provider that simplifies managing [Helm](https://helm.sh/) releases using [Helm CLI](https://helm.sh/docs/helm/).

Do you need to seamlessly integrate Helm deployments into your Terraform infrastructure, while also considering a future switch to GitOps? TerraHelm is the solution. While Terraform itself isn't ideal for complex Helm orchestration, TerraHelm bridges the gap by incorporating the Helm CLI directly, which allows for efficient debugging and simplifies other Helm tasks within your Terraform workflows. TerraHelm streamlines the process by managing the Helm binary and providing essential configuration options for Helm releases. Furthermore, TerraHelm empowers a flexible IaC approach: it fetches charts and values from various sources, allowing independent storage of Terraform code, charts, and values. This promotes modularity and reusability within your infrastructure definitions, enhancing your infrastructure management now and laying the groundwork for a smooth GitOps transition in the future.

## Features

- **Binary Management**: TerraHelm downloads and installs the specified Helm binary to a designated `cache_dir` directory (`.terraform/terrahelm_cache/`), enabling effortless switching between Helm versions.
- **Integration**: TerraHelm provider supports downloads of charts and values from various sources like Git, Mercurial, HTTP, Amazon S3, Google GCP streamlining the integration of custom Helm charts into your configuration.
- **Easy Debugging**: Need to troubleshoot your Helm deployments? TerraHelm lets you execute the same Helm CLI commands it uses. This simplifies the process by allowing you to replicate the provider's actions directly.

## Documentation

- [Provider Docs](https://registry.terraform.io/providers/mikhae1/terrahelm/latest/docs)

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

### Chart Repository release

```hcl
resource "terrahelm_release" "mysql" {
  name             = "mysql"
  chart_repository = "bitnami"
  chart_path       = "mysql"

  values = data.template_file.values.rendered
}

data "template_file" "values" {
  template = file("values.yaml")
  vars = {
    username = var.username
    password = var.password
  }
}
```

### Git Repository release

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

### Chart Url release

The `chart_url` parameter allows fetching charts from various sources, providing much more flexibility compared to `git_repository`.

```hcl
resource "terrahelm_release" "nginx" {
  name       = "nginx"
  namespace  = "nginx"

  # fetch chart from variety of protocols: http::, file::, s3::, gcs::, hg::
  chart_url  = "github.com/mikhae1/terraform-provider-terrahelm//tests/charts/?ref=master&depth=1"
  chart_path = "./nginx"

  values_files = [
    "./values/nginx/common.yaml", # relative to chart directory
    "https://raw.githubusercontent.com/mikhae1/terraform-provider-terrahelm/master/tests/charts/values/nginx/dev-values.yaml",
  ]
}
```

#### General parameters

- `archive` - The archive format to use to unarchive this file, or "" (empty string) to disable unarchiving
- `checksum` - Checksum to verify the downloaded file or archive (`./chart.tgz?checksum=md5:12345678`, `./chart.tgz?checksum=file:./chart.tgz.sha256sum`)
- `filename` - When in file download mode, allows specifying the name of the downloaded file on disk. Has no effect in directory mode.

Here are examples covering different protocols and key features:

#### Local files

```hcl
resource "terrahelm_release" "local_chart" {
  name      = "local-chart"
  chart_url = "/path/to/chart"
}
```

```hcl
resource "terrahelm_release" "local_chart" {
  name      = "local-chart"
  chart_url = "file::path/to/local/chart.tgz"
}
```

#### Git

```hcl
resource "terrahelm_release" "git_chart" {
  name      = "git-chart"
  chart_url = "github.com/kubernetes/ingress-nginx//charts/ingress-nginx?ref=helm-chart-4.8.3&depth=1"
}
```

```hcl
resource "terrahelm_release" "git_chart" {
  name      = "git-chart"
  chart_url = "git::git@github.com:bitnami/charts.git//bitnami/nginx?depth=1"
}
```

##### Supported parameters

- `ref` - The Git ref to checkout. This is a ref, so it can point to a commit SHA, a branch name, etc.
- `sshkey` - An SSH private key to use during clones. The provided key must be a base64-encoded string. For example, to generate a suitable sshkey from a private key file on disk, you would run base64 -w0 <file>.
- `depth` - The Git clone depth. The provided number specifies the last n revisions to clone from the repository.

#### Mercurial

```hcl
resource "terrahelm_release" "hg_chart" {
  name      = "hg-chart"
  chart_url = "hg::https://example.com/hg/repo//chart?rev=123"
}
```

#### HTTP

```hcl
resource "terrahelm_release" "http_chart" {
  name      = "http-chart"
  chart_url = "https://charts.bitnami.com/bitnami/nginx-15.12.2.tgz//nginx"
}
```

To use HTTP basic authentication, prepend `username:password@` to the hostname.

#### Amazon S3

```hcl
resource "terrahelm_release" "s3_chart" {
  name      = "s3-chart"
  chart_url = "s3::https://s3.amazonaws.com/bucket/chart.tgz"
}
```

```hcl
resource "terrahelm_release" "s3_chart" {
  name      = "s3-chart"
  chart_url = "bucket.s3-eu-west-1.amazonaws.com/bucket/chart"
}
```

##### Supported parameters

- `aws_access_key_id` - AWS access key.
- `aws_access_key_secret` - AWS access key secret.
- `aws_access_token` - AWS access token if this is being used.
- `aws_profile` - Use this profile from local `~/.aws/ config`. Takes priority over key and token.
- `region` - AWS regions to use.

Note: it will also read these from standard AWS environment variables if they're set.

#### Google GCP

```hcl
resource "terrahelm_release" "gcp_chart" {
  name      = "gcp-chart"
  chart_url = "www.googleapis.com/storage/v1/bucket/chart"
}
```

```hcl
resource "terrahelm_release" "gcp_chart" {
  name      = "gcp-chart"
  chart_url = "gcs::https://www.googleapis.com/storage/v1/bucket/chart.zip"
}
```

### Data Source

```hcl
data "terrahelm_release" "nginx" {
  name      = "nginx"
  namespace = "nginx"
}
```

## Using Values and Values Files

### Overview

When deploying Helm charts with this Terraform provider, you have the flexibility to customize the values passed to the Helm chart using either a values string (`values`) or values files (`values_files`). When both `values` and `values_files` are provided, the values from values take precedence over the values from the files. This means that you can override specific values from files by providing them directly in the values parameter.

This section provides examples and explanations of how to use these parameters effectively.

### Values (string)

The values parameter allows you to directly specify the values for the Helm chart in a YAML format as a string. This is useful when the values are relatively simple and can be easily represented inline:

```hcl
resource "helm_release" "example" {
  name           = "my-chart"
  chart_version  = "1.2.3"
  values         = <<EOF
  replicaCount: 3
  image:
    repository: nginx
    tag: "1.19.7"
  ingress:
    enabled: true
    hosts:
      - host: example.com
        paths: ["/"]
  EOF
}
```

### Values Files

The `values_files` parameter allows you to specify one or more YAML files containing values for the Helm chart, whether they are local files, URLs, or files relative to the chart repository. This approach offers a powerful way to store values files outside Terraform in a GitOps compatible manner, enabling the use of separate repositories for the chart and value files. Notably, `values_files` supports most of the parameters from `chart_url`, allowing leverage diverse storage solutions for each individual file.

```hcl
resource "helm_release" "example" {
  name           = "my-chart"
  chart_url      = "github.com/bitnami/charts//bitnami/nginx?depth=1"
  values_files   = [
    "https://example.com/values/common-values.yaml", // Http URL to values file
    "git::https://github.com/org/gitops//values/prod-values.yaml", // Git repository URL to values file
    "s3::https://s3.amazonaws.com/bucket/prod/ingress-values.yaml" // S3 bucket URI to values file
  ]
}
```

Values files will be added to Helm CLI in the order of appearance, and the values from the latest files will override the values from the first ones.


Files starting with `.` are treated as relative to the chart repository itself:

```hcl
resource "terrahelm_release" "chart_values_files" {
  name             = "nginx-values"
  chart_url        = "github.com/mikhae1/terraform-provider-terrahelm//tests/charts/?ref=master&depth=1"
  chart_path       = "./nginx"

  values_files = [
    "./values/nginx/common.yaml",
    "./values/nginx/dev-values.yaml",
  ]
}
```

So the `charts` directory will be downloaded first and `charts/values/nginx/common.yaml`, `charts/values/nginx/dev-values.yaml` will be passed to the Helm CLI.

## Examples

Refer to the examples [here](./examples).

## Troubleshooting

If you encounter issues with Helm release, utilize the Helm CLI for debugging. Set the `TF_LOG=INFO` environment variable to view Helm commands in the provider's logs:

```sh
$ TF_LOG=INFO terraform apply
...
terrahelm_release.nginx: Still creating... [2m50s elapsed]
terrahelm_release.nginx: Still creating... [3m0s elapsed]2023-04-01T18:46:53.636+0300 [INFO]  provider.terraform-provider-terrahelm:

Running Helm command:
  .terraform/terrahelm_cache/helm/v3.7.1/helm install nginx .terraform/terrahelm_cache/repos/nginx-http-490743bd --kube-context my-cluster --namespace nginx --create-namespace --version 13.2.1 -f .terraform/terrahelm_cache/values/charts.git/main/nginx-f6749b77d453441e-values.yaml
...
```

You can now invoke helm commands directly from the command line using the same helm binary and values:

```sh
$ .terraform/terrahelm_cache/helm/v3.7.1/helm ...
```
