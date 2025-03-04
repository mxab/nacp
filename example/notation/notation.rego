package image_verification

import future.keywords

errors contains msg if {
	some g, t
	input.job.TaskGroups[g].Tasks[t].Driver == "docker"
	image := input.job.TaskGroups[g].Tasks[t].Config.image

	# check
	not notation_verify_image(image)
	msg := sprintf("TaskGroup %d Task %d image is invalid (image %s)", [g, t, image])
}
