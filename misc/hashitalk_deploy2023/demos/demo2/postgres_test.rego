package postgres

import future.keywords

test_patch_vault_policy if {
	patch_ops := patch with input as {
		"ID": "app",
		"Name": "my-app",
		"TaskGroups": [{
			"Name": "app",
			"Tasks": [{
				"Meta": {"postgres": "libpq"},
				"Name": "app",
			}],
		}],
	}
	patch_ops == {
		{
			"op": "add",
			"path": "/TaskGroups/0/Tasks/0/Vault",
			"value": {
				"ChangeMode": "restart",
				"ChangeSignal": "SIGHUP",
				"Env": true,
				"Namespace": "",
				"Policies": [],
			},
		},
		{
			"op": "add",
			"path": "/TaskGroups/0/Tasks/0/Vault/Policies/-",
			"value": "my-app-db-access",
		},
		{
			"op": "add",
			"path": "/TaskGroups/0/Tasks/0/Templates",
			"value": [],
		},
		{
			"op": "add",
			"path": "/TaskGroups/0/Tasks/0/Templates/-",
			"value": {
				"ChangeMode": "restart",
				"DestPath": "${NOMAD_SECRETS_DIR}/postgres.env",
				"EmbeddedTmpl": "PGDATABASE=my_app\n{{ with secret \"db/my_app/creds/admin\" }}\nPGUSER={{ .Data.username }}\nPGPASSWORD={{ .Data.password }}\n{{ end }}\n{{ range nomadService \"postgres\" }}\nPGHOSTADDR={{ .Address }}\nPGPORT={{ .Port }}\n{{ end }}",
				"Envvars": true,
				"LeftDelim": "{{",
				"Perms": "0644",
				"RightDelim": "}}",
				"Splay": 5000000000,
			},
		},
	}
}
