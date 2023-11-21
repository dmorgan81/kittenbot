resource "aws_s3_bucket" "kittenbot" {
  bucket_prefix = "kittenbot-"
}

resource "aws_s3_bucket_policy" "kittenbot" {
  bucket = aws_s3_bucket.kittenbot.id
  policy = data.aws_iam_policy_document.kittenbot_bucket.json
}

data "aws_iam_policy_document" "kittenbot_bucket" {
  statement {
    sid = "CloudFrontAccessBucket"

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.kittenbot.arn}/*"]

    condition {
      test     = "StringEquals"
      values   = [aws_cloudfront_distribution.kittenbot.arn]
      variable = "aws:SourceArn"
    }
  }
}
