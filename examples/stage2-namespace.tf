
module cntk_namespace {
  source = "./provider-resources"

  namespace       = "provider-test"
  server_name     = gitops_repo.repo.server_name
  gitops_config   = jsonencode(local.gitops_config)
  git_credentials = jsonencode(gitops_repo.repo.git_credentials)
  kubeseal_cert   = gitops_repo.repo.sealed_secrets_cert
}

resource local_file namespace {
  filename = "${path.cwd}/.namespace"

  content = module.cntk_namespace.name
}

resource local_file dest1 {
  filename = "${path.cwd}/.dest1"

  content = module.cntk_namespace.dest1
}

resource local_file dest2 {
  filename = "${path.cwd}/.dest2"

  content = module.cntk_namespace.dest2
}
