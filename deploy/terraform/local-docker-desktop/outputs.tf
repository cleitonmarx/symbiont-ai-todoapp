output "namespace" {
  description = "Namespace where the app is deployed."
  value       = kubernetes_namespace_v1.todoapp.metadata[0].name
}

output "helm_release_name" {
  description = "Installed Helm release name."
  value       = helm_release.todoapp.name
}

output "http_url" {
  description = "HTTP API URL via ingress host routing."
  value       = "http://${var.http_host}"
}

output "graphql_url" {
  description = "GraphQL URL via ingress host routing."
  value       = "http://${var.graphql_host}"
}
