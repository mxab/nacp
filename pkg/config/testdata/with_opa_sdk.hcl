
opa_sdk "example" {
    config_path = "/my/path/to/config.json"
}

validator "opa_bundle" "some_validator" {

    opa_sdk_rule {
        path = "/my/validation/policy"
    }
}

mutator "opa_bundle_json_patch" "some_mutator" {

    opa_sdk_rule {
        path = "/my/mutation/policy"
    }

}
