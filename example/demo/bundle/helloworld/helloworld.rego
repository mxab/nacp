package helloworld

errors contains msg if {
	not input.job.Meta.hello
	trace("Hello label is missing")
	msg := "Every job must have a `hello` label"
}

errors contains msg if {
	value := input.job.Meta.hello

	value != "world"
	msg := sprintf("Hello label must be `world`; found `%v`", [value])
}
