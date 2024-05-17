package digest

import rego.v1


errors contains err_msg if {
    some g, t
	input.TaskGroups[g].Tasks[t].Driver == "docker"
	image := input.TaskGroups[g].Tasks[t].Config.image

    not regex.match( "@sha256:[a-f0-9]{64}$", image)

    err_msg := sprintf("Invalid image reference: %v", [image])
}
