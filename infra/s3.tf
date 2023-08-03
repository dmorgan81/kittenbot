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

resource "aws_s3_access_point" "kittenbot" {
  bucket = aws_s3_bucket.kittenbot.id
  name   = "kittenbot"
}

resource "aws_s3control_object_lambda_access_point" "kittenbot" {
  name = "kittenbot"
  configuration {
    supporting_access_point = aws_s3_access_point.kittenbot.arn

    transformation_configuration {
      actions = ["GetObject"]

      content_transformation {
        aws_lambda {
          function_arn = aws_lambda_function.html.arn
        }
      }
    }
  }
}

resource "aws_s3control_object_lambda_access_point_policy" "kittenbot" {
  name   = aws_s3control_object_lambda_access_point.kittenbot.name
  policy = data.aws_iam_policy_document.kittenbot_ap.json
}

data "aws_iam_policy_document" "kittenbot_ap" {
  statement {
    sid = "CloudFrontAccessAP"

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    actions   = ["s3-object-lambda:Get*"]
    resources = [aws_s3control_object_lambda_access_point.kittenbot.arn]

    condition {
      test     = "StringEquals"
      values   = [aws_cloudfront_distribution.kittenbot.arn]
      variable = "aws:SourceArn"
    }
  }
}
