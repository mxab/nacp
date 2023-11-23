package logging

import future.keywords

test_patch if {

	actual_patch := patch with input as {
		"ID": "my-app",
		"Name": "my-app",
		"Type": "service",
		"TaskGroups": [{
			"Name": "app",
			"Tasks": [{
				"Name": "main",
				"Meta": {"logging": "true"},
			}]
		}]
	}



    count(actual_patch) == 2

}
