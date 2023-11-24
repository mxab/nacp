package department

import future.keywords

test_has_error_if_no_owner_is_set if {
	result := errors with input as {}
	result == {"Whoopsi, it looks like you forgot to set a team email for emergency contacts. Please put 'owner=\"<yourteam>@example.com\" in the meta block of the job.'"}
}

test_has_error_if_not_a_valid_email if {
	result := errors with input as {"Meta": {"owner": "something@localhost"}}
	
	result == {"Oh no, 'something@localhost' is not a valid email address, please use something @example.com"}
}

test_has_warning_for_example_org_email if {
	result := warnings with input as {"Meta": {"owner": "foo@example.org"}}
	
	result == {"Hey, can you please update the foo@example.org owner to a @example.com email"}
}
