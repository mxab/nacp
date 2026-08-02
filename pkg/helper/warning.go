// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Vendored verbatim from github.com/hashicorp/nomad at v1.6.3, helper/warning.go.
//
// Nomad switched to BUSL-1.1 at v1.6.6. v1.6.5 and earlier shipped under MPL-2.0,
// but v1.6.6's license names "Nomad Version 1.6.4 or later" as the Licensed Work
// (v1.7.0 restates this as "1.7.0 or later"), so v1.6.3 was chosen because it
// predates that boundary on either reading and is unambiguously MPL-2.0 -- the
// same license as NACP. This file is identical at v1.6.3 and v1.6.5.
//
// The code is unchanged from that release; only this note was added, so NACP
// keeps producing byte-identical warning strings without depending on a
// BUSL-1.1 module.

package helper

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// MergeMultierrorWarnings takes warnings and merges them into a returnable
// string. This method is used to return API and CLI errors.
func MergeMultierrorWarnings(errs ...error) string {
	if len(errs) == 0 {
		return ""
	}

	var mErr multierror.Error
	_ = multierror.Append(&mErr, errs...)
	mErr.ErrorFormat = warningsFormatter

	return mErr.Error()
}

// warningsFormatter is used to format warnings.
func warningsFormatter(es []error) string {
	sb := strings.Builder{}

	switch len(es) {
	case 0:
		return ""
	case 1:
		sb.WriteString("1 warning:\n")
	default:
		sb.WriteString(fmt.Sprintf("%d warnings:\n", len(es)))
	}

	for _, err := range es {
		sb.WriteString(fmt.Sprintf("\n* %s", strings.TrimSpace(err.Error())))
	}

	return sb.String()
}
