
module cntk_namespace {
  source = "./gitops_namespace"

  name = "cntk"
  server_name = module.gitops.server_name
  config = module.gitops.gitops_config
  credentials = module.gitops.git_credentials
}

resource local_file namespace {
  filename = "${path.cwd}/.namespace"

  content = module.cntk_namespace.name
}
