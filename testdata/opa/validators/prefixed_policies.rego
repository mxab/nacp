package prefixed_policies

import future.keywords
import future.keywords.in

task_group_policies contains name if {
	name := input.job.TaskGroups[_].Vault.Policies[_]
}

task_policies contains name if {
	name := input.job.TaskGroups[_].Tasks[_].Vault.Policies[_]
}
policy_prefix := sprintf("%s-", [input.job.ID])

errors[msg] {

	some p in task_policies
	not startswith(p, policy_prefix)
	msg := sprintf("Task policy '%v' must start with '%v'", [p, policy_prefix])
}
errors[msg] {

	some p in task_group_policies
	not startswith(p, policy_prefix)
	msg := sprintf("Task group policy '%v' must start with '%v'", [p, policy_prefix])
}