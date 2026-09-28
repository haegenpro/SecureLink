output "bucket_name" {
  description = "Name of the S3 bucket used for uploaded files."
  value       = aws_s3_bucket.files.bucket
}

output "bucket_arn" {
  value = aws_s3_bucket.files.arn
}

output "queue_url" {
  description = "URL of the SQS queue that receives S3 object-created events."
  value       = aws_sqs_queue.events.id
}

output "queue_arn" {
  value = aws_sqs_queue.events.arn
}

output "api_role_arn" {
  description = "IAM role ARN with the permissions the API needs (not attached to any compute by this project)."
  value       = aws_iam_role.api.arn
}

output "worker_role_arn" {
  description = "IAM role ARN with the permissions the worker needs (not attached to any compute by this project)."
  value       = aws_iam_role.worker.arn
}
