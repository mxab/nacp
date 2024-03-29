package owner

import future.keywords

# "input" is a nomad job

errors contains "Whoopsi, it looks like you forgot to set a team email for emergency contacts. Please put 'owner=\"<yourteam>@example.com\" in the meta block of the job.'" if {
	not input.Meta.owner
}

warnings contains sprintf("Hey, can you please update the %s owner to a @example.com email", [input.Meta.owner]) if {
	input.Meta.owner
	endswith(input.Meta.owner, "@example.org")
}

errors contains sprintf("Oh no, '%s' is not a valid email address, please use something @example.com", [input.Meta.owner]) if {
	input.Meta.owner
	not valid_email
}

valid_email if endswith(input.Meta.owner, "@example.org")

valid_email if endswith(input.Meta.owner, "@example.com")
