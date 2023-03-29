terraform {
  required_providers {
    terrahelm = {
      source  = "local/mikhae1/terrahelm"
      version = "0.1.0"
    }
  }
}

provider "terrahelm" {
  helm_version = "v3.7.1"
}

resource "terrahelm_release" "nginx" {
  name           = "nginx1"
  git_repository = "https://github.com/bitnami/charts.git"
  git_reference  = "main"
  chart_path     = "bitnami/nginx"
  namespace      = "nginx"
  timeout        = 60
  chart_version  = "13.2.1"
  # release_chart  = "1.2"

  #   values = <<EOF
  # replicaCount: 2
  # EOF
}

output "release_status" {
  value = terrahelm_release.nginx.release_status
}

# output "release_values" {
#   value = terrahelm_release.nginx.release_values
# }
