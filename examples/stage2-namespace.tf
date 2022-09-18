
module cntk_namespace {
  source = "./provider-resources"

  namespace       = "provider-test"
  server_name     = module.gitops.server_name
  gitops_config   = module.gitops.gitops_config
  git_credentials = module.gitops.git_credentials
  kubeseal_cert   = module.gitops.sealed_secrets_cert
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
