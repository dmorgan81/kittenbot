# Kittenbot

The repo contains everything required for deploying [kittenbot.io](https://kittenbot.io). You can also use this repo to create a clone for a different domain.

## Requirements

To successfully deploy this you will need a few things:

* A [Dezgo](https://dezgo.com) API key and some credits in your account. Because we generate a single image a day those credits will last quite a while.
* A [Terraform Cloud](https://app.terraform.io) account.
* A registered domain. Currently this domain is assumed to be registered with AWS.

## Design

At its core this is simply a static website hosted in S3 and served by CloudFront. CloudFront does most of the heavy lifting, including caching.

An EventBridge schedule invokes a lambda every day. The lambda makes a call to Dezgo to generate an image with the passed in prompt and model; the prompt and model are configured via Terraform variables. The lambda then templates out a new `latest.html` and uploads everything to S3. Finally the lambda creates a CloudFront cache invalidation for `latest.html` and the generated image.
