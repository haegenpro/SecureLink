locals {
  name_prefix = "${var.project_name}-${var.environment}"
  bucket_name = var.bucket_name != "" ? var.bucket_name : "${local.name_prefix}-files"
}

# --- S3: private bucket for uploaded files ------------------------------

resource "aws_s3_bucket" "files" {
  bucket = local.bucket_name
}

resource "aws_s3_bucket_public_access_block" "files" {
  bucket = aws_s3_bucket.files.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "files" {
  bucket = aws_s3_bucket.files.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_versioning" "files" {
  bucket = aws_s3_bucket.files.id
  versioning_configuration {
    status = "Enabled"
  }
}

# --- SQS: queue for S3 object-created events ----------------------------

resource "aws_sqs_queue" "events" {
  name                       = "${local.name_prefix}-events"
  visibility_timeout_seconds = var.queue_visibility_timeout_seconds
  message_retention_seconds  = var.queue_message_retention_seconds
}

# Allow only this specific S3 bucket to publish to the queue.
resource "aws_sqs_queue_policy" "events" {
  queue_url = aws_sqs_queue.events.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowS3BucketNotification"
        Effect    = "Allow"
        Principal = { Service = "s3.amazonaws.com" }
        Action    = "sqs:SendMessage"
        Resource  = aws_sqs_queue.events.arn
        Condition = {
          ArnEquals = { "aws:SourceArn" = aws_s3_bucket.files.arn }
        }
      }
    ]
  })
}

# --- S3 -> SQS event wiring ----------------------------------------------

resource "aws_s3_bucket_notification" "events" {
  bucket = aws_s3_bucket.files.id

  queue {
    queue_arn = aws_sqs_queue.events.arn
    events    = ["s3:ObjectCreated:*"]
  }

  depends_on = [aws_sqs_queue_policy.events]
}
