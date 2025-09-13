package opajsonpatchtesting

errors contains errMsg if {
	errMsg := "This is a error message"
}

warnings contains warnMsg if {
	warnMsg := "This is a warning message"
}

patch contains operation if {
	not input.Meta
	operation := {
		"op": "add",
		"path": "/Meta",
		"value": {},
	}
}

patch contains operation if {
	is_null(input.Meta)
	operation := {
		"op": "add",
		"path": "/Meta",
		"value": {},
	}
}

patch contains operation if {
	not input.Meta.hello
	operation := {
		"op": "add",
		"path": "/Meta/hello",
		"value": "world",
	}
}
