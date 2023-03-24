terraform {
  required_providers {
    terrahelm = {
      source  = "local/mikhae1/terrahelm"
      version = "0.1.0"
    }
  }
}

resource "terrahelm_release" "nginx" {
  name           = "nginx"
  git_repository = "https://github.com/bitnami/charts.git"
  git_reference  = "main"
  chart_path     = "bitnami/nginx"
  namespace      = "nginx"
  timeout        = 60

  #   values = <<EOF
  # replicaCount: 2
  # EOF
}

output "release_status" {
  value = terrahelm_release.nginx.release_status
}
