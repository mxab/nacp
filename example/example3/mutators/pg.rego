package pginject

import rego.v1

vault_policy := "db-access"

add_vault_ops contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres

	object.get(input.TaskGroups[g].Tasks[t], "Vault", null) == null

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Vault", [g, t]),
		"value": {
			"ChangeMode": "restart",
			"ChangeSignal": "SIGHUP",
			"Env": true,
			"Namespace": "",
			"Policies": [],
		},
	}
}

add_vault_policy_ops contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Vault/Policies/-", [g, t]),
		"value": vault_policy,
	}
}

add_env_template_block_ops contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres

	object.get(input.TaskGroups[g].Tasks[t], "Templates", null) == null

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Templates", [g, t]),
		"value": [],
	}
}

add_env_template_ops contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Templates/-", [g, t]),
		"value": {
			"ChangeMode": "restart",
			"ChangeScript": null,
			"ChangeSignal": "",
			"DestPath": "${NOMAD_SECRETS_DIR}/postgres.env",
			"EmbeddedTmpl": env_tmpl,
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

env_tmpl := native_tmpl if {
	input.TaskGroups[g].Tasks[t].Meta.postgres == "native"

	native_tmpl := concat("\n", [
		"{{ range nomadService \"postgres\" }}",
		"PGHOSTADDR={{ .Address }}",
		"PGPORT={{ .Port }}",
		"{{ end }}",
		"PGDATABASE=postgres",
		"{{ with secret \"postgres/creds/dev\" }}",
		"PGUSER={{ .Data.username }}",
		"PGPASSWORD={{ .Data.password }}",
		"{{ end }}",
	])
}

env_tmpl := spring_boot_tmpl if {
	input.TaskGroups[g].Tasks[t].Meta.postgres == "springboot"

	spring_boot_tmpl := concat("\n", [
		"{{ range nomadService \"postgres\" }}",
		"SPRING_DATASOURCE_URL=jdbc:postgresql://{{ .Address }}:{{ .Port }}/postgres",
		"{{ end }}",
		"{{ with secret \"postgres/creds/dev\" }}",
		"SPRING_DATASOURCE_USERNAME={{ .Data.username }}",
		"SPRING_DATASOURCE_PASSWORD={{ .Data.password }}",
		"{{ end }}",
	])
}

add_constaint_block_ops contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres

	object.get(input.TaskGroups[g].Tasks[t], "Constraints", null) == null

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Constraints", [g]),
		"value": [],
	}
}

add_vault_constraint_block_ops contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres

	constraints := object.get(input.TaskGroups[g].Tasks[t], "Constraints", [])
	every constraint in constraints {
		constraint != {
			"LTarget": "${attr.vault.version}",
			"Operand": "semver",
			"RTarget": ">= 0.6.1",
		}
	}

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Constraints/-", [g]),
		"value": {
			"LTarget": "${attr.vault.version}",
			"Operand": "semver",
			"RTarget": ">= 0.6.1",
		},
	}
}

patch := [operation |
	some ops in [
		add_vault_ops,
		add_vault_policy_ops,
		add_env_template_block_ops,
		add_env_template_ops,
		add_constaint_block_ops,
		add_vault_constraint_block_ops,
	]
	operation := ops[_]
]
