package hello_world_meta_test

import data.hello_world_meta.patch

import future.keywords

test_hello_world if {
	e := patch with input as {
		"ID": "my-job",
		"Meta": {},
	}
	e[{
		"op": "add",
		"path": "/Meta/hello",
		"value": "world",
	}]
}

test_hello_world_add_meta if {
	e := patch with input as {"ID": "my-job"}
	count(e) == 2
	trace(sprintf("patch: %v", [e]))

	e == {
		{
			"op": "add",
			"path": "/Meta",
			"value": {},
		},
		{
			"op": "add",
			"path": "/Meta/hello",
			"value": "world",
		},
	}
}

test_hello_world_add_meta_if_meta_null if {
	e := patch with input as {
		"ID": "my-job",
		"Meta": null,
	}
	count(e) == 2
	trace(sprintf("patch: %v", [e]))

	e == {
		{
			"op": "add",
			"path": "/Meta",
			"value": {},
		},
		{
			"op": "add",
			"path": "/Meta/hello",
			"value": "world",
		},
	}
}

test_hello_world_no_code_if_exists if {
	e := patch with input as {
		"ID": "my-job",
		"Meta": {"hello": "world"},
	}
	count(e) == 0
}
