package system

main := {
	"errors" :errors,
	"warnings" : warnings
}
errors contains errMsg if {
	errMsg := "This is a error message"
}

warnings contains warnMsg if {
	warnMsg := "This is a warning message"
}
