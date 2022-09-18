output "name" {
  value       = var.namespace
  description = "Namespace name"
}

output dest1 {
  value = gitops_seal_secrets.no_annotation.dest_dir
}

output dest2 {
  value = gitops_seal_secrets.annotation.dest_dir
}
