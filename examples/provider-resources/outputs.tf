output "name" {
  value       = var.namespace
  description = "Namespace name"
}

output dest1 {
  value = gitops_seal_secrets.no_annotation.dest_dir
}
