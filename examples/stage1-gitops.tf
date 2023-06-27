resource random_string suffix {
  length = 6

  upper = false
  special = false
}

locals {
  repo_name = "${var.git_repo}-${random_string.suffix.result}"
}

resource gitops_repo repo {
  gitops_namespace = var.gitops_namespace
  sealed_secrets_cert = module.cert.cert
  strict = true
}

data gitops_repo_config repo {
  server_name = gitops_repo.repo.result_server_name
  branch = gitops_repo.repo.result_branch
  bootstrap_url = gitops_repo.repo.url
  username = gitops_repo.repo.result_username
  token = gitops_repo.repo.result_token
  ca_cert = gitops_repo.repo.result_ca_cert
  ca_cert_file = gitops_repo.repo.result_ca_cert_file
}

resource null_resource config_output {
  triggers = {
    config = jsonencode(data.gitops_repo_config.repo.gitops_config)
  }

  provisioner "local-exec" {
    command = "echo 'Config output: ${self.triggers.config}'"
  }
}

resource gitops_repo repo2 {
  depends_on = [gitops_repo.repo]

  host = var.git_host
  org  = var.git_org
  repo = local.repo_name
  username = var.git_username
  token = var.git_token
  public = true
  gitops_namespace = var.gitops_namespace
  sealed_secrets_cert = module.cert.cert
  strict = false
}

resource local_file git_repo {
  filename = "${path.cwd}/.git_repo"

  content = gitops_repo.repo.repo_slug
}

resource local_file git_token {
  filename = "${path.cwd}/.git_token"

  content = var.git_token
}
