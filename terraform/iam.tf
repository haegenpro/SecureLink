# IAM policies scoped to exactly the S3/SQS actions the API and worker need.
# Roles are provided so a future compute deployment (EC2/ECS/Lambda) can
# attach them; no deployment is implemented by this project, so these roles
# are not attached to any running resource yet.

data "aws_iam_policy_document" "assume_by_ec2" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

# --- API: needs to presign S3 PUT/GET URLs and read/write file metadata ---

data "aws_iam_policy_document" "api_permissions" {
  statement {
    sid    = "S3ObjectReadWrite"
    effect = "Allow"
    actions = [
      "s3:PutObject",
      "s3:GetObject",
    ]
    resources = ["${aws_s3_bucket.files.arn}/*"]
  }
}

resource "aws_iam_policy" "api" {
  name   = "${local.name_prefix}-api-policy"
  policy = data.aws_iam_policy_document.api_permissions.json
}

resource "aws_iam_role" "api" {
  name               = "${local.name_prefix}-api-role"
  assume_role_policy = data.aws_iam_policy_document.assume_by_ec2.json
}

resource "aws_iam_role_policy_attachment" "api" {
  role       = aws_iam_role.api.name
  policy_arn = aws_iam_policy.api.arn
}

# --- Worker: needs to consume SQS and read object metadata (not the S3 -----
# --- object itself; the worker never re-downloads the file) ---------------

data "aws_iam_policy_document" "worker_permissions" {
  statement {
    sid    = "SQSConsume"
    effect = "Allow"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
    ]
    resources = [aws_sqs_queue.events.arn]
  }

  statement {
    sid       = "S3ObjectHeadOnly"
    effect    = "Allow"
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.files.arn}/*"]
  }
}

resource "aws_iam_policy" "worker" {
  name   = "${local.name_prefix}-worker-policy"
  policy = data.aws_iam_policy_document.worker_permissions.json
}

resource "aws_iam_role" "worker" {
  name               = "${local.name_prefix}-worker-role"
  assume_role_policy = data.aws_iam_policy_document.assume_by_ec2.json
}

resource "aws_iam_role_policy_attachment" "worker" {
  role       = aws_iam_role.worker.name
  policy_arn = aws_iam_policy.worker.arn
}
