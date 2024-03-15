terraform {
  required_providers {
    terrahelm = {
      source = "local/mikhae1/terrahelm"
    }
  }
}

provider "terrahelm" {
  helm_version = "v3.9.1"
  kube_context = "rancher-desktop"
}

#
# git_repository
#
resource "terrahelm_release" "git_repository" {
  name           = "example"
  git_repository = "https://github.com/helm/examples.git"
  git_reference  = "main"
  chart_path     = "charts/hello-world"
  timeout        = 60
  atomic         = true
  insecure       = true

  values = <<EOF
  replicaCount: 1
  serviceAccount:
    create: true
  EOF
}

#
# chart_url
#
resource "terrahelm_release" "chart_url_http" {
  name             = "traefik"
  chart_url        = "https://traefik.github.io/charts/traefik/traefik-26.1.0.tgz//traefik"
  namespace        = "traefik"
  create_namespace = true
  timeout          = 60
  atomic           = true
}

resource "terrahelm_release" "chart_url" {
  name             = "nginx"
  chart_url        = "github.com/kubernetes/ingress-nginx//charts/ingress-nginx?ref=helm-chart-4.8.3&depth=1"
  namespace        = "nginx"
  create_namespace = true
  timeout          = 60
  atomic           = true
}


output "release_status" {
  value = terrahelm_release.git_repository.release_status
}

data "terrahelm_release" "nginx" {
  name      = "nginx"
  namespace = "nginx"

  depends_on = [
    terrahelm_release.chart_url
  ]
}

output "data_nginx" {
  value = data.terrahelm_release.nginx
}

#
# chart_repository
#
resource "terrahelm_release" "chart_repository" {
  name             = "fluentd"
  chart_repository = "bitnami"
  chart_path       = "fluentd"
  namespace        = "fluentd"
  create_namespace = true
  debug            = true
}

data "terrahelm_release" "fluentd" {
  name      = "fluentd"
  namespace = "fluentd"

  depends_on = [
    terrahelm_release.chart_repository
  ]
}

output "data_fluentd" {
  value = data.terrahelm_release.fluentd
}

#
# values_files
#
resource "terrahelm_release" "chart_values_files" {
  name             = "nginx-values"
  chart_url        = "github.com/mikhae1/terraform-provider-terrahelm//tests/charts/?ref=master&depth=1"
  chart_path       = "./nginx"
  namespace        = "nginx-values"
  create_namespace = true
  timeout          = 60
  atomic           = true

  values_files = [
    "https://raw.githubusercontent.com/mikhae1/terraform-provider-terrahelm/master/tests/charts/values/nginx/common.yaml",
    "./values/nginx/dev-values.yaml",
  ]

  values = <<EOF
  serviceAccount:
    name: overriden
  EOF
}
