mutator "opa_json_patch" "pginject" {

    opa_rule {
        query = <<EOH
        patch = data.pginject.patch
        EOH
        filename = "mutators/pg.rego"
    }
}
