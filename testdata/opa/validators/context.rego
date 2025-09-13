package context

import future.keywords.contains
import future.keywords.if

# IP validation
errors contains msg if {
     blocked_ips := {"192.168.1.10"}
     ip := input.context.clientIP
     blocked_ips[ip]
     msg := sprintf("IP address %v is in blocklist", [ip])
}

# Policy validation
errors contains msg if {
    restricted_policies := {"nomad_reject"}
    policy = input.context.tokenInfo.Policies[_]
    restricted_policies[policy]
    msg := sprintf("Policy %v is not allowed", [policy])
}

# Debug warnings
warnings contains msg if {
    warn_policy := {"nomad_warn"}
    policy = input.context.tokenInfo.Policies[_]
    warn_policy[policy]
    msg := sprintf("Debug: TokenInfo: %v", [input.context.tokenInfo])
}
