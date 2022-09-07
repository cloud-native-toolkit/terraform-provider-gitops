resource random_string suffix {
  length = 6

  upper = false
  special = false
}

locals {
  repo_name = "${var.git_repo}-${random_string.suffix.result}"
}

module "gitops" {
  source = "github.com/cloud-native-toolkit/terraform-tools-gitops"

  host = var.git_host
  org  = var.git_org
  repo = local.repo_name
  token = var.git_token
  public = true
  username = var.git_username
  gitops_namespace = var.gitops_namespace
  sealed_secrets_cert = module.cert.cert
  strict = true
}

resource local_file git_repo {
  filename = "${path.cwd}/.git_repo"

  content = module.gitops.config_repo
}

resource local_file git_token {
  filename = "${path.cwd}/.git_token"

  content = module.gitops.config_token
}
