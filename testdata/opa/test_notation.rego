package opatest

errors[errMsg] {

	image := input.job.TaskGroups[0].Tasks[0].Config.image

	not notation_verify_image(image)
	errMsg := "Image is not in valid"
}
