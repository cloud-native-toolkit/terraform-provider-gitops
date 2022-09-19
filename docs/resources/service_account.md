---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "gitops_service_account Resource - terraform-provider-gitops"
subcategory: ""
description: |-
  
---

# gitops_service_account (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `config` (String)
- `credentials` (String, Sensitive)
- `name` (String)
- `namespace` (String)

### Optional

- `all_service_accounts` (Boolean) Flag indicating the rbac rules should be applied to all service accounts in the namespace
- `branch` (String)
- `cluster_scope` (Boolean) Flag setting the RBAC rules at cluster scope vs namespace scope
- `create_service_account` (Boolean) Flag indicating the service account should be created
- `layer` (String)
- `rbac_namespace` (String) The namespace where the rbac rules should be created. If not provided defaults to the service account namespace
- `roles` (Block List) The list of cluster roles that should be applied to the service account (see [below for nested schema](#nestedblock--roles))
- `rules` (Block List) (see [below for nested schema](#nestedblock--rules))
- `sccs` (List of String) The list of sccs that should be associated with the service account. Supports anyuid and/or privileged
- `server_name` (String)
- `service_account_name` (String)
- `tmp_dir` (String) The temporary directory where config files are written before adding to gitops repo

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--roles"></a>
### Nested Schema for `roles`

Required:

- `name` (String) The name of the cluster role that will be applied


<a id="nestedblock--rules"></a>
### Nested Schema for `rules`

Required:

- `api_groups` (List of String) The apiGroups for the resources in the rule
- `resources` (List of String) The resources targeted by the rule
- `verbs` (List of String) The verbs or actions that can be performed on the resources

Optional:

- `resource_names` (List of String) The names of the resources targeted by the rule

