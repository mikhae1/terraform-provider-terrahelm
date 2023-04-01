terraform {
  required_providers {
    terrahelm = {
      source  = "local/mikhae1/terrahelm"
      version = "1.0.0"
    }
  }
}

provider "terrahelm" {
  helm_version = "v3.9.1"
  kube_context = "rancher-desktop"
}

resource "terrahelm_release" "nginx" {
  name             = "nginx"
  git_repository   = "https://github.com/bitnami/charts.git"
  git_reference    = "main"
  chart_path       = "bitnami/nginx"
  namespace        = "nginx"
  create_namespace = true
  timeout          = 60
  atomic           = true

  values = <<EOF
  replicaCount: 1
  serviceAccount:
    create: true
    name: nginx
  EOF
}

output "release_status" {
  value = terrahelm_release.nginx.release_status
}

data "terrahelm_release" "nginx" {
  name      = "nginx"
  namespace = "nginx"

  depends_on = [
    terrahelm_release.nginx
  ]
}

output "data_nginx" {
  value = data.terrahelm_release.nginx
}

resource "terrahelm_release" "fluentd" {
  name             = "fluentd"
  helm_repository  = "bitnami"
  chart_path       = "fluentd"
  namespace        = "fluentd"
  create_namespace = true
}

data "terrahelm_release" "fluentd" {
  name      = "fluentd"
  namespace = "fluentd"

  depends_on = [
    terrahelm_release.fluentd
  ]
}

output "data_fluentd" {
  value = data.terrahelm_release.fluentd
}
