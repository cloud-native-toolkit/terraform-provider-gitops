locals {
  application_branch = "main"
  create_operator_group = true
  argocd_namespace = "openshift-gitops"
  ci = true
}

resource gitops_namespace ns {

  name = var.namespace
  create_operator_group = local.create_operator_group
  argocd_namespace = local.argocd_namespace
  dev_namespace = local.ci
  server_name = var.server_name
  branch = local.application_branch
  config = yamlencode(var.gitops_config)
  credentials = yamlencode(var.git_credentials)
}

resource gitops_rbac test {
  name = "test-rbac"
  namespace = gitops_namespace.ns.name
  server_name = var.server_name
  branch = local.application_branch
  config = yamlencode(var.gitops_config)
  credentials = yamlencode(var.git_credentials)
  roles {
    name = "cluster-admin"
  }
  rules {
    apiGroup = [""]
    resources = ["configmaps"]
    verbs = [""]
  }
}
