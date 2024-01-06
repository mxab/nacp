package opatest

errors[errMsg] {

	image := input.TaskGroups[0].Tasks[0].Config.image

	not validNotationImage(image)
	errMsg := "Image is not in valid"
}
