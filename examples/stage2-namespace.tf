
module cntk_namespace {
  source = "./provider-resources"

  namespace       = "provider-test"
  server_name     = module.gitops.server_name
  gitops_config   = module.gitops.gitops_config
  git_credentials = module.gitops.git_credentials
}

resource local_file namespace {
  filename = "${path.cwd}/.namespace"

  content = module.cntk_namespace.name
}
git
