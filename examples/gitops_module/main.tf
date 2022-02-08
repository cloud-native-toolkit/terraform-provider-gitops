terraform {
  required_providers {
    gitops = {
      version = "0.1"
      source  = "hashicorp.com/cntk/gitops"
    }
  }
}

resource gitops_module ns {
    name = var.name
    namespace = var.namespace
    content_dir = "${path.module}/yaml"
    server_name = var.server_name
    layer = var.layer
    type = var.type
    config = var.config
    credentials = var.credentials
}
