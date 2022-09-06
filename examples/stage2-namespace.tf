
module cntk_namespace {
  source = "github.com/cloud-native-toolkit/terraform-gitops-namespace.git?ref=v1.12.2"

  name = "cntk"
  server_name = module.gitops.server_name
  config = module.gitops.gitops_config
  credentials = module.gitops.git_credentials
  ci = false
}

resource local_file namespace {
  filename = "${path.cwd}/.namespace"

  content = module.cntk_namespace.name
}
