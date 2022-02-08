terraform {
  required_providers {
    gitops = {
      version = "0.1"
      source  = "hashicorp.com/cntk/gitops"
    }
  }
}

resource gitops_namespace ns {
    name = "cntk"
    content_dir = "${path.module}/yaml"
    server_name = var.server_name
    config = var.config
    credentials = var.credentials
}
