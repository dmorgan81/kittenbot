variable "image_tag" {
  type        = string
  description = "Image tag for lambda"
  default     = "latest"
}

locals {
  image_uri = "${aws_ecr_repository.kittenbot.repository_url}@${data.aws_ecr_image.kittenbot.image_digest}"
}

variable "dezgo_key" {
  type        = string
  description = "Dezgo.com API key"
  validation {
    condition     = length(var.dezgo_key) > 6 && substr(var.dezgo_key, 0, 6) == "DEZGO-"
    error_message = "Must start with 'DEZGO-'"
  }
  sensitive = true
}

resource "aws_ssm_parameter" "dezgo_key" {
  name            = "/kittenbot/dezgo-key"
  type            = "SecureString"
  tier            = "Standard"
  allowed_pattern = "^DEZGO-.+$"
  value           = var.dezgo_key
}

variable "prompts" {
  type = list(object({
    model  = string
    prompt = string
  }))
  description = "Models and prompts"
  validation {
    condition     = length(var.prompts) > 1
    error_message = "List of models/prompts must not be empty"
  }
}

resource "aws_ssm_parameter" "prompts" {
  for_each = { for idx, p in var.prompts : idx => format("%s|%s", p.model, p.prompt) }

  name           = format("/kittenbot/prompts/%d", each.key)
  type           = "String"
  insecure_value = each.value
}

resource "aws_lambda_function" "kittenbot" {
  function_name = "kittenbot"
  role          = aws_iam_role.lambda.arn
  image_uri     = local.image_uri
  package_type  = "Image"
  timeout       = 30

  environment {
    variables = {
      "DEZGO_KEY_PARAM" : aws_ssm_parameter.dezgo_key.name
      "PROMPTS_PARAM" : substr(aws_ssm_parameter.prompts[0].name, 0, length(aws_ssm_parameter.prompts[0].name) - 2)
      "BUCKET" : aws_s3_bucket.kittenbot.id
      "DISTRIBUTION" : aws_cloudfront_distribution.kittenbot.id
    }
  }

  depends_on = [aws_cloudwatch_log_group.lambda]
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/kittenbot"
  retention_in_days = 7
  skip_destroy      = true
}

resource "aws_iam_role" "lambda" {
  name_prefix        = "kittenbot-lambda-"
  assume_role_policy = data.aws_iam_policy_document.lambda_sts.json
}

data "aws_iam_policy_document" "lambda_sts" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_policy" "lambda" {
  name_prefix = "kittenbot-lambda-"
  policy      = data.aws_iam_policy_document.lambda.json
}

data "aws_iam_policy_document" "lambda" {
  statement {
    actions   = ["ssm:GetParameter"]
    resources = [aws_ssm_parameter.dezgo_key.arn]
  }

  statement {
    actions   = ["ssm:GetParametersByPath"]
    resources = [substr(aws_ssm_parameter.prompts[0].arn, 0, length(aws_ssm_parameter.prompts[0].arn) - 2)]
  }

  statement {
    actions   = [
      "s3:GetObject",
      "s3:ListBucket",
      "s3:PutObject"
    ]
    resources = ["${aws_s3_bucket.kittenbot.arn}/*"]
  }

  statement {
    actions   = ["cloudfront:CreateInvalidation"]
    resources = [aws_cloudfront_distribution.kittenbot.arn]
  }
}

resource "aws_iam_role_policy_attachment" "lambda" {
  role       = aws_iam_role.lambda.name
  policy_arn = aws_iam_policy.lambda.arn
}

resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}
