package opajsonpatchtesting

errors[errMsg] {
	errMsg := "This is a error message"
}

warnings[warnMsg] {
	warnMsg := "This is a warning message"
}
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
