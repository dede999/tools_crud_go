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

// Update handles full and partial updates adhering to all domain invariants.
// Returns whether the state changed, previous state, new state, and any validation error.
func (d *Device) Update(name *string, brand *string, state *DeviceState) (bool, DeviceState, DeviceState, error) {
	if d.State == StateInUse {
		if (name != nil && strings.TrimSpace(*name) != d.Name) ||
			(brand != nil && strings.TrimSpace(*brand) != d.Brand) {
			return false, d.State, d.State, ErrCannotUpdateInUseDevice
		}
	}

	var newName = d.Name
	if name != nil {
		vName, err := validateField(*name, MaxNameLength, ErrInvalidName, ErrNameTooLong)
		if err != nil {
			return false, d.State, d.State, err
		}
		newName = vName
	}

	var newBrand = d.Brand
	if brand != nil {
		vBrand, err := validateField(*brand, MaxBrandLength, ErrInvalidBrand, ErrBrandTooLong)
		if err != nil {
			return false, d.State, d.State, err
		}
		newBrand = vBrand
	}

	d.Name = newName
	d.Brand = newBrand

	stateChanged := false
	prevState := d.State
	nextState := d.State

	if state != nil {
		if !IsValidState(*state) {
			return false, d.State, d.State, ErrInvalidState
		}
		if d.State != *state {
			stateChanged = true
			prevState = d.State
			nextState = *state
			d.State = *state
		}
	}

	return stateChanged, prevState, nextState, nil
}

// CanDelete checks whether the device can be deleted according to business rules.
func (d *Device) CanDelete() error {
	if d.State == StateInUse {
		return ErrCannotDeleteInUseDevice
	}
	return nil
}
