package image_verification

import future.keywords

errors contains msg if {
	some g, t
	input.TaskGroups[g].Tasks[t].Driver == "docker"
	image := input.TaskGroups[g].Tasks[t].Config.image
	not valid_notation_image(image)
	msg := sprintf("TaskGroup %d Task %d image is invalid (image %s)", [g, t, image])
}
