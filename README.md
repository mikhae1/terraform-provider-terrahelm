# TerraHelm Provider

**TerraHelm** is a third-party [Terraform](https://www.terraform.io/) provider that simplifies managing [Helm](https://helm.sh/) releases using [Helm CLI](https://helm.sh/docs/helm/).

TerraHelm is a Terraform provider designed to seamlessly integrate Helm chart deployments into your Terraform workflows, offering a unified, atomic, and declarative approach to managing both cloud infrastructure and Kubernetes applications. If you've struggled with the limitations of HashiCorp's native Helm provider, TerraHelm provides a modern alternative built for reliability, simplicity, and tight Terraform integration.

While the native Helm provider handles basic deployments, it falls short when you need to orchestrate complex Helm operations:
- Limited Customization and Configuration Flexibility: The HashiCorp Helm provider primarily focuses on deploying Helm charts from repositories and configuring them using Terraform-defined variables. However, in complex deployments, users often need more nuanced control over the deployment process, such as conditional deployments, complex dependencies between charts, or advanced error handling strategies which can be better handled through direct interaction with the Helm CLI.
- Restricted Access to Helm CLI Features: The HashiCorp provider encapsulates Helm functionality, exposing only a subset of the Helm CLI’s capabilities through Terraform.
- Debugging and Troubleshooting: Helm deployments can sometimes fail due to issues such as template rendering errors, chart dependencies, or configuration mismatches. While the HashiCorp Helm provider does provide error logs, troubleshooting complex issues might require deeper insights into the Helm operations, such as step-by-step execution logs or interactive debugging sessions, which are more directly accessible through the Helm CLI itself.
- Version Management: Complex environments might require specific versions of Helm for different applications or different parts of the same application due to already established CI/CD integrations, legacy code, or security requirements.
- Performance Challenges with Large Deployments: The Helm provider can slow down when deploying large charts or multiple Helm charts due to slow Terraform’s state management logic. TerraHelm mitigates this by using the Helm CLI directly, reducing Terraform state bloat and enhancing performance.
- Integration with External Storages: In complex orchestration tasks, Helm charts might need to interact more extensively with external systems for fetching configuration data or managing secrets, for example. TerraHelm allows for custom scripts and hooks that can dynamically interact with these systems during the deployment process, offering a level of customization that the HashiCorp provider won't support natively (see examples below).

TerraHelm is built specifically with these challenges in mind. It will boost both the efficiency and simplicity of managing Helm-related tasks and provide powerful capabilities, like:
- **Direct Helm CLI Integration:**
TerraHelm invokes the Helm CLI directly rather than wrapping a subset of Helm’s functionality. This ensures you get the full power of Helm, including advanced commands, nuanced configurations, and extensive debugging information during deployments.
- **Enhanced Customization & Debugging:**
With direct helm-cli command logs, troubleshooting becomes more straightforward when something goes wrong.
- **Efficient Binary Management & Version Control:**
Automatically manages the downloading and installation of the appropriate Helm binary into a designated cache (`.terraform/terrahelm_cache/`). This enables effortless switching between Helm versions without manual intervention, reducing the risk of version mismatches or Terraform state bloat.
- **Multiple Chart Sources & Protocols:**
TerraHelm supports retrieving both Helm charts and value files from a wide array of sources including Git, Mercurial, HTTP, Amazon S3, and Google Cloud Storage.
- **Customizable Post-Rendering:**
Seamlessly integrate post-renderers to inject extra configurations or perform manifest transformations before deployment (like secrets rendering).
- **Improved Performance:**
By directly executing Helm commands, TerraHelm minimizes the overhead imposed by Terraform state management logic. This leads to faster deployments—especially when managing large-scale or multiple Helm charts concurrently.

## Documentation

- [Provider Docs](https://registry.terraform.io/providers/mikhae1/terrahelm/latest/docs)

## Installation

To install the TerraHelm Provider, include a provider requirements section in your Terraform code for automatic installation and management:

```hcl
terraform {
  required_providers {
    terrahelm = {
      source  = "mikhae1/terrahelm"
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

## Usage Examples

### Provider configuration

Configure the provider with the desired Helm version and Kubernetes context:

```hcl
provider "terrahelm" {
  helm_version = "v3.9.4"
  kube_context = "your-kube-context"
}
```

### Deploying Helm Chart from a Repository

Deploy a Helm chart from a standard Helm repository:

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

### Deploying a Chart from a folder

```hcl
resource "terrahelm_release" "local_chart" {
  name             = "local-chart"
  chart_repository = "/path/to/charts"
  chart_path       = "my-chart"
}
```

### Deploying a Chart from a Git Repository

Deploy a chart using a Git repository as source:

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

### Data Source

You can use the `data "terrahelm_release"` data source to fetch release information from the Kubernetes cluster.

```hcl
data "terrahelm_release" "nginx" {
  name      = "nginx"
  namespace = "nginx"
}
```

## Advanced Use Cases

### Flexible Chart Source Handling

The `chart_url` parameter allows fetching charts from various sources, providing much more flexibility compared to `git_repository`:

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

#### Supported Chart URL arguments

- `archive` - The archive format to use to unarchive this file, or "" (empty string) to disable unarchiving
- `checksum` - Checksum to verify the downloaded file or archive (`./chart.tgz?checksum=md5:12345678`, `./chart.tgz?checksum=file:./chart.tgz.sha256sum`)
- `filename` - When in file download mode, allows specifying the name of the downloaded file on disk. Has no effect in directory mode.

Here are more examples covering different protocols and key features:

#### Local files

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

**Supported chart_url parameters:**
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

#### HTTP(s)

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

**Supported AWS S3 parameters:**
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

## Flexible Values Handling

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

The `values_files` parameter allows you to specify one or more YAML files containing values for the Helm chart, whether they are local files, URLs, or files relative to the chart repository. This approach offers a powerful way to store values files outside Terraform in a IaC compatible manner, enabling the use of separate repositories for the chart and value files. Notably, `values_files` supports most of the parameters from `chart_url`, allowing leverage diverse storage solutions for each individual file.

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

### Post-Renderer Integration

Helm provides support for [post-renderers](https://helm.sh/docs/topics/advanced/#post-rendering), which allow you to modify the Kubernetes manifests generated by Helm before they are deployed to your cluster. This can be useful for tasks such as:

- Injecting additional configuration or secrets.
- Integrating with [kustomize](https://github.com/thomastaylor312/advanced-helm-demos/tree/master/post-render) or any external systems or tools.
- Performing custom validation or transformation of the manifests.

#### Using Post-Renderers

To configure a post-renderer for a Helm release, you can use the `post_renderer` and `post_renderer_url` arguments in the `terrahelm_release` resource:

```hcl
resource "terrahelm_release" "example" {
  # ... other configuration ...

  # Use existing my-post-renderer binary
  post_renderer = "/path/to/my-post-renderer arg1 arg2"

  # Alternatively, you can specify a URL to download the script
  post_renderer_url = "https://example.com/path/to/my-post-renderer.sh"
}
```

Where:

**post_renderer**:
- This argument specifies the command to run as the post-renderer. The command should accept the rendered Kubernetes manifests on standard input and output the modified manifests on standard output.
- You can also provide additional arguments to the post-renderer command by separating them with spaces.

**post_renderer_url**:
- This argument allows you to specify a URL from where the post-renderer script will be downloaded, same features supported  chat.
- TerraHelm will automatically download the script and make it executable.
- If you only specify `post_renderer_url` without `post_renderer`, the downloaded script will be used as the post-renderer command.

### Safely injecting secrets with TerraHelm

[secfetch](https://github.com/mikhae1/secfetch) allows replace secrets in Helm charts and values using the following placeholder syntax: `{prefix}//{secret-path}//{target-key}`.
It supports AWS SSM, AWS Secrets Manager, Environment variables and others.

Here is how you can use it to replace secrets from values:
```hcl
resource "terrahelm_release" "postrender" {
  name              = "nginx"
  chart_url         = "github.com/mikhae1/terraform-provider-terrahelm//tests/charts/?ref=master&depth=1"
  post_renderer_url = "https://github.com/mikhae1/secfetch/releases/latest/download/secfetch-darwin-amd64.zip"
  chart_path        = "nginx"
  namespace         = "postrender"
  create_namespace  = true
  timeout           = 60
  atomic            = true
  debug             = true

  values_files = [
    "https://raw.githubusercontent.com/mikhae1/terraform-provider-terrahelm/master/tests/charts/values/nginx/common.yaml",
  ]

  # values will be replaced by post renderer just before the deploy to Kubernetes
  values = <<EOF
  basicAuth:
    enabled: true
    username: "base64://YWRtMW4="       // will be replaced to: username: "adm1n"
    password: "base64://cGFzc3cwcmQ="   // will be: password: "passw0rd"
  EOF
}
```

## More examples

Refer to the [examples](./examples) directory for more examples.

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
