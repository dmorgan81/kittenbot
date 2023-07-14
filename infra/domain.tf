variable "domain" {
  type        = string
  description = "Domain name"
}

data "aws_route53_zone" "zone" {
  name = "${var.domain}."
}

provider "aws" {
  alias  = "aws_us_east_1"
  region = "us-east-1" # CloudFront requires certificates to live in us-east-1
}

resource "aws_route53_record" "domain" {
  for_each = toset([var.domain, "www.${var.domain}"])

  name    = each.key
  type    = "A"
  zone_id = data.aws_route53_zone.zone.id

  alias {
    name                   = aws_cloudfront_distribution.kittenbot.domain_name
    zone_id                = aws_cloudfront_distribution.kittenbot.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_acm_certificate" "cert" {
  domain_name               = var.domain
  validation_method         = "DNS"
  subject_alternative_names = ["*.${var.domain}"]
  provider                  = aws.aws_us_east_1

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "validation" {
  for_each = {
    for dvo in aws_acm_certificate.cert.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  allow_overwrite = true
  name            = each.value.name
  records         = [each.value.record]
  type            = each.value.type
  ttl             = 60
  zone_id         = data.aws_route53_zone.zone.id
}

resource "aws_acm_certificate_validation" "validation" {
  certificate_arn         = aws_acm_certificate.cert.arn
  validation_record_fqdns = [for record in aws_route53_record.validation : record.fqdn]
}
