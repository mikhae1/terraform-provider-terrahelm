terraform {
  required_providers {
    terrahelm = {
      source  = "local/mikhae1/terrahelm"
      version = "0.1.0"
    }
  }
}

resource "terrahelm_chart" "nginx" {
  name           = "nginx"
  git_repository = "https://github.com/bitnami/charts.git"
  git_reference  = "main"
  chart_path     = "bitnami/nginx"
  namespace      = "nginx-namespace"

  values = <<EOF
replicaCount: 1
service:
  type: LoadBalancer
EOF
}

output "release_status" {
  value = terrahelm_chart.nginx.release_status
}
