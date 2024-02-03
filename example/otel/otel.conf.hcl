mutator "opa_json_patch" "inject_otel" {

  opa_rule {
    query    = <<EOH
        patch = data.otel.patch
        EOH
    filename = "otel.rego"
  }

}
