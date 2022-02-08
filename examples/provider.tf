
provider "gitops" {
  username = var.git_username
  token = var.git_token
  bin_dir  = module.setup_clis.bin_dir
}
