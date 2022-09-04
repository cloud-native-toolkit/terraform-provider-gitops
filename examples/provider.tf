
module setup_clis {
  source = "github.com/cloud-native-toolkit/terraform-util-clis.git"

  clis = ["gitu", "igc"]
}

provider "gitops" {
  username = var.git_username
  token = var.git_token
  bin_dir  = module.setup_clis.bin_dir
}

resource local_file bin_dir {
  filename = "${path.cwd}/.bin_dir"

  content = module.setup_clis.bin_dir
}
