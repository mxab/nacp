package validator

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/mxab/nacp/admissionctrl/types"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/require"
)

type DummyVerifier struct {
}

func (m *DummyVerifier) VerifyImage(ctx context.Context, imageReference string) error {

	if imageReference == "invalidimage:latest" {
		return errors.New("invalid image")
	}

	return nil
}

func TestNotationValidatorValidate(t *testing.T) {

	tt := []struct {
		name string

		expectedErrors []error

		tasks []struct {
			driver string
			image  string
		}
	}{
		{
			name: "valid image",
			tasks: []struct {
				driver string
				image  string
			}{

				{
					driver: "docker",
					image:  "validimage:latest",
				},
			},
			expectedErrors: nil,
		},
		{
			name: "invalid image",
			tasks: []struct {
				driver string
				image  string
			}{

				{
					driver: "docker",
					image:  "invalidimage:latest",
				},
			},
			expectedErrors: []error{
				errors.New("invalid image"),
			},
		},
		{
			name: "invalid image in second task",
			tasks: []struct {
				driver string
				image  string
			}{
				{
					driver: "docker",
					image:  "validimage:latest",
				},
				{
					driver: "docker",
					image:  "invalidimage:latest",
				},
			},
			expectedErrors: []error{
				errors.New("invalid image"),
			},
		},
		{
			name: "non docker task",
			tasks: []struct {
				driver string
				image  string
			}{
				{
					driver: "magic",
					image:  "invalidimage:latest",
				},
			},
			expectedErrors: nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mockImageVerifier := new(DummyVerifier)

			notationValidator := &NotationValidator{
				logger:   nil,
				name:     "notation",
				verifier: mockImageVerifier,
			}

			groups := []*api.TaskGroup{}
			for _, task := range tc.tasks {
				groups = append(groups, &api.TaskGroup{
					Tasks: []*api.Task{
						{
							Driver: task.driver,
							Config: map[string]interface{}{
								"image": task.image,
							},
						},
					},
				})
			}

			payload := &types.Payload{
				Job: &api.Job{
					TaskGroups: groups,
				},
			}

			errors, err := notationValidator.Validate(t.Context(), payload)
			require.Equal(t, tc.expectedErrors, errors)
			require.NoError(t, err)

		})
	}

}

func TestNewNotationValidator(t *testing.T) {
	mockImageVerifier := new(DummyVerifier)

	notationValidator := NewNotationValidator(slog.New(slog.DiscardHandler), "notation", mockImageVerifier)
	require.Equal(t, mockImageVerifier, notationValidator.verifier)
	require.Equal(t, "notation", notationValidator.name)
	require.NotNil(t, notationValidator.logger)
}
func TestNotationValidatorName(t *testing.T) {
	mockImageVerifier := new(DummyVerifier)

	notationValidator := &NotationValidator{
		logger:   nil,
		name:     "notation",
		verifier: mockImageVerifier,
	}
	require.Equal(t, "notation", notationValidator.Name())
}
