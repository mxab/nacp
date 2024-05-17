package image_verification

import rego.v1

errors contains msg if {
	some g, t
	input.TaskGroups[g].Tasks[t].Driver == "docker"
	image := input.TaskGroups[g].Tasks[t].Config.image

	# check if image is verified
	not notation_verify_image(image)

	msg := sprintf("TaskGroup %d Task %d image cannot be verified (image %s)", [g, t, image])
}
