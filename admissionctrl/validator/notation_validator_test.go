package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockVerifier struct {
	mock.Mock
}

func (m *MockVerifier) VerifyImage(ctx context.Context, imageReference string) error {
	args := m.Called(ctx, imageReference)
	return args.Error(0)
}

func TestNotationValidatorValidate(t *testing.T) {

	tt := []struct {
		name          string
		image         string
		verifierError error
	}{
		{
			name:  "valid image",
			image: "myimage",
		},
		{
			name:          "invalid image",
			image:         "myimage",
			verifierError: errors.New("invalid image"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mockImageVerifier := new(MockVerifier)

			mockImageVerifier.On("VerifyImage", mock.Anything, tc.image).Return(tc.verifierError)

			notationValidator := &NotationValidator{
				logger:   nil,
				name:     "notation",
				verifier: mockImageVerifier,
			}
			errors, err := notationValidator.Validate(&api.Job{
				TaskGroups: []*api.TaskGroup{
					{
						Tasks: []*api.Task{
							{

								Config: map[string]interface{}{
									"image": tc.image,
								},
							},
						},
					},
				},
			})
			require.NoError(t, err)

			if tc.verifierError == nil {
				require.Empty(t, errors)
			} else {
				require.Equal(t, []error{
					tc.verifierError,
				}, errors)
			}
		})
	}

}

func TestNewNotationValidator(t *testing.T) {
	mockImageVerifier := new(MockVerifier)

	notationValidator := NewNotationValidator(hclog.NewNullLogger(), "notation", mockImageVerifier)
	require.Equal(t, mockImageVerifier, notationValidator.verifier)
	require.Equal(t, "notation", notationValidator.name)
	require.NotNil(t, notationValidator.logger)
}
func TestNotationValidatorName(t *testing.T) {
	mockImageVerifier := new(MockVerifier)

	notationValidator := &NotationValidator{
		logger:   nil,
		name:     "notation",
		verifier: mockImageVerifier,
	}
	require.Equal(t, "notation", notationValidator.Name())
}
