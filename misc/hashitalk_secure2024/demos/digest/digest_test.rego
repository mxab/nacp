package digest_test

import rego.v1

import data.digest

test_has_error_when_image_is_tagged if {
	result := digest.errors with input as {"TaskGroups": [{"Tasks": [{
		"Driver": "docker",
		"Config": {"image": "alpine:3.19.1"},
	}]}]}
	result == { "Invalid image reference: alpine:3.19.1" }
}

test_has_no_errors_if_image_uses_digest if {
	result := digest.errors with input as {"TaskGroups": [{"Tasks": [{
		"Driver": "docker",
		"Config": {"image": "alpine@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b"},
	}]}]}

	result == set()
}
