variable "image_tag" {
  type        = string
  description = "Image tag for lambda"
  default     = "latest"
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
    condition = length(var.prompts) > 1
    error_message = "List of models/prompts must not be empty"
  }
}

resource "aws_ssm_parameter" "prompts" {
  for_each = { for idx, p in var.prompts : idx => format("%s|%s", p.model, p.prompt) }

  name           = format("/kittenbot/prompts/%d", each.key)
  type           = "String"
  insecure_value = each.value
}

resource "aws_lambda_function" "image" {
  function_name = "kittenbot-image"
  role          = aws_iam_role.lambda_image.arn
  image_uri     = "${aws_ecr_repository.kittenbot.repository_url}:${var.image_tag}"
  package_type  = "Image"
  timeout       = 30

  image_config {
    command = ["kittenbot-image"]
  }

  environment {
    variables = {
      "DEZGO_KEY_PARAM" : aws_ssm_parameter.dezgo_key.name
      "PROMPTS_PARAM" : substr(aws_ssm_parameter.prompts[0].name, 0, length(aws_ssm_parameter.prompts[0].name) - 2)
      "BUCKET" : aws_s3_bucket.kittenbot.id
      "DISTRIBUTION" : aws_cloudfront_distribution.kittenbot.id
    }
  }

  depends_on = [aws_cloudwatch_log_group.lambda_image]
}

resource "aws_cloudwatch_log_group" "lambda_image" {
  name              = "/aws/lambda/kittenbot-image"
  retention_in_days = 7
  skip_destroy      = true
}

resource "aws_iam_role" "lambda_image" {
  name_prefix        = "kittenbot-image-lambda-"
  assume_role_policy = data.aws_iam_policy_document.lambda_image_sts.json
}

data "aws_iam_policy_document" "lambda_image_sts" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_policy" "lambda_image" {
  name_prefix = "kittenbot-image-lambda-"
  policy      = data.aws_iam_policy_document.lambda_image.json
}

data "aws_iam_policy_document" "lambda_image" {
  statement {
    actions   = ["ssm:GetParameter"]
    resources = [aws_ssm_parameter.dezgo_key.arn]
  }

  statement {
    actions   = ["ssm:GetParametersByPath"]
    resources = [substr(aws_ssm_parameter.prompts[0].arn, 0, length(aws_ssm_parameter.prompts[0].arn) - 2)]
  }

  statement {
    actions = ["s3:PutObject"]
    resources = ["${aws_s3_bucket.kittenbot.arn}/*"]
  }

  statement {
    actions   = ["cloudfront:CreateInvalidation"]
    resources = [aws_cloudfront_distribution.kittenbot.arn]
  }
}

resource "aws_iam_role_policy_attachment" "lambda_image" {
  role       = aws_iam_role.lambda_image.name
  policy_arn = aws_iam_policy.lambda_image.arn
}

resource "aws_iam_role_policy_attachment" "lambda_image_logs" {
  role       = aws_iam_role.lambda_image.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_lambda_function" "html" {
  function_name = "kittenbot-html"
  role          = aws_iam_role.lambda_html.arn
  image_uri     = "${aws_ecr_repository.kittenbot.repository_url}:${var.image_tag}"
  package_type  = "Image"
  timeout       = 30

  image_config {
    command = ["kittenbot-html"]
  }

  environment {
    variables = {
      "BUCKET" : aws_s3_bucket.kittenbot.id
    }
  }

  depends_on = [aws_cloudwatch_log_group.lambda_html]
}

resource "aws_cloudwatch_log_group" "lambda_html" {
  name              = "/aws/lambda/kittenbot-html"
  retention_in_days = 7
  skip_destroy      = true
}

resource "aws_iam_role" "lambda_html" {
  name_prefix        = "kittenbot-html-lambda-"
  assume_role_policy = data.aws_iam_policy_document.lambda_html_sts.json
}

data "aws_iam_policy_document" "lambda_html_sts" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_policy" "lambda_html" {
  name_prefix = "kittenbot-html-lambda-"
  policy      = data.aws_iam_policy_document.lambda_html.json
}

data "aws_iam_policy_document" "lambda_html" {
  statement {
    actions = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.kittenbot.arn}/*"]
  }
}

resource "aws_iam_role_policy_attachment" "lambda_html" {
  role       = aws_iam_role.lambda_html.name
  policy_arn = aws_iam_policy.lambda_html.arn
}

resource "aws_iam_role_policy_attachment" "lambda_html_logs" {
  role       = aws_iam_role.lambda_html.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonS3ObjectLambdaExecutionRolePolicy"
}

resource "aws_lambda_permission" "lambda_html" {
  action = "lambda:InvokeFunction"
  function_name = aws_lambda_function.html.function_name
  principal = "cloudfront.amazonaws.com"
}
