resource random_string suffix {
  length = 6

  upper = false
  special = false
}

locals {
  repo_name = "${var.git_repo}-${random_string.suffix.result}"
  gitops_entries = gitops_repo.repo.gitops_config
  gitops_config = {
    boostrap = {
      "argocd-config" = local.gitops_entries["bootstrap"]["argocd"]
    }
    bootstrap = {
      "argocd-config" = local.gitops_entries["bootstrap"]["argocd"]
    }
    infrastructure = {
      "argocd-config" = local.gitops_entries["infrastructure"]["argocd"]
      payload = local.gitops_entries["infrastructure"]["payload"]
    }
    services = {
      "argocd-config" = local.gitops_entries["services"]["argocd"]
      payload = local.gitops_entries["services"]["payload"]
    }
    applications = {
      "argocd-config" = local.gitops_entries["applications"]["argocd"]
      payload = local.gitops_entries["applications"]["payload"]
    }
  }
}

resource gitops_repo repo {
  host = var.git_host
  org  = var.git_org
  repo = local.repo_name
  username = var.git_username
  token = var.git_token
  public = true
  gitops_namespace = var.gitops_namespace
  sealed_secrets_cert = module.cert.cert
  strict = true
}

resource local_file git_repo {
  filename = "${path.cwd}/.git_repo"

  content = gitops_repo.repo.repo_slug
}

resource local_file git_token {
  filename = "${path.cwd}/.git_token"

  content = gitops_repo.repo.token
}
