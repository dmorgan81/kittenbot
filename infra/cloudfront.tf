resource "aws_cloudfront_origin_access_control" "kittenbot" {
  name                              = aws_s3_bucket.kittenbot.bucket_regional_domain_name
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

data "aws_cloudfront_cache_policy" "caching_optimized" {
  name = "Managed-CachingOptimized"
}

resource "aws_cloudfront_response_headers_policy" "kittenbot_png" {
  name = "KittenBotPng"
  custom_headers_config {
    items {
      header   = "X-Robots-Tag"
      override = true
      value    = "noindex"
    }
  }
}

resource "aws_cloudfront_distribution" "kittenbot" {
  enabled = true

  origin {
    domain_name              = aws_s3_bucket.kittenbot.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.kittenbot.id
    origin_id                = aws_s3_bucket.kittenbot.id
  }

  default_root_object = "index.html"
  price_class         = "PriceClass_100"
  is_ipv6_enabled     = true

  default_cache_behavior {
    cache_policy_id        = data.aws_cloudfront_cache_policy.caching_optimized.id
    target_origin_id       = aws_s3_bucket.kittenbot.id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true
  }

  ordered_cache_behavior {
    path_pattern               = "*.png"
    cache_policy_id            = data.aws_cloudfront_cache_policy.caching_optimized.id
    target_origin_id           = aws_s3_bucket.kittenbot.id
    viewer_protocol_policy     = "redirect-to-https"
    allowed_methods            = ["GET", "HEAD"]
    cached_methods             = ["GET", "HEAD"]
    response_headers_policy_id = aws_cloudfront_response_headers_policy.kittenbot_png.id
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
      locations        = []
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}
