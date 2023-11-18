package department
import future.keywords

errors contains "No owner is set" if {
	not input.Meta.owner
}

valid_owners := {"sales", "development" , "finance", "hr", "legal", "operations"}

errors contains sprintf("%s is not a valid owner", [input.Meta.owner]) if {

	input.Meta.owner
    not input.Meta.owner in valid_owners

}
