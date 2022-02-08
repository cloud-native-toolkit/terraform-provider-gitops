
module cntk_namespace {
  source = "./gitops_namespace"

  server_name = module.gitops.server_name
  config = jsonencode(module.gitops.gitops_config)
  credentials = jsonencode(module.gitops.git_credentials)
}
