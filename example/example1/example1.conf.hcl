validator "opa" "costcenter_opa_validator" {

    opa_rule {
        query = <<EOH
        errors = data.costcenter_meta.errors
        EOH
        filename = "validators/costcenter_meta.rego"
    }
}
