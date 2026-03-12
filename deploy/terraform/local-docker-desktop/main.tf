provider "kubernetes" {
  config_path    = pathexpand(var.kubeconfig_path)
  config_context = var.kube_context
}

provider "helm" {
  kubernetes {
    config_path    = pathexpand(var.kubeconfig_path)
    config_context = var.kube_context
  }
}

locals {
  chart_dir                = abspath("${path.module}/${var.chart_path}")
  secret_name              = "${var.release_name}-secrets"
  ingress_nginx_repository = "https://kubernetes.github.io/ingress-nginx"

  helm_values = {
    image = {
      repository = var.image_repository
      tag        = var.image_tag
      pullPolicy = "IfNotPresent"
    }

    services = {
      http = {
        type     = var.service_type
        port     = 8080
        nodePort = var.http_node_port
      }
      graphql = {
        type     = var.service_type
        port     = 8085
        nodePort = var.graphql_node_port
      }
    }

    ingress = {
      enabled     = var.ingress_enabled
      className   = var.ingress_class_name
      annotations = var.ingress_annotations
      hosts = {
        http    = var.http_host
        graphql = var.graphql_host
      }
    }

    env = {
      secrets = {
        create         = false
        existingSecret = local.secret_name
        optional       = true
      }
    }
  }
}

resource "kubernetes_namespace_v1" "ingress_nginx" {
  count = var.manage_ingress_nginx ? 1 : 0

  metadata {
    name = var.ingress_nginx_namespace
  }
}

resource "helm_release" "ingress_nginx" {
  count = var.manage_ingress_nginx ? 1 : 0

  name             = var.ingress_nginx_release_name
  namespace        = kubernetes_namespace_v1.ingress_nginx[0].metadata[0].name
  repository       = local.ingress_nginx_repository
  chart            = "ingress-nginx"
  create_namespace = false
  version          = var.ingress_nginx_chart_version

  values = [yamlencode(var.ingress_nginx_values)]

  atomic          = true
  cleanup_on_fail = true
  wait            = true
  timeout         = var.helm_timeout_seconds

  depends_on = [
    kubernetes_namespace_v1.ingress_nginx,
  ]
}

resource "kubernetes_namespace_v1" "todoapp" {
  metadata {
    name = var.namespace
  }
}

resource "kubernetes_secret_v1" "todoapp_sensitive_env" {
  metadata {
    name      = local.secret_name
    namespace = kubernetes_namespace_v1.todoapp.metadata[0].name
  }

  type = "Opaque"

  data = {
    LLM_API_KEY           = var.llm_api_key
    LLM_EMBEDDING_API_KEY = var.llm_embedding_api_key
    MCP_GATEWAY_API_KEY   = var.mcp_gateway_api_key
  }
}

resource "helm_release" "todoapp" {
  name             = var.release_name
  namespace        = kubernetes_namespace_v1.todoapp.metadata[0].name
  chart            = local.chart_dir
  create_namespace = false

  values = [yamlencode(local.helm_values)]

  atomic          = true
  cleanup_on_fail = true
  wait            = true
  timeout         = var.helm_timeout_seconds

  depends_on = [
    helm_release.ingress_nginx,
    kubernetes_namespace_v1.todoapp,
    kubernetes_secret_v1.todoapp_sensitive_env,
  ]
}
