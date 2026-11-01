package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DeviceState string

const (
	StateAvailable DeviceState = "available"
	StateInUse     DeviceState = "in-use"
	StateInactive  DeviceState = "inactive"
)

var (
	ErrInvalidName              = errors.New("device name cannot be empty")
	ErrInvalidBrand             = errors.New("device brand cannot be empty")
	ErrInvalidState             = errors.New("invalid device state, must be one of: available, in-use, inactive")
	ErrCannotUpdateInUseDevice  = errors.New("name and brand cannot be updated if device is in use")
	ErrCannotDeleteInUseDevice  = errors.New("in use devices cannot be deleted")
	ErrDeviceNotFound           = errors.New("device not found")
)

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
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrInvalidName
	}

	trimmedBrand := strings.TrimSpace(brand)
	if trimmedBrand == "" {
		return nil, ErrInvalidBrand
	}

	if state == "" {
		state = StateAvailable
	}

	if !IsValidState(state) {
		return nil, ErrInvalidState
	}

	return &Device{
		ID:        uuid.NewString(),
		Name:      trimmedName,
		Brand:     trimmedBrand,
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

	if name != nil {
		trimmedName := strings.TrimSpace(*name)
		if trimmedName == "" {
			return false, d.State, d.State, ErrInvalidName
		}
		d.Name = trimmedName
	}

	if brand != nil {
		trimmedBrand := strings.TrimSpace(*brand)
		if trimmedBrand == "" {
			return false, d.State, d.State, ErrInvalidBrand
		}
		d.Brand = trimmedBrand
	}

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
