
variable "gitops_config" {
  type = string
  description = "Config information regarding the gitops repo structure"
}

variable "git_credentials" {
  type = string
  description = "The credentials for the gitops repo(s)"
  sensitive   = true
}

variable "server_name" {
  type        = string
  description = "The name of the server"
  default     = "default"
}

variable "namespace" {
}

variable "kubeseal_cert" {
  default = ""
}
