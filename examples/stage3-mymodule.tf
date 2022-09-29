
module sonarqube {
  source = "github.com/cloud-native-toolkit/terraform-gitops-sonarqube.git?ref=v1.3.0"

  namespace       = module.cntk_namespace.name
  server_name     = gitops_repo.repo.server_name
  gitops_config   = jsondecode(gitops_repo.repo.gitops_config)
  git_credentials = jsondecode(gitops_repo.repo.git_credentials)
  kubeseal_cert   = gitops_repo.repo.sealed_secrets_cert
}

module dashboard {
  source = "github.com/cloud-native-toolkit/terraform-gitops-dashboard.git?ref=v1.7.0"

  namespace       = module.cntk_namespace.name
  server_name     = gitops_repo.repo.server_name
  gitops_config   = jsondecode(gitops_repo.repo.gitops_config)
  git_credentials = jsondecode(gitops_repo.repo.git_credentials)
}
