mutator "opa_json_patch" "hello_world_opa_mutator" {

    resolve_token = true
    opa_rule {
        query = <<EOH
        patch = data.hello_world_meta.patch
        EOH
        filename = "mutators/hello_world_meta.rego"
    }
}
