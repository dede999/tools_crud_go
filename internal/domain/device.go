package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type DeviceState string

const (
	StateAvailable DeviceState = "available"
	StateInUse     DeviceState = "in-use"
	StateInactive  DeviceState = "inactive"

	MaxNameLength  = 255
	MaxBrandLength = 255
)

var (
	ErrInvalidName             = errors.New("device name cannot be empty")
	ErrNameTooLong             = errors.New("device name exceeds maximum length of 255 characters")
	ErrInvalidBrand            = errors.New("device brand cannot be empty")
	ErrBrandTooLong            = errors.New("device brand exceeds maximum length of 255 characters")
	ErrInvalidCharacters       = errors.New("input contains invalid or unprintable control characters")
	ErrInvalidState            = errors.New("invalid device state, must be one of: available, in-use, inactive")
	ErrCannotUpdateInUseDevice = errors.New("name and brand cannot be updated if device is in use")
	ErrCannotDeleteInUseDevice = errors.New("in use devices cannot be deleted")
	ErrDeviceNotFound          = errors.New("device not found")
)

func validateField(val string, maxLen int, emptyErr, tooLongErr error) (string, error) {
	if !utf8.ValidString(val) {
		return "", ErrInvalidCharacters
	}

	for _, r := range val {
		// Reject null byte and non-printable control characters
		if r < 32 {
			return "", ErrInvalidCharacters
		}
	}

	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return "", emptyErr
	}

	if utf8.RuneCountInString(trimmed) > maxLen {
		return "", tooLongErr
	}

	return trimmed, nil
}

func IsValidState(s DeviceState) bool {
	switch s {
	case StateAvailable, StateInUse, StateInactive:
		return true
	default:
		return false
	}
}

type Device struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Brand     string      `json:"brand"`
	State     DeviceState `json:"state"`
	CreatedAt time.Time   `json:"created_at"`
}

func NewDevice(name, brand string, state DeviceState) (*Device, error) {
	validatedName, err := validateField(name, MaxNameLength, ErrInvalidName, ErrNameTooLong)
	if err != nil {
		return nil, err
	}

	validatedBrand, err := validateField(brand, MaxBrandLength, ErrInvalidBrand, ErrBrandTooLong)
	if err != nil {
		return nil, err
	}

	if state == "" {
		state = StateAvailable
	}

	if !IsValidState(state) {
		return nil, ErrInvalidState
	}

	return &Device{
		ID:        uuid.NewString(),
		Name:      validatedName,
		Brand:     validatedBrand,
		State:     state,
		CreatedAt: time.Now().UTC(),
	}, nil
}

type UpdateParams struct {
	Name  *string
	Brand *string
	State *DeviceState
}

type UpdateResult struct {
	StateChanged  bool
	PreviousState DeviceState
	NewState      DeviceState
}

// Update handles full and partial updates adhering to all domain invariants.
// Returns an UpdateResult describing any state transition and a validation error if invariants fail.
func (d *Device) Update(params UpdateParams) (*UpdateResult, error) {
	// Domain rule: Name and brand properties cannot be updated if the device is currently in-use.
	// Only state transitions (e.g. releasing the device back to 'available' or 'inactive') are permitted.
	if d.State == StateInUse {
		if (params.Name != nil && strings.TrimSpace(*params.Name) != d.Name) ||
			(params.Brand != nil && strings.TrimSpace(*params.Brand) != d.Brand) {
			return nil, ErrCannotUpdateInUseDevice
		}
	} else {
		// When device is not in-use, validate and apply name and brand updates
		if params.Name != nil {
			vName, err := validateField(*params.Name, MaxNameLength, ErrInvalidName, ErrNameTooLong)
			if err != nil {
				return nil, err
			}
			d.Name = vName
		}

		if params.Brand != nil {
			vBrand, err := validateField(*params.Brand, MaxBrandLength, ErrInvalidBrand, ErrBrandTooLong)
			if err != nil {
				return nil, err
			}
			d.Brand = vBrand
		}
	}

	stateChanged := false
	prevState := d.State
	nextState := d.State

	if params.State != nil {
		if !IsValidState(*params.State) {
			return nil, ErrInvalidState
		}
		if d.State != *params.State {
			stateChanged = true
			prevState = d.State
			nextState = *params.State
			d.State = *params.State
		}
	}

	return &UpdateResult{
		StateChanged:  stateChanged,
		PreviousState: prevState,
		NewState:      nextState,
	}, nil
}

// ValidateDelete checks whether the device can be deleted according to business rules.
func (d *Device) ValidateDelete() error {
	if d.State == StateInUse {
		return ErrCannotDeleteInUseDevice
	}
	return nil
}
