# TerraHelm Provider

The TerraHelm Provider is a plugin for [Terraform](https://www.terraform.io/) that allows managing [Helm](https://helm.sh/) releases using the Helm CLI.

It is important to note that Terraform may not be the best tool for orchestrating Helm deployments, but if you choose to use it, utilizing the Helm CLI itself is the most effective approach. This provider downloads and installs the Helm binary if it is not already installed, and provides the necessary configuration options to connect to a Kubernetes cluster. Using the Helm CLI makes it much easier to debug release installations and perform other Helm-related tasks.

## Installation

In order to build and install the `terraform-provider-terrahelm` provider from source, you need to have Go installed. Please run the following command:

    make install

After that, run Terraform provider initialization:

    terraform init

## Usage

The provider can be used to manage Helm releases using the `terrahelm_release` resource and data source. A `terrahelm_release` resource represents a Helm release that is installed on a Kubernetes cluster.

For example, to install a Helm chart using the TerraHelm Provider:

```hcl
provider "terrahelm" {
  helm_version = "v3.7.1"
  kube_context = "kind-cluster"
}

resource "terrahelm_release" "example" {
  name       = "my-chart"
  chart      = "stable/mysql"
  namespace  = "default"
  values     = [data.template_file.values.rendered]
}

data "template_file" "values" {
  template = file("values.yaml")
  vars = {
    username = var.username
    password = var.password
  }
}
```

## Documentation

- [Provider docs](./docs/index.md)

## Troubleshooting

If you encounter any issues with the TerraHelm Provider, you can use the Helm CLI to debug the issue. You can find the command in the provider's logs by setting `TF_LOG=INFO` environment variable:

```sh
$ TF_LOG=INFO terraform apply
...
terrahelm_release.nginx: Still creating... [2m50s elapsed]
terrahelm_release.nginx: Still creating... [3m0s elapsed]2023-03-30T18:46:53.636+0300 [INFO]  provider.terraform-provider-terrahelm:
  Running helm command:
  .terraform/terrahelm_cache/helm/v3.7.1/helm install nginx .terraform/terrahelm_cache/repos/charts.git/main/bitnami/nginx --kube-context rancher-desktop --namespace nginx --create-namespace --version 13.2.1 -f .terraform/terrahelm_cache/values/charts.git/main/nginx-f6749b77d453441e-values.yaml --logtostderr
```

You can invoke helm commands directly from the command line using the same helm binary that the provider uses. This can be useful for verifying that the Helm binary is working correctly, and for troubleshooting issues with specific Helm releases.
