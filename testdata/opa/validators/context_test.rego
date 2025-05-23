package context_test

import data.context

# Test that a blocked IP produces the expected error.
test_ip_blocked {
    input := {
        "context": {
            "clientIP": "192.168.1.10",
            "tokenInfo": {
                "Policies": []  # No policies, so only the IP error should fire.
            }
        }
    }
    expected := {"IP address 192.168.1.10 is in blocklist"}
    context.errors with input as input == expected
}

# Test that an allowed IP produces no error.
test_ip_allowed {
    input := {
        "context": {
            "clientIP": "192.168.1.11",  # Not in the blocklist.
            "tokenInfo": {
                "Policies": []
            }
        }
    }
    context.errors with input as input == {}
}

# Test that a restricted policy ("nomad_reject") produces an error.
test_policy_reject {
    input := {
        "context": {
            "clientIP": "192.168.1.11",  # Use an allowed IP.
            "tokenInfo": {
                "Policies": ["nomad_reject"]
            }
        }
    }
    expected := {"Policy nomad_reject is not allowed"}
    context.errors with input as input == expected
}

# Test that a non-restricted policy does not produce an error.
test_policy_allowed {
    input := {
        "context": {
            "clientIP": "192.168.1.11",
            "tokenInfo": {
                "Policies": ["allowed_policy"]
            }
        }
    }
    context.errors with input as input == {}
}

# Test that a policy triggering a warning ("nomad_warn") produces the expected debug warning.
test_debug_warning {
    input := {
        "context": {
            "clientIP": "192.168.1.11",
            "tokenInfo": {
                "Policies": ["nomad_warn"],
                "detail": "test detail"
            }
        }
    }
    expected_warning := sprintf("Debug: TokenInfo: %v", [input.context.tokenInfo])
    context.warnings with input as input == {expected_warning}
}

# Test that a token without a warn policy produces no warnings.
test_no_debug_warning {
    input := {
        "context": {
            "clientIP": "192.168.1.11",
            "tokenInfo": {
                "Policies": ["some_policy"]
            }
        }
    }
    context.warnings with input as input == {}
}
