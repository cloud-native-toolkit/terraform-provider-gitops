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
      "argocd-config" = [for o in local.gitops_entries : o if o.layer == "bootstrap" && o.type == "argocd"][0]
    }
    bootstrap = {
      "argocd-config" = [for o in local.gitops_entries : o if o.layer == "bootstrap" && o.type == "argocd"][0]
    }
    infrastructure = {
      "argocd-config" = [for o in local.gitops_entries : o if o.layer == "infrastructure" && o.type == "argocd"][0]
      payload = [for o in local.gitops_entries : o if o.layer == "infrastructure" && o.type == "payload"][0]
    }
    services = {
      "argocd-config" = [for o in local.gitops_entries : o if o.layer == "services" && o.type == "argocd"][0]
      payload = [for o in local.gitops_entries : o if o.layer == "services" && o.type == "payload"][0]
    }
    applications = {
      "argocd-config" = [for o in local.gitops_entries : o if o.layer == "applications" && o.type == "argocd"][0]
      payload = [for o in local.gitops_entries : o if o.layer == "applications" && o.type == "payload"][0]
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
