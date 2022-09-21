
data clis_check clis {
}

provider "gitops" {
  username = var.git_username
  token = var.git_token
  bin_dir  = data.clis_check.clis
}

resource local_file bin_dir {
  filename = "${path.cwd}/.bin_dir"

  content = data.clis_check.clis
}
