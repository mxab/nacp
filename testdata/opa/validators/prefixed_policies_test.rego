package prefixed_policies_test

import data.prefixed_policies.errors

import future.keywords

test_no_errors if {
    count(errors) == 0 with input as {
		"ID": "example",
		"Name": "example",
		"TaskGroups": [{
			"Name": "cache",
			"Tasks": [],
		}],
		"Type": "service",
	}

}

test_no_errors_for_valid_policy if {
	count(errors) == 0 with input as {
		"ID": "example",
		"Name": "example",
		"TaskGroups": [{
			"Name": "cache",
			"Tasks": [{"Vault": {"Policies": ["example-redis"]}}],
		}],
		"Type": "service",
	}
}
test_no_errors_for_multi_valid_policy  if {
    count(errors) == 0 with input as {
		"ID": "example",
		"Name": "example",
		"TaskGroups": [{
			"Name": "cache",
			"Tasks": [{"Vault": {"Policies": [
				"example-redis",
				"example-mysql",
			]}}],
		}],
		"Type": "service",
	}
}

test_errors_for_wrong_task_policy if {
	count(errors) == 1 with input as {
		"ID": "example",
		"Name": "example",
		"TaskGroups": [{
			"Name": "cache",
			"Tasks": [{"Vault": {"Policies": ["some-randome-policy"]}}],
		}],
		"Type": "service",
	}

}
test_errors_for_multi_wrong_policy {

	count(errors) == 2 with input as {
		"ID": "example",
		"Name": "example",
		"TaskGroups": [{
			"Name": "cache",
			"Tasks": [{"Vault": {"Policies": ["some-randome-policy","also-not-valid"]}}],
		}],
		"Type": "service",
	}
    

}

test_errors_for_wrong_task_group_policy if {
	count(errors) == 1 with input as {
		"ID": "example",
		"Name": "example",
		"TaskGroups": [{
            "Vault": {"Policies": ["some-randome-policy"]},
			"Name": "cache",
			"Tasks": [],
		}],
		"Type": "service",
	}

}