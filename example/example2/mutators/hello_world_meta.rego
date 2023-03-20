package hello_world_meta


patch[operation] {

    not input.Meta
    operation := {
        "op": "add",
        "path": "/Meta",
        "value": {}
    }
}
patch[operation] {

    is_null(input.Meta)
    operation := {
        "op": "add",
        "path": "/Meta",
        "value": {}
    }
}
patch[operation] {

    not input.Meta.hello
    operation := {
        "op": "add",
        "path": "/Meta/hello",
        "value": "world"
    }
}
