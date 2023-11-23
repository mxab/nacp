package logging
import future.keywords


patch contains operation if {

	input.TaskGroups[g].Tasks[t].Meta.logging

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/%d/leader", [g, t]),
		"value": true,
	}
}

patch contains operation if {
	input.TaskGroups[g].Tasks[t].Meta.logging

	operation := {
		"op": "add",
		"path": sprintf("/TaskGroups/%d/Tasks/-", [g]),
		"value": {
          "Name": "grafanaagent",
          "Driver": "docker",
          "User": "",
          "Lifecycle": {
            "Hook": "prestart",
            "Sidecar": true
          },
          "Config": {
            "image": "grafana-agent-sidecar:v1"
          },
          "Templates": [
            {
              "Envvars": true,
              "DestPath": "secrets/grafanaagent.env",
              "EmbeddedTmpl": concat("\n",[
                "{{ with nomadVar \"grafana\" }}",
                "LOKI_URL={{ .loki_endpoint }}",
                "LOKI_USERNAME={{ .loki_username }}",
                "LOKI_API_TOKEN={{ .loki_api_token }}",
                "{{ end }}",
                sprintf("TASK_TO_LOG=%s", [input.TaskGroups[g].Tasks[t].Name]),
              ]),
              "ChangeMode": "restart",
              "Splay": 5000000000,
              "Perms": "0644",
              "ErrMissingKey": false
            }
          ]
        }
	}
}
