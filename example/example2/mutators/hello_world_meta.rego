package hello_world_meta

import rego.v1

add_meta_ops contains operation if {
	object.get(input, "Meta", null) == null

	operation := {
		"op": "add",
		"path": "/Meta",
		"value": {},
	}
}

add_hello_to_meta_ops contains operation if {
	object.get(input, ["Meta", "hello"], null) == null

	operation := {
		"op": "add",
		"path": "/Meta/hello",
		"value": "world",
	}
}

patch := [ operation |
    some ops in [add_meta_ops, add_hello_to_meta_ops]
    operation := ops[_]
]
