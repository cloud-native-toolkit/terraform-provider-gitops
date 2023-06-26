locals {
  application_branch = "main"
  create_operator_group = true
  argocd_namespace = "openshift-gitops"
  ci = true
  sealed_secret_dest = gitops_seal_secrets.no_annotation.dest_dir
}

resource gitops_namespace ns {

  name = var.namespace
  create_operator_group = local.create_operator_group
  argocd_namespace = local.argocd_namespace
  dev_namespace = local.ci
  server_name = var.server_name
  branch = local.application_branch
  config = var.gitops_config
  credentials = var.git_credentials
}

resource gitops_service_account test {
  name = "test-rbac"
  namespace = gitops_namespace.ns.name
  server_name = var.server_name
  branch = local.application_branch
  config = var.gitops_config
  credentials = var.git_credentials
  roles {
    name = "cluster-admin"
  }
  rules {
    api_groups = [""]
    resources = ["configmaps","secrets"]
    verbs = ["*"]
  }
  pull_secrets = ["test"]
}

resource gitops_seal_secrets no_annotation {
  source_dir = "${path.module}/secrets"
  dest_dir   = "${path.cwd}/no-annotation"
  kubeseal_cert = var.kubeseal_cert
  tmp_dir = "${path.cwd}/.tmp/no-annotation"
}

resource gitops_seal_secrets annotation {
  source_dir = "${path.module}/secrets"
  dest_dir   = "${path.cwd}/annotation"
  kubeseal_cert = var.kubeseal_cert
  tmp_dir = "${path.cwd}/.tmp/annotation"
  annotations = ["test=value"]
}

resource random_password docker_password {
  length = 16
}

resource gitops_pull_secret test {
  name = "test-secret"
  namespace = gitops_namespace.ns.name
  server_name = var.server_name
  branch = local.application_branch
  layer = "services"
  credentials = var.git_credentials
  config = var.gitops_config
  kubeseal_cert = var.kubeseal_cert
  registry_server = "quay.io"
  registry_username = "myuser"
  registry_password = random_password.docker_password.result
  secret_name = "mysecret"
}

data gitops_metadata_cluster cluster {
  server_name = var.server_name
  branch = local.application_branch
  credentials = var.git_credentials
  config = var.gitops_config
}

resource null_resource cluster_data {
  triggers = {
    data = {
      ingressSubdomain = data.gitops_metadata_cluster.cluster.default_ingress_subdomain
      ingressSecret = data.gitops_metadata_cluster.cluster.default_ingress_secret
      clusterType = data.gitops_metadata_cluster.cluster.cluster_type
      kubeVersion = data.gitops_metadata_cluster.cluster.kube_version
      openShiftVersion = data.gitops_metadata_cluster.cluster.openshift_version
    }
  }

  provisioner "local-exec" {
    command = "echo 'Cluster config: ${jsonencode(self.triggers.data)}'"
  }
}

data gitops_metadata_packages packages {
  server_name = var.server_name
  branch = local.application_branch
  credentials = var.git_credentials
  config = var.gitops_config
  filter = ["openshift-gitops-operator", "argocd-operator"]
}


resource null_resource package_data {

  provisioner "local-exec" {
    command = "echo 'Package config: ${jsonencode(data.gitops_metadata_packages.packages.packages)}'"
  }
}
