mutator "opa_json_patch" "hello_world_opa_mutator" {

    opa_rule {
        query = <<EOH
        patch = data.pginject.patch
        EOH
        filename = "mutators/pg.rego"
    }
}
