package postgres
import future.keywords

jobName := input.Name

patch contains operation if {
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

patch contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Vault/Policies/-", [g, t]),
		"value": sprintf("%s-db-access", [jobName])
	}
}

patch contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres

	object.get(input.TaskGroups[g].Tasks[t], "Templates", null) == null

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Templates", [g, t]),
		"value": [],
	}
}

patch contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.postgres


	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/Templates/-", [g, t]),
		"value": {
			"ChangeMode": "restart",
			"DestPath": "${NOMAD_SECRETS_DIR}/postgres.env",
			"EmbeddedTmpl": env_tmpl,
			"Envvars": true,
			"LeftDelim": "{{",
			"Perms": "0644",
			"RightDelim": "}}",
			"Splay": 5000000000
		},
	}
}
env_tmpl := libpq_tmpl if {
	input.TaskGroups[g].Tasks[t].Meta.postgres == "libpq"

    db_name := replace(jobName, "-", "_")
	libpq_tmpl:= concat("\n", [
        sprintf("PGDATABASE=%s",[db_name]),
		sprintf("{{ with secret \"db/%s/creds/admin\" }}", [db_name]),
		"PGUSER={{ .Data.username }}",
		"PGPASSWORD={{ .Data.password }}",
		"{{ end }}",
        "{{ range nomadService \"postgres\" }}",
		"PGHOSTADDR={{ .Address }}",
		"PGPORT={{ .Port }}",
		"{{ end }}",
	])

}
env_tmpl := spring_boot_tmpl if {
	input.TaskGroups[g].Tasks[t].Meta.postgres == "springboot"

    db_name := replace(jobName, "-", "_")

	spring_boot_tmpl:= concat("\n", [
		"{{ range nomadService \"postgres\" }}",
		sprintf("SPRING_DATASOURCE_URL=jdbc:postgresql://{{ .Address }}:{{ .Port }}/%s",[db_name]),
		"{{ end }}",
		sprintf("{{ with secret \"db/%s/creds/admin\" }}", [db_name]),
		"SPRING_DATASOURCE_USERNAME={{ .Data.username }}",
		"SPRING_DATASOURCE_PASSWORD={{ .Data.password }}",
		"{{ end }}",
	])

}
