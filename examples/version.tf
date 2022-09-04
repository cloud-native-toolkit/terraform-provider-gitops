terraform {
  required_providers {
    gitops = {
      source  = "cloud-native-toolkit/gitops"
      version = "0.0.0"
    }
  }
}
provider_installation {
  filesystem_mirror {
    path    = "/tmp/terraform/providers"
    include = ["registry.terraform.io/cloud-native-toolkit/*"]
  }
  direct {
    exclude = ["registry.terraform.io/cloud-native-toolkit/*"]
  }
}
