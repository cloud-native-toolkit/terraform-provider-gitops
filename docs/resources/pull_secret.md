---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "gitops_pull_secret Resource - terraform-provider-gitops"
subcategory: ""
description: |-
  
---

# gitops_pull_secret (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `config` (String)
- `credentials` (String, Sensitive)
- `kubeseal_cert` (String) The certificate that will be used to encrypt the secret with kubeseal
- `layer` (String)
- `name` (String)
- `namespace` (String)
- `registry_password` (String, Sensitive) The password to the container registry that will be stored in the pull secret
- `registry_server` (String) The host name of the server that will be stored in the pull secret
- `registry_username` (String) The username to the container registry that will be stored in the pull secret

### Optional

- `branch` (String)
- `secret_name` (String) The name of the secret that will be created. If not provided the module name will be used
- `server_name` (String)
- `tmp_dir` (String)
- `type` (String)

### Read-Only

- `id` (String) The ID of this resource.

