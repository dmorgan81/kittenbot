resource "aws_ecr_repository" "kittenbot" {
  name = "kittenbot"
  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "kittenbot" {
  repository = aws_ecr_repository.kittenbot.name

  policy = <<EOF
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Expire untagged images older than 14 days",
            "selection": {
                "tagStatus": "untagged",
                "countType": "sinceImagePushed",
                "countUnit": "days",
                "countNumber": 14
            },
            "action": {
                "type": "expire"
            }
        },
        {
            "rulePriority": 2,
            "description": "Keep last 30 images",
            "selection": {
                "tagStatus": "any",
                "countType": "imageCountMoreThan",
                "countNumber": 30
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
EOF
}

data "aws_ecr_image" "kittenbot" {
  registry_id     = aws_ecr_repository.kittenbot.registry_id
  repository_name = aws_ecr_repository.kittenbot.name
  image_tag       = var.image_tag
}
