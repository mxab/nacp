package opatest

errors contains errMsg if {
	errMsg := "This is a error message"
}

warnings contains warnMsg if {
	warnMsg := "This is a warning message"
}
patch contains op if {
	op := {"op": "add", "path": "/Meta", "value": {"foo": "bar"}}
}
