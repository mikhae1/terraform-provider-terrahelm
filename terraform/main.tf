terraform {
  required_providers {
    helm-git = {
      source  = "local/mikhae1/helm-git"
      version = "0.1.0"
    }
  }
}

provider "helm-git" {
}

resource "helm_git_chart" "nginx" {
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
  value = helm_git_chart.nginx.release_status
}
