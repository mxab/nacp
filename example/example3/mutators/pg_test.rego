package pginject_test

import data.pginject.error
import data.pginject.patch

import rego.v1

test_pginject if {
	patch_ops := patch with input as {"job": {
		"TaskGroups": [{
			"Name": "app",
			"Tasks": [{
				"Meta": {"postgres": "native"},
				"Name": "app",
			}],
		}],
		"Type": "service",
	}}
	count(patch_ops) == 6

	# test each entry of patchops

	print("checking patch_ops[0]")
	patch_ops[0] == {
		"op": "add",
		"path": "/TaskGroups/0/Tasks/0/Vault",
		"value": {
			"ChangeMode": "restart",
			"ChangeSignal": "SIGHUP",
			"Env": true,
			"Namespace": "",
			"Policies": [],
		},
	}
	print("checking patch_ops[1]")
	patch_ops[1] == {
		"op": "add",
		"path": "/TaskGroups/0/Tasks/0/Vault/Policies/-",
		"value": "db-access",
	}

	print("checking patch_ops[2]")
	patch_ops[2] == {
		"op": "add",
		"path": "/TaskGroups/0/Tasks/0/Templates",
		"value": [],
	}
	print("checking patch_ops[3]")
	print(patch_ops[3])
	patch_ops[3] == {
		"op": "add",
		"path": "/TaskGroups/0/Tasks/0/Templates/-",
		"value": {
			"ChangeMode": "restart",
			"ChangeScript": null,
			"ChangeSignal": "",
			"DestPath": "${NOMAD_SECRETS_DIR}/postgres.env",
			"EmbeddedTmpl": "{{ range nomadService \"postgres\" }}\nPGHOSTADDR={{ .Address }}\nPGPORT={{ .Port }}\n{{ end }}\nPGDATABASE=postgres\n{{ with secret \"postgres/creds/dev\" }}\nPGUSER={{ .Data.username }}\nPGPASSWORD={{ .Data.password }}\n{{ end }}",
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
	print("checking patch_ops[4]")
	patch_ops[4] == {
		"op": "add",
		"path": "/TaskGroups/0/Constraints",
		"value": [],
	}
	print("checking patch_ops[5]")
	patch_ops[5] == {
		"op": "add",
		"path": "/TaskGroups/0/Constraints/-",
		"value": {
			"LTarget": "${attr.vault.version}",
			"Operand": "semver",
			"RTarget": ">= 0.6.1",
		},
	}
}

# test_error_foo_not_set if {
# 	err := error with input as {}
# 	err["please set foo"]
# }
# test_error_foo_is_null if {
# 	err := error with input as {"foo": null}
# 	err["please set foo"]
# }
