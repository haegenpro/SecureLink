#!/bin/sh
# Runs inside the LocalStack container on startup (mounted at
# /etc/localstack/init/ready.d/init.sh) to provision the S3 bucket, SQS
# queue, and S3 -> SQS event notification this project needs, mirroring what
# terraform/ provisions against real AWS.
set -eu

REGION="us-east-1"
BUCKET="securelink-files"
QUEUE="securelink-events"

awslocal s3 mb "s3://${BUCKET}" --region "${REGION}"

QUEUE_URL=$(awslocal sqs create-queue --queue-name "${QUEUE}" --region "${REGION}" --query QueueUrl --output text)
QUEUE_ARN=$(awslocal sqs get-queue-attributes --queue-url "${QUEUE_URL}" --attribute-names QueueArn --region "${REGION}" --query Attributes.QueueArn --output text)

awslocal sqs set-queue-attributes --queue-url "${QUEUE_URL}" --region "${REGION}" --attributes '{
  "Policy": "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"sqs:SendMessage\",\"Resource\":\"'"${QUEUE_ARN}"'\"}]}"
}'

awslocal s3api put-bucket-notification-configuration --bucket "${BUCKET}" --region "${REGION}" --notification-configuration '{
  "QueueConfigurations": [
    {
      "QueueArn": "'"${QUEUE_ARN}"'",
      "Events": ["s3:ObjectCreated:*"]
    }
  ]
}'

echo "LocalStack provisioned: bucket=${BUCKET} queue=${QUEUE_URL}"
