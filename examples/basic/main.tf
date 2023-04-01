terraform {
  required_providers {
    terrahelm = {
      source  = "github.com/mikhae1/terrahelm"
      version = "1.0.0"
    }
  }
}

provider "terrahelm" {
  helm_version = "v3.9.4"
  kube_context = "kind"
}

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

output "release_status" {
  value = terrahelm_release.nginx.release_status
}
