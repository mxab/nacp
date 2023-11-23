package department

import future.keywords

test_has_error_if_no_owner_is_set if {
	result := errors with input as {}
	result == {"No owner is set"}
}

test_has_error_if_not_a_valid_department if {
	result := errors with input as {"Meta": {"owner": "housekeeping"}}
	result == {"housekeeping is not a valid owner"}
}
