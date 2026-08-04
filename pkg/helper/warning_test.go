package helper

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
)

// These are characterization tests: they pin the observable output of
// MergeMultierrorWarnings rather than how it is implemented, so the function can
// be reworked (for example onto the standard library's errors package) while
// proving the rendered warnings are unchanged.
//
// The output is user-visible -- it is what NACP returns in the warning fields of
// Nomad's job register, plan and validate responses -- so any diff here is a
// change in the API contract, not an implementation detail.

func TestMergeMultierrorWarnings(t *testing.T) {
	tests := []struct {
		name string
		errs []error
		want string
	}{
		{
			name: "no warnings yields empty string",
			errs: nil,
			want: "",
		},
		{
			name: "only nil warnings yields empty string",
			errs: []error{nil, nil},
			want: "",
		},
		{
			name: "single warning is singular in the header",
			errs: []error{errors.New("some warning")},
			want: "1 warning:\n\n* some warning",
		},
		{
			name: "multiple warnings are counted and bulleted",
			errs: []error{errors.New("first"), errors.New("second")},
			want: "2 warnings:\n\n* first\n* second",
		},
		{
			name: "nil entries are skipped and do not affect the count",
			errs: []error{nil, errors.New("only one"), nil},
			want: "1 warning:\n\n* only one",
		},
		{
			name: "surrounding whitespace is trimmed from each entry",
			errs: []error{errors.New("  padded  \n")},
			want: "1 warning:\n\n* padded",
		},
		{
			// The production caller passes a single aggregate holding every
			// warning, so without this expansion every response would collapse
			// to "1 warning". Any reimplementation must preserve it.
			name: "a multierror aggregate is expanded into individual entries",
			errs: []error{multierror.Append(nil, errors.New("first"), errors.New("second"))},
			want: "2 warnings:\n\n* first\n* second",
		},
		{
			name: "an empty aggregate yields empty string",
			errs: []error{&multierror.Error{}},
			want: "",
		},
		{
			name: "a typed-nil aggregate contributes nothing and does not panic",
			errs: []error{(*multierror.Error)(nil), errors.New("only one")},
			want: "1 warning:\n\n* only one",
		},
		{
			// Expansion is one level deep only: a nested aggregate is rendered
			// with its own formatting rather than being flattened further.
			name: "nested aggregates are expanded one level only",
			errs: []error{&multierror.Error{Errors: []error{
				multierror.Append(nil, errors.New("inner")),
				errors.New("outer"),
			}}},
			want: "2 warnings:\n\n* 1 error occurred:\n\t* inner\n* outer",
		},
		{
			// Mirrors the real proxy path: Nomad's own warnings arrive as an
			// already-formatted string that gets wrapped in a single error, so it
			// stays one entry while the admission warning becomes another. The
			// same string is asserted end-to-end in cmd/nacp's TestProxy.
			name: "an upstream warning string stays a single entry",
			errs: []error{
				fmt.Errorf("%s", multierror.Append(nil, errors.New("some warning")).Error()),
				errors.New("some warning"),
			},
			want: "2 warnings:\n\n* 1 error occurred:\n\t* some warning\n* some warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MergeMultierrorWarnings(tt.errs...))
		})
	}
}

// TestMergeMultierrorWarningsDoesNotExpandErrorsJoin records a known limitation
// rather than desired behaviour.
//
// Expansion is driven by go-multierror, which only recognises its own
// *multierror.Error. An error built with the standard library's errors.Join is
// therefore treated as one warning, and its children run together separated by
// the newline that errors.Join inserts.
//
// Nothing in NACP builds warnings with errors.Join today, so this is latent. If
// the function is reworked to also expand errors.Join -- which go-multierror's
// own README nudges towards -- this test is expected to change, and that change
// should be called out as a deliberate one.
func TestMergeMultierrorWarningsDoesNotExpandErrorsJoin(t *testing.T) {
	joined := errors.Join(errors.New("first"), errors.New("second"))

	assert.Equal(t, "1 warning:\n\n* first\nsecond", MergeMultierrorWarnings(joined))
}
