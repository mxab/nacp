package image_verification_test

import future.keywords

import data.image_verification

mock_notation_verify_image("validimage:latest") := true

mock_notation_verify_image("invalidimage:latest") := false

test_has_no_errors_for_valid_image if {
	result := image_verification.errors with input as {"TaskGroups": [{"Tasks": [{
		"Driver": "docker",
		"Config": {"image": "validimage:latest"},
	}]}]}
		with notation_verify_image as mock_notation_verify_image

	result == set()
}

test_has_errors_for_invalid_image if {
	result := image_verification.errors with input as {"TaskGroups": [{"Tasks": [{
		"Driver": "docker",
		"Config": {"image": "invalidimage:latest"},
	}]}]}
		with notation_verify_image as mock_notation_verify_image

	result == {"TaskGroup 0 Task 0 image is invalid (image invalidimage:latest)"}
}
