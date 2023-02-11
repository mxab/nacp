package nacp.vault_policy_prefix

import future.keywords.every
import future.keywords.if
import future.keywords.contains

default allow :=false
policies contains name if {
    name := input.TaskGroups[_].Tasks[_].Vault.Policies[_]
}
policy_prefix := concat("-", [input.ID, ""])
allow if {

    every p in policies {
        startswith(p, policy_prefix)
    }
}
