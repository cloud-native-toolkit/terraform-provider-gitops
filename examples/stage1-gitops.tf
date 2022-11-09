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

  content = gitops_repo.repo.token
}
