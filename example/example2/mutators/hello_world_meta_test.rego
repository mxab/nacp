package hello_world_meta_test

import data.hello_world_meta

import rego.v1

test_hello_world if {
	e := hello_world_meta.patch with input as {"job": {
		"ID": "my-job",
		"Meta": {},
	}}
	count(e) == 1
	e == [{
		"op": "add",
		"path": "/Meta/hello",
		"value": "world",
	}]
}

test_hello_world_add_meta if {
	e := hello_world_meta.patch with input as {"job": {"ID": "my-job"}}

	e == [
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
	]
}

test_hello_world_add_meta_if_meta_null if {
	e := hello_world_meta.patch with input as {"job": {
		"ID": "my-job",
		"Meta": null,
	}}
	count(e) == 2

	e == [
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
	]
}

test_hello_world_no_code_if_exists if {
	e := hello_world_meta.patch with input as {"job": {
		"ID": "my-job",
		"Meta": {"hello": "world"},
	}}
	count(e) == 0
}
