
validator "opa" "some_opa_validator" {

    opa_rule {
        query = "errors = data.costcenter_meta.errors"
        filename = "testdata/opa/validators/costcenter_meta.rego"
    }
}

mutator "opa_json_patch" "some_opa_mutator" {

    opa_rule {
        query = "patch = data.hello_world_meta.patch"
        filename = "testdata/opa/mutators/hello_world_meta.rego"
    }
}
