
opa_bundle "platform" {
    config_path = "/my/path/to/config.json"

    require_signing = true
}

opa_bundle "team_a" {
    config_path = "/my/path/to/team-a.json"

    ready_timeout    = "45s"
    decision_timeout = "2s"
}

validator "opa_bundle" "some_validator" {

    bundle_rule {
        source = "platform"
        path   = "/my/validation/policy"
    }
}

mutator "opa_bundle_json_patch" "some_mutator" {

    bundle_rule {
        source = "team_a"
        path   = "/my/mutation/policy"
    }

}
