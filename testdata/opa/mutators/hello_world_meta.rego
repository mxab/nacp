package hello_world_meta

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
