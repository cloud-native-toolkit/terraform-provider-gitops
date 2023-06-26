module "cluster" {
  source = "github.com/cloud-native-toolkit/terraform-ocp-login.git"

  server_url = var.server_url
  login_user = var.login_user
  login_password = var.login_password
  login_token = var.login_token
}
