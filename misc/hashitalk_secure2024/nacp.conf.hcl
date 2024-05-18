validator "opa" "enforce_digest" {

  opa_rule {
    query    = <<EOH

        errors = data.digest.errors
        EOH

    filename = "digest/digest.rego"
  }
}

/* //PART2
validator "opa" "verify_image" {

  opa_rule {
    query    = <<EOH

        errors = data.image_verification.errors
        EOH

    filename = "notation/notation.rego"
  }
  notation {
    repo_plain_http   = true
    trust_store_dir   = "/Users/max/Library/Application Support/notation"
    trust_policy_file = "/Users/max/Library/Application Support/notation/trustpolicy.json"
  }
}
*/
