package otel

import rego.v1

# check for all task if they have a "Meta" field "otel" = "true"
# if it doesn't have a Templates field or its null create a patch for that
# inject the same data as mentioned here: https://github.com/hashicorp/nomad/commit/fb4887505c82346a8f9046f956530058ab92e55a#diff-ad403bc14b99a07b6bf1d5599b9109113bc30d03afd88d7c007dd55f1bdb6b2cR44
add_templates_list_ops contains op if {
	some g, t
	input.job.TaskGroups[g].Tasks[t].Meta.otel == "true"

	object.get(input.job.TaskGroups[g].Tasks[t], "Templates", null) == null

	op := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Templates", [g, t]),
		"value": [],
	}
}

# create template for all task that have "Meta" field "otel" = "true"

add_otel_env_template_ops contains op if {
	some g, t
	input.job.TaskGroups[g].Tasks[t].Meta.otel == "true"

	EmbeddedTmpl := sprintf("%s=%s", ["OTEL_RESOURCE_ATTRIBUTES", concat(
		",",
		[
			"nomad.alloc.id={{ env \"NOMAD_ALLOC_ID\" }}",
			"nomad.alloc.name={{ env \"NOMAD_ALLOC_NAME\" }}",
			"nomad.alloc.index={{ env \"NOMAD_ALLOC_INDEX\" }}",
			"nomad.alloc.createTime={{ timestamp }}",
			# "nomad.eval.id={{ env \"NOMAD_EVAL_ID\" }}",
			"nomad.group.name={{ env \"NOMAD_GROUP_NAME\" }}",
			"nomad.job.id={{ env \"NOMAD_JOB_ID\" }}",
			"nomad.job.name={{ env \"NOMAD_JOB_NAME\" }}",
			"nomad.job.parentId={{ env \"NOMAD_JOB_PARENT_ID\" }}",
			sprintf("nomad.job.type=%s", [object.get(input.job, "Type", "service")]),
			"nomad.namespace={{ env \"NOMAD_NAMESPACE\" }}",
			"nomad.node.id={{ env \"node.unique.id\" }}",
			"nomad.node.name={{ env \"node.unique.name\" }}",
			"nomad.node.datacenter={{ env \"node.datacenter\"}}",
			"nomad.node.class={{ env \"node.class\" }}",
			"nomad.node.address={{ env \"attr.unique.network.ip-address\"}}",
			"nomad.region={{ env \"node.region\" }}",
			"nomad.task.name={{ env \"NOMAD_TASK_NAME\" }}",
			sprintf("nomad.task.driver=%s", [input.job.TaskGroups[g].Tasks[t].Driver]),
		],
	)])
	print("EmbeddedTmpl: ", EmbeddedTmpl)
	op := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Templates/-", [g, t]),
		"value": {
			"EmbeddedTmpl": EmbeddedTmpl,
			"DestPath": "local/otel.env",
			"ChangeMode": "restart",
			"ChangeScript": null,
			"ChangeSignal": "",
			"Envvars": true,
			"ErrMissingKey": false,
			"Gid": null,
			"LeftDelim": "{{",
			"Perms": "0644",
			"RightDelim": "}}",
			"SourcePath": "",
			"Splay": 5000000000,
			"Uid": null,
			"VaultGrace": 0,
			"Wait": null,
		},
	}
}

patch := [op |
	some ops in [
		add_templates_list_ops,
		add_otel_env_template_ops,
	]
	some op in ops
]
