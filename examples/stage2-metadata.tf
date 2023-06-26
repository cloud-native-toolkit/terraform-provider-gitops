
resource gitops_metadata metadata {
  server_name = gitops_repo.repo.result_server_name
  branch = gitops_repo.repo.result_branch
  credentials = gitops_repo.repo.git_credentials
  config = gitops_repo.repo.gitops_config
  kube_config_path = module.cluster.platform.kubeconfig
}
