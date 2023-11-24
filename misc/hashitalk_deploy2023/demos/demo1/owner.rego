package department
import future.keywords

errors contains "Whoopsi, it looks like you forgot to set a team email for emergency contacts. Please put 'owner=\"<yourteam>@example.com\" in the meta block of the job.'" if {
	not input.Meta.owner
}

errors contains sprintf("Oh no, '%s' is not a valid email address, please use something @example.com", [input.Meta.owner]) if {

	input.Meta.owner
	not_valid_email
}

warnings contains sprintf("Hey, can you please update the %s owner to a @example.com email", [input.Meta.owner]) if {

	input.Meta.owner
	endswith(input.Meta.owner, "@example.org")
	

}

not_valid_email if not endswith(input.Meta.owner, "@example.com")
not_valid_email if not endswith(input.Meta.owner, "@example.org")