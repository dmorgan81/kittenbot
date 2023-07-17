variable "prompt" {
  type        = string
  description = "Prompt for image generation"
}

variable "model" {
  type        = string
  description = "Model on dezgo to use for image generation"
}

resource "aws_scheduler_schedule" "kittenbot" {
  name_prefix         = "kittenbot-"
  schedule_expression = "cron(30 0 * * ? *)" # every day at 00:30 (30 minutes after midnight)

  flexible_time_window {
    mode                      = "FLEXIBLE"
    maximum_window_in_minutes = 20
  }

  target {
    arn      = aws_lambda_function.kittenbot.arn
    role_arn = aws_iam_role.eventbridge.arn
    retry_policy {
      maximum_event_age_in_seconds = 300
      maximum_retry_attempts       = 3
    }
  }
}


resource "aws_iam_role" "eventbridge" {
  name_prefix        = "kittenbot-eventbridge-"
  assume_role_policy = data.aws_iam_policy_document.eventbridge_sts.json
}

data "aws_iam_policy_document" "eventbridge_sts" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["scheduler.amazonaws.com"]
    }
  }
}

resource "aws_iam_policy" "eventbridge" {
  name_prefix = "kittenbot-eventbridge-"
  policy      = data.aws_iam_policy_document.eventbridge.json
}

data "aws_iam_policy_document" "eventbridge" {
  statement {
    actions   = ["lambda:InvokeFunction"]
    resources = [aws_lambda_function.kittenbot.arn]
  }
}

resource "aws_iam_role_policy_attachment" "eventbridge" {
  role       = aws_iam_role.eventbridge.name
  policy_arn = aws_iam_policy.eventbridge.arn
}
