variable "kubeconfig_path" {
  description = "Path to kubeconfig file."
  type        = string
  default     = "~/.kube/config"
}

variable "kube_context" {
  description = "Kubeconfig context to target."
  type        = string
  default     = "docker-desktop"
}

variable "namespace" {
  description = "Namespace where the release will be installed."
  type        = string
  default     = "todoapp"
}

variable "release_name" {
  description = "Helm release name."
  type        = string
  default     = "todoapp"
}

variable "chart_path" {
  description = "Local path to Helm chart."
  type        = string
  default     = "../../helm/todoapp"
}

variable "image_repository" {
  description = "Container image repository for app workloads."
  type        = string
  default     = "todoapp"
}

variable "image_tag" {
  description = "Container image tag for app workloads."
  type        = string
  default     = "dev"
}

variable "service_type" {
  description = "Kubernetes service type for HTTP and GraphQL services."
  type        = string
  default     = "ClusterIP"
}

variable "http_node_port" {
  description = "NodePort for HTTP API service when service_type is NodePort."
  type        = number
  default     = 30080
}

variable "graphql_node_port" {
  description = "NodePort for GraphQL API service when service_type is NodePort."
  type        = number
  default     = 30085
}

variable "ingress_enabled" {
  description = "Enable host-based ingress routing on port 80."
  type        = bool
  default     = true
}

variable "manage_ingress_nginx" {
  description = "Whether this stack manages ingress-nginx controller installation."
  type        = bool
  default     = true
}

variable "ingress_nginx_namespace" {
  description = "Namespace for ingress-nginx controller release."
  type        = string
  default     = "ingress-nginx"
}

variable "ingress_nginx_release_name" {
  description = "Helm release name for ingress-nginx controller."
  type        = string
  default     = "ingress-nginx"
}

variable "ingress_nginx_chart_version" {
  description = "Optional chart version for ingress-nginx. Set null to use the latest chart."
  type        = string
  default     = null
  nullable    = true
}

variable "ingress_nginx_values" {
  description = "Extra Helm values for ingress-nginx controller."
  type        = map(any)
  default     = {}
}

variable "ingress_class_name" {
  description = "IngressClass name (for example: nginx)."
  type        = string
  default     = "nginx"
}

variable "http_host" {
  description = "Host/subdomain for HTTP API ingress rule."
  type        = string
  default     = "todoapp.local"
}

variable "graphql_host" {
  description = "Host/subdomain for GraphQL ingress rule."
  type        = string
  default     = "graphql.todoapp.local"
}

variable "ingress_annotations" {
  description = "Annotations applied to the app ingress resource."
  type        = map(string)
  default     = {}
}

variable "llm_api_key" {
  description = "Optional API key for LLM chat/summarization endpoints."
  type        = string
  default     = ""
  sensitive   = true
}

variable "llm_embedding_api_key" {
  description = "Optional API key for embedding endpoint."
  type        = string
  default     = ""
  sensitive   = true
}

variable "mcp_gateway_api_key" {
  description = "Optional API key for MCP gateway."
  type        = string
  default     = ""
  sensitive   = true
}

variable "helm_timeout_seconds" {
  description = "Timeout in seconds for Helm release operations."
  type        = number
  default     = 900
}
