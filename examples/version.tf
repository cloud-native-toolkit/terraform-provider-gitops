terraform {
  required_providers {
    gitops = {
      #source  = "cloud-native-toolkit/gitops"
      source  = "terraform.local/local/gitops"
      version = "0.0.0"
    }
  }
}
