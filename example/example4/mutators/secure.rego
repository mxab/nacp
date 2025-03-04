package secure

import future.keywords

patch contains op if {
	input.job.TaskGroups[g].Meta.secure

	serviceToSecure := input.job.TaskGroups[g].Meta.secure

	renamedService := sprintf("internal-%s", [serviceToSecure])

	tmpl := concat("\n", [
		"provider = \"keycloak-oidc\"",
		sprintf("client_id = \"%s\"", [serviceToSecure]),
		"client_secret = \"secret\"",
		sprintf("redirect_url = \"http://%s.nomad.local/oauth2/callback\"", [serviceToSecure]),
		"oidc_issuer_url = \"http://keycloak.nomad.local/realms/demo\"",
		"email_domains = [\"*\"]",
		"cookie_secret=\"TEktZxl3wbO9cL3mkm-DyMRvjhhJqxf7Xk8fcZQFq-U=\"",
		"http_address=\"http://0.0.0.0:4180\"",
		"oidc_extra_audiences=[\"account\"]",
		"cookie_secure=false",
		sprintf("{{ range nomadService \"%s\" }}", [renamedService]),
		"upstreams = [",
		"  \"http://{{ .Address }}:{{ .Port }}\"",
		"]",
		"{{ end }}",
	])

	op := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/1", [g]),
		"value": {
			"Affinities": null,
			"Artifacts": null,
			"CSIPluginConfig": null,
			"Config": {
				"args": [
					"--config",
					"/local/config.cfg",
				],
				"image": "bitnami/oauth2-proxy:7.4.0",
				"ports": ["auth"],
			},
			"Constraints": null,
			"DispatchPayload": null,
			"Driver": "docker",
			"Env": null,
			"Identity": null,
			"KillSignal": "",
			"KillTimeout": 5000000000,
			"Kind": "",
			"Leader": false,
			"Lifecycle": null,
			"LogConfig": {
				"MaxFileSizeMB": 10,
				"MaxFiles": 10,
			},
			"Meta": null,
			"Name": "auth",
			"Resources": {
				"CPU": 100,
				"Cores": 0,
				"Devices": null,
				"DiskMB": 0,
				"IOPS": 0,
				"MemoryMB": 300,
				"MemoryMaxMB": 0,
				"Networks": null,
			},
			"RestartPolicy": {
				"Attempts": 2,
				"Delay": 15000000000,
				"Interval": 1800000000000,
				"Mode": "fail",
			},
			"ScalingPolicies": null,
			"Services": null,
			"ShutdownDelay": 0,
			"Templates": [{
				"ChangeMode": "restart",
				"ChangeScript": null,
				"ChangeSignal": "",
				"DestPath": "${NOMAD_TASK_DIR}/config.cfg",
				"EmbeddedTmpl": tmpl,
				"Envvars": false,
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
			}],
			"User": "",
			"Vault": null,
			"VolumeMounts": null,
		},
	}
}

patch contains op if {
	input.job.TaskGroups[g].Meta.secure
	serviceToSecure := input.job.TaskGroups[g].Meta.secure
	renamedService := sprintf("internal-%s", [serviceToSecure])
	input.job.TaskGroups[g].Services[s].Name == serviceToSecure

	op := {
		"op": "replace",
		"path": sprintf("/TaskGroups/%d/Services/%d/Name", [g, s]),
		"value": renamedService,
	}
}

patch contains op if {
	input.job.TaskGroups[g].Meta.secure
	serviceToSecure := input.job.TaskGroups[g].Meta.secure
	renamedService := sprintf("internal-%s", [serviceToSecure])
	input.job.TaskGroups[g].Services[s].Name == serviceToSecure

	op := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Services/%d", [g, s + 1]),
		"value": {
			"Address": "",
			"AddressMode": "auto",
			"CanaryMeta": null,
			"CanaryTags": null,
			"Checks": null,
			"Connect": null,
			"EnableTagOverride": false,
			"Meta": null,
			"Name": serviceToSecure,
			"Namespace": "default",
			"OnUpdate": "require_healthy",
			"PortLabel": "auth",
			"Provider": "nomad",
			"TaggedAddresses": null,
			"Tags": null,
			"TaskName": "",
		},
	}
}

patch contains op if {
	input.job.TaskGroups[g].Meta.secure

	op := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Networks/0/DynamicPorts/1", [g]),
		"value": {
			"HostNetwork": "default",
			"Label": "auth",
			"To": 4180,
			"Value": 0,
		},
	}
}
