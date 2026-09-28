variable "aws_region" {
  description = "AWS region to provision resources in."
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Short name used as a prefix for resource names."
  type        = string
  default     = "securelink"
}

variable "environment" {
  description = "Deployment environment (e.g. dev, staging, prod), appended to resource names."
  type        = string
  default     = "dev"
}

variable "bucket_name" {
  description = "Name of the S3 bucket used for uploaded files. Must be globally unique."
  type        = string
  default     = ""
}

variable "queue_visibility_timeout_seconds" {
  description = "How long a received SQS message is hidden from other consumers before becoming visible again for retry."
  type        = number
  default     = 60
}

variable "queue_message_retention_seconds" {
  description = "How long SQS retains an unprocessed message before dropping it."
  type        = number
  default     = 345600 # 4 days
}
