package opa

import (
	"errors"
	"fmt"
)

// Decision is the document every NACP policy is expected to produce, no matter
// whether it is evaluated as an embedded rego file or as a rule inside a remote
// bundle: optional errors, optional warnings, and for mutators a JSON Patch.
type Decision struct {
	Errors   []error
	Warnings []error
	Patch    []interface{}
	// HasPatch distinguishes a policy that deliberately returned no patch from
	// one that returned an empty list of operations.
	HasPatch bool
}

// ParseMode selects how strictly a decision document is interpreted.
type ParseMode int

const (
	// Strict rejects anything that does not match the documented decision
	// shape. Bundle adapters use it so that a mistyped decision path, or a
	// policy that yields a scalar instead of an object, fails the admission
	// instead of silently letting the job through.
	Strict ParseMode = iota
	// Lenient ignores values that do not match the documented shape. The
	// embedded adapters use it to keep the behaviour they shipped with, where a
	// missing or wrongly typed binding is treated as "nothing to report".
	Lenient
)

// ParseDecision converts the raw result of an OPA evaluation into a Decision.
func ParseDecision(raw interface{}, mode ParseMode) (*Decision, error) {
	decision := &Decision{Patch: []interface{}{}}

	document, err := decisionDocument(raw, mode)
	if err != nil {
		return nil, err
	}
	if document == nil {
		return decision, nil
	}

	decision.Errors, err = parseMessages(document["errors"], "error", mode)
	if err != nil {
		return nil, err
	}
	decision.Warnings, err = parseMessages(document["warnings"], "warning", mode)
	if err != nil {
		return nil, err
	}
	decision.Patch, decision.HasPatch, err = parsePatch(document["patch"], mode)
	if err != nil {
		return nil, err
	}

	return decision, nil
}

func decisionDocument(raw interface{}, mode ParseMode) (map[string]interface{}, error) {
	if document, ok := raw.(map[string]interface{}); ok {
		return document, nil
	}
	if mode == Lenient {
		return nil, nil
	}
	if raw == nil {
		return nil, errors.New("policy decision is undefined")
	}
	return nil, fmt.Errorf("policy yielded an invalid decision value: %v", raw)
}

// parseMessages reads an errors or warnings list. noun is the singular form
// used in error messages, so the plural collection reads "errors value" while a
// bad element reads "error entry value".
func parseMessages(raw interface{}, noun string, mode ParseMode) ([]error, error) {
	if raw == nil {
		return nil, nil
	}

	entries, ok := raw.([]interface{})
	if !ok {
		if mode == Lenient {
			return nil, nil
		}
		return nil, fmt.Errorf("policy yielded an invalid %ss value: %v", noun, raw)
	}

	messages := make([]error, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		message, ok := entry.(string)
		if !ok {
			if mode == Lenient {
				// The embedded adapters have always rendered whatever the policy
				// produced, so keep doing that rather than dropping the entry.
				messages = append(messages, fmt.Errorf("%v", entry))
				continue
			}
			return nil, fmt.Errorf("policy yielded an invalid %s entry value: %v", noun, entry)
		}
		messages = append(messages, errors.New(message))
	}
	return messages, nil
}

func parsePatch(raw interface{}, mode ParseMode) ([]interface{}, bool, error) {
	if raw == nil {
		return []interface{}{}, false, nil
	}

	operations, ok := raw.([]interface{})
	if !ok {
		if mode == Lenient {
			return []interface{}{}, false, nil
		}
		return nil, false, fmt.Errorf("policy yielded an invalid patch value: %v", raw)
	}

	if mode == Strict {
		for _, operation := range operations {
			if _, ok := operation.(map[string]interface{}); !ok {
				return nil, false, fmt.Errorf("policy yielded an invalid patch entry value: %v", operation)
			}
		}
	}

	return operations, true, nil
}
