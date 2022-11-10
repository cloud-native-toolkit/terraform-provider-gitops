
data clis_check clis {
}

provider "gitops" {
  bin_dir  = data.clis_check.clis.bin_dir

  host = var.git_host
  org  = var.git_org
  repo = local.repo_name
  username = var.git_username
  token = var.git_token
  server_name = "default"
  public = true
}

resource local_file bin_dir {
  filename = "${path.cwd}/.bin_dir"

  content = data.clis_check.clis.bin_dir
}
