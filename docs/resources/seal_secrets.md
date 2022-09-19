---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "gitops_seal_secrets Resource - terraform-provider-gitops"
subcategory: ""
description: |-
  
---

# gitops_seal_secrets (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `dest_dir` (String)
- `kubeseal_cert` (String)
- `source_dir` (String)

### Optional

- `annotations` (List of String) The list of annotations that should be added to the generated Sealed Secrets. Expected format of each annotation is a string if 'key=value'
- `tmp_dir` (String) The temporary directory where the cert will be written

### Read-Only

- `id` (String) The ID of this resource.

