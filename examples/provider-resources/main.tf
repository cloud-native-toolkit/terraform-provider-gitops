locals {
  application_branch = "main"
  create_operator_group = true
  argocd_namespace = "openshift-gitops"
  ci = true
}

#resource gitops_namespace ns {
#
#  name = var.namespace
#  create_operator_group = local.create_operator_group
#  argocd_namespace = local.argocd_namespace
#  dev_namespace = local.ci
#  server_name = var.server_name
#  branch = local.application_branch
#  config = yamlencode(var.gitops_config)
#  credentials = yamlencode(var.git_credentials)
#}
#
#resource gitops_service_account test {
#  name = "test-rbac"
#  namespace = gitops_namespace.ns.name
#  server_name = var.server_name
#  branch = local.application_branch
#  config = yamlencode(var.gitops_config)
#  credentials = yamlencode(var.git_credentials)
#  roles {
#    name = "cluster-admin"
#  }
#  rules {
#    api_groups = [""]
#    resources = ["configmaps","secrets"]
#    verbs = ["*"]
#  }
#}

resource gitops_seal_secrets no_annotation {
  source_dir = "${path.module}/secrets"
  dest_dir   = "${path.cwd}/no-annotation"
  kubeseal_cert = var.kubeseal_cert
  tmp_dir = "${path.cwd}/.tmp/no-annotation"
}
#
#resource gitops_seal_secrets annotation {
#  source_dir = "${path.module}/secrets"
#  dest_dir   = "${path.cwd}/annotation"
#  kubeseal_cert = var.kubeseal_cert
#  tmp_dir = "${path.cwd}/.tmp/annotation"
#  annotations = ["test=value"]
#}
