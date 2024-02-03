package otel_test

import data.otel
import rego.v1

expected_template_block_patch := {
	"op": "add",
	"path": "/TaskGroups/0/Tasks/0/Templates/-",
	"value": {
		"EmbeddedTmpl": sprintf("%s=%s", ["OTEL_RESOURCE_ATTRIBUTES", concat(
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
				"nomad.job.type=service",
				"nomad.namespace={{ env \"NOMAD_NAMESPACE\" }}",
				"nomad.node.id={{ env \"node.unique.id\" }}",
				"nomad.node.name={{ env \"node.unique.name\" }}",
				"nomad.node.datacenter={{ env \"node.datacenter\"}}",
				"nomad.node.class={{ env \"node.class\" }}",
				"nomad.node.address={{ env \"attr.unique.network.ip-address\"}}",
				"nomad.region={{ env \"node.region\" }}",
				"nomad.task.name={{ env \"NOMAD_TASK_NAME\" }}",
				"nomad.task.driver=docker",
			],
		)]),
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

test_otel_patch if {
	input_job := {
		"ID": "my-job",
		"Name": "my-job",
		"Type": "service",
		"TaskGroups": [{
			"Name": "my-group",
			"Tasks": [{
				"Driver": "docker",
				"Meta": {"otel": "true"},
				"Name": "my-task",
				"Templates": [],
			}],
		}],
	}
	patch_ops := otel.patch with input as input_job

	patch_ops == [expected_template_block_patch]
}
test_otel_patch_default_to_type_service if {
	input_job := {
		"ID": "my-job",
		"Name": "my-job",

		"TaskGroups": [{
			"Name": "my-group",
			"Tasks": [{
				"Driver": "docker",
				"Meta": {"otel": "true"},
				"Name": "my-task",
				"Templates": [],
			}],
		}],
	}
	patch_ops := otel.patch with input as input_job

	patch_ops == [expected_template_block_patch]
}
test_otel_patch_full if {
	input_job := {
		"ID": "my-job",
		"Name": "my-job",
		"Type": "service",
		"TaskGroups": [{
			"Name": "my-group",
			"Tasks": [{
				"Driver": "docker",
				"Meta": {"otel": "true"},
				"Name": "my-task",
			}],
		}],
	}
	patch_ops := otel.patch with input as input_job

	patch_ops == [
		{
			"op": "add",
			"path": "/TaskGroups/0/Tasks/0/Templates",
			"value": [],
		},
		expected_template_block_patch,
	]
}
