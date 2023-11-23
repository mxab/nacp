validator "opa" "department" {

    opa_rule {
        query = <<EOH
        errors = data.department.errors
        EOH
        filename = "demo1/department.rego"
    }
}
mutator "opa_json_patch" "postgres" {

    opa_rule {
        query = <<EOH
        patch = data.postgres.patch
        EOH
        filename = "demo2/postgres.rego"
    }
}
mutator "opa_json_patch" "logging" {

    opa_rule {
        query = <<EOH
        patch = data.logging.patch
        EOH
        filename = "demo3/logging.rego"
    }
}
