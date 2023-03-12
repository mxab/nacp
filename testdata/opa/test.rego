package opatest

errors[errMsg] {
	errMsg := "This is a error message"
}

warnings[warnMsg] {
	warnMsg := "This is a warning message"
}
patch[op] {
	op := {"op": "add", "path": "/Meta", "value": {"foo": "bar"}}
}
