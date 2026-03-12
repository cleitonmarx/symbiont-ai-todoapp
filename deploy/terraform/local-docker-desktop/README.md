# local-docker-desktop Terraform stack

This stack deploys:

- `ingress-nginx` controller (Helm)
- the local app Helm chart (`deploy/helm/todoapp`)

to an existing Kubernetes cluster context (default: `docker-desktop`).

Default routing mode is ingress host-based routing on port `80`:

- `todoapp.local` -> HTTP API
- `graphql.todoapp.local` -> GraphQL API

Ingress controller installation is managed by this stack by default (`manage_ingress_nginx = true`).

## `/etc/hosts` mapping

Verify current entries:

```bash
grep -nE 'todoapp\.local|graphql\.todoapp\.local' /etc/hosts || true
```

Add default host mappings:

```bash
echo "127.0.0.1 todoapp.local graphql.todoapp.local" | sudo tee -a /etc/hosts
```

If you set custom values for `http_host` and `graphql_host`, add those names instead.

## Usage

Build the app image locally first (from repository root):

```bash
docker build -t todoapp:dev .
```

Then run Terraform in this stack directory:

```bash
terraform init
terraform apply
```

The built image must match `image_repository` and `image_tag` in `terraform.tfvars`
(defaults: `todoapp` and `dev`).

## Updating the app image

Recommended: use a new image tag on each build, then apply with that tag:

```bash
TAG=dev-$(date +%Y%m%d%H%M%S)
docker build -t todoapp:$TAG .
terraform apply -var="image_tag=$TAG"
```

If you rebuilt the same tag (for example `todoapp:dev`), Terraform may detect no
diff. In that case, force release replacement:

```bash
terraform apply -replace=helm_release.todoapp
```

## Existing ingress-nginx installation

If `ingress-nginx` was already installed manually, import it into this Terraform
state before `terraform apply`:

```bash
terraform import kubernetes_namespace_v1.ingress_nginx[0] ingress-nginx
terraform import helm_release.ingress_nginx[0] ingress-nginx/ingress-nginx
```

To remove resources:

```bash
terraform destroy
```
