package costcenter_meta_test

import data.costcenter_meta.errors

import future.keywords

test_missing_costcenter if {
	count(errors) == 1 with input as {
		"ID": "my-job",
		"Meta": {},
	}
}

test_costcenter_prefix_wrong if {
	count(errors) == 1 with input as {
		"ID": "my-job",
		"Meta": {"costcenter": "my-costcenter"},
	}
}

test_costcenter_correct if {
	count(errors) == 0 with input as {
		"ID": "my-job",
		"Meta": {"costcenter": "cccode-my-costcenter"},
	}
}
