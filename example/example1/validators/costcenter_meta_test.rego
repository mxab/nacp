package costcenter_meta_test
import data.costcenter_meta.errors

import future.keywords

test_missing_costcenter if {
	errs := errors with input as {
		"ID": "my-job",
		"Meta": {},
	}

	errs["Every job must have a costcenter metadata label"]

	count(errs) == 1

}

test_costcenter_prefix_wrong if {
	errs := errors with input as {
		"ID": "my-job",
		"Meta": {"costcenter": "my-costcenter"},
	}
	errs["Costcenter code must start with `cccode-`; found `my-costcenter`"]
	count(errs) == 1
}

test_costcenter_correct if {
	errs := errors with input as {
		"ID": "my-job",
		"Meta": {"costcenter": "cccode-my-costcenter"},
	}
	count(errs) == 0

}
