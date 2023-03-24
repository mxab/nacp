mutator "opa_json_patch" "secure_mutator" {

    opa_rule {
        query = <<EOH
        patch = data.secure.patch
        EOH
        filename = "mutators/secure.rego"
    }
}
