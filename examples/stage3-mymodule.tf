
module sonarqube {
  source = "github.com/cloud-native-toolkit/terraform-gitops-sonarqube.git?ref=v1.3.0"

  namespace       = module.cntk_namespace.name
  gitops_config   = module.gitops.gitops_config
  git_credentials = module.gitops.git_credentials
  kubeseal_cert   = module.gitops.sealed_secrets_cert
  server_name     = module.gitops.server_name
}

module dashboard {
  source = "github.com/cloud-native-toolkit/terraform-gitops-dashboard.git?ref=v1.7.0"

  namespace       = module.cntk_namespace.name
  gitops_config   = module.gitops.gitops_config
  git_credentials = module.gitops.git_credentials
  server_name     = module.gitops.server_name
}
