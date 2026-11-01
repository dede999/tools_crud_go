package domain_test

import (
	"testing"
	"time"

	"github.com/dede999/tools_crud_go/internal/domain"
)

func TestNewDevice(t *testing.T) {
	t.Run("successfully create valid device", func(t *testing.T) {
		dev, err := domain.NewDevice("MacBook Pro", "Apple", domain.StateAvailable)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dev.ID == "" {
			t.Errorf("expected generated ID, got empty")
		}
		if dev.Name != "MacBook Pro" {
			t.Errorf("expected name 'MacBook Pro', got %q", dev.Name)
		}
		if dev.Brand != "Apple" {
			t.Errorf("expected brand 'Apple', got %q", dev.Brand)
		}
		if dev.State != domain.StateAvailable {
			t.Errorf("expected state 'available', got %q", dev.State)
		}
		if dev.CreatedAt.IsZero() {
			t.Errorf("expected non-zero creation time")
		}
	})

	t.Run("default to available if state is empty", func(t *testing.T) {
		dev, err := domain.NewDevice("Pixel 8", "Google", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dev.State != domain.StateAvailable {
			t.Errorf("expected default state 'available', got %q", dev.State)
		}
	})

	t.Run("fail when name is empty or whitespace", func(t *testing.T) {
		_, err := domain.NewDevice("   ", "Apple", domain.StateAvailable)
		if err != domain.ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})

	t.Run("fail when brand is empty or whitespace", func(t *testing.T) {
		_, err := domain.NewDevice("MacBook Pro", "  ", domain.StateAvailable)
		if err != domain.ErrInvalidBrand {
			t.Errorf("expected ErrInvalidBrand, got %v", err)
		}
	})

	t.Run("fail when state is invalid", func(t *testing.T) {
		_, err := domain.NewDevice("MacBook Pro", "Apple", domain.DeviceState("broken"))
		if err != domain.ErrInvalidState {
			t.Errorf("expected ErrInvalidState, got %v", err)
		}
	})
}

func TestDevice_Update(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	t.Run("allow updating name, brand, state when device is available", func(t *testing.T) {
		dev := domain.Device{
			ID:        "test-id",
			Name:      "Old Name",
			Brand:     "Old Brand",
			State:     domain.StateAvailable,
			CreatedAt: fixedTime,
		}

		newName := "New Name"
		newBrand := "New Brand"
		newState := domain.StateInUse

		changed, prev, next, err := dev.Update(&newName, &newBrand, &newState)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !changed || prev != domain.StateAvailable || next != domain.StateInUse {
			t.Errorf("state transition mismatch: changed=%v prev=%v next=%v", changed, prev, next)
		}
		if dev.Name != newName || dev.Brand != newBrand || dev.State != newState {
			t.Errorf("device values not updated properly")
		}
		if !dev.CreatedAt.Equal(fixedTime) {
			t.Errorf("creation time must never be updated")
		}
	})

	t.Run("block updating name or brand when device is in-use", func(t *testing.T) {
		dev := domain.Device{
			ID:        "test-id",
			Name:      "Active Device",
			Brand:     "Apple",
			State:     domain.StateInUse,
			CreatedAt: fixedTime,
		}

		newName := "Brand New Name"
		_, _, _, err := dev.Update(&newName, nil, nil)
		if err != domain.ErrCannotUpdateInUseDevice {
			t.Errorf("expected ErrCannotUpdateInUseDevice when updating name, got %v", err)
		}

		newBrand := "Dell"
		_, _, _, err = dev.Update(nil, &newBrand, nil)
		if err != domain.ErrCannotUpdateInUseDevice {
			t.Errorf("expected ErrCannotUpdateInUseDevice when updating brand, got %v", err)
		}
	})

	t.Run("allow changing state from in-use to inactive or available", func(t *testing.T) {
		dev := domain.Device{
			ID:        "test-id",
			Name:      "Active Device",
			Brand:     "Apple",
			State:     domain.StateInUse,
			CreatedAt: fixedTime,
		}

		newState := domain.StateAvailable
		changed, prev, next, err := dev.Update(nil, nil, &newState)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !changed || prev != domain.StateInUse || next != domain.StateAvailable {
			t.Errorf("state transition mismatch: changed=%v prev=%v next=%v", changed, prev, next)
		}
		if dev.State != domain.StateAvailable {
			t.Errorf("expected state to be available, got %q", dev.State)
		}
	})

	t.Run("fail if updating to invalid state", func(t *testing.T) {
		dev := domain.Device{
			ID:        "test-id",
			Name:      "Device",
			Brand:     "Apple",
			State:     domain.StateAvailable,
			CreatedAt: fixedTime,
		}

		invalidState := domain.DeviceState("unknown")
		_, _, _, err := dev.Update(nil, nil, &invalidState)
		if err != domain.ErrInvalidState {
			t.Errorf("expected ErrInvalidState, got %v", err)
		}
	})
}

func TestDevice_CanDelete(t *testing.T) {
	t.Run("allow delete when available or inactive", func(t *testing.T) {
		availableDev := domain.Device{State: domain.StateAvailable}
		if err := availableDev.CanDelete(); err != nil {
			t.Errorf("expected available device to be deletable, got %v", err)
		}

		inactiveDev := domain.Device{State: domain.StateInactive}
		if err := inactiveDev.CanDelete(); err != nil {
			t.Errorf("expected inactive device to be deletable, got %v", err)
		}
	})

	t.Run("prevent delete when in-use", func(t *testing.T) {
		inUseDev := domain.Device{State: domain.StateInUse}
		if err := inUseDev.CanDelete(); err != domain.ErrCannotDeleteInUseDevice {
			t.Errorf("expected ErrCannotDeleteInUseDevice, got %v", err)
		}
	})
}
