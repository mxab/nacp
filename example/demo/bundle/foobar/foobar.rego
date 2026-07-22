package foobar

import rego.v1

add_meta_ops contains operation if {
	object.get(input.job, "Meta", null) == null

	operation := {
		"op": "add",
		"path": "/Meta",
		"value": {},
	}
}

add_foo_to_meta_ops contains operation if {
	object.get(input.job, ["Meta", "foo"], null) == null

	operation := {
		"op": "add",
		"path": "/Meta/foo",
		"value": "bar",
	}
}

patch := [operation |
	some ops in [add_meta_ops, add_foo_to_meta_ops]
	operation := ops[_]
]
