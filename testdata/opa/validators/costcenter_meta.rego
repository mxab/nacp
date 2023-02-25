
package costcenter_meta


import future.keywords.contains
import future.keywords.if

# This definition checks if the costcenter label is not provided. Each rule definition
# contributes to the set of error messages.
errors contains msg if {
	# The `not` keyword turns an undefined statement into a true statement. If any
	# of the keys are missing, this statement will be true.
    
    
	not input.Meta.costcenter
    trace("Costcenter code is missing")

	msg := "Every job must have a costcenter metadata label"
}

# This definition checks if the costcenter label is formatted appropriately. Each rule
# definition contributes to the set of error messages.
errors contains msg if {
	value := input.Meta.costcenter
    
	not startswith(value, "cccode-")
	msg := sprintf("Costcenter code must start with `cccode-`; found `%v`", [value])
}
