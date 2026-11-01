package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dede999/tools_crud_go/internal/domain"
	"github.com/dede999/tools_crud_go/internal/handler"
	"github.com/dede999/tools_crud_go/internal/notifier"
	"github.com/dede999/tools_crud_go/internal/repository"
)

// fakeRepository implements repository.DeviceRepository in-memory for lightning-fast handler testing
type fakeRepository struct {
	mu      sync.RWMutex
	devices map[string]domain.Device
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		devices: make(map[string]domain.Device),
	}
}

func (f *fakeRepository) Create(ctx context.Context, d *domain.Device) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.devices[d.ID] = *d
	return nil
}

func (f *fakeRepository) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	d, ok := f.devices[id]
	if !ok {
		return nil, domain.ErrDeviceNotFound
	}
	return &d, nil
}

func (f *fakeRepository) List(ctx context.Context, brand *string, state *domain.DeviceState) ([]domain.Device, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var list []domain.Device
	for _, d := range f.devices {
		if brand != nil && *brand != "" && !strings.EqualFold(d.Brand, *brand) {
			continue
		}
		if state != nil && *state != "" && d.State != *state {
			continue
		}
		list = append(list, d)
	}
	return list, nil
}

func (f *fakeRepository) Update(ctx context.Context, d *domain.Device) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.devices[d.ID]; !ok {
		return domain.ErrDeviceNotFound
	}
	f.devices[d.ID] = *d
	return nil
}

func (f *fakeRepository) Delete(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.devices[id]; !ok {
		return domain.ErrDeviceNotFound
	}
	delete(f.devices, id)
	return nil
}

var _ repository.DeviceRepository = (*fakeRepository)(nil)

func setupTestServer() (http.Handler, *fakeRepository, *notifier.EventBroker) {
	repo := newFakeRepository()
	broker := notifier.NewBroker()
	h := handler.NewDeviceHandler(repo, broker)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux, repo, broker
}

func TestDeviceHandler_Create(t *testing.T) {
	mux, _, _ := setupTestServer()

	t.Run("successfully create device", func(t *testing.T) {
		payload := `{"name":"MacBook Pro","brand":"Apple","state":"available"}`
		req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var created domain.Device
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if created.ID == "" || created.Name != "MacBook Pro" || created.Brand != "Apple" {
			t.Errorf("unexpected device created: %+v", created)
		}
	})

	t.Run("reject empty name", func(t *testing.T) {
		payload := `{"name":"","brand":"Apple","state":"available"}`
		req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("reject invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader("{invalid-json"))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestDeviceHandler_GetByID(t *testing.T) {
	mux, repo, _ := setupTestServer()

	dev, _ := domain.NewDevice("Pixel 9", "Google", domain.StateAvailable)
	_ = repo.Create(context.Background(), dev)

	t.Run("get existing device", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/devices/"+dev.ID, nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("return 404 for non-existing device", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/devices/non-existent-id", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})
}

func TestDeviceHandler_List(t *testing.T) {
	mux, repo, _ := setupTestServer()

	d1, _ := domain.NewDevice("iPhone 15", "Apple", domain.StateAvailable)
	d2, _ := domain.NewDevice("Galaxy S24", "Samsung", domain.StateInUse)
	_ = repo.Create(context.Background(), d1)
	_ = repo.Create(context.Background(), d2)

	t.Run("list all devices", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/devices", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var items []domain.Device
		_ = json.Unmarshal(rec.Body.Bytes(), &items)
		if len(items) != 2 {
			t.Errorf("expected 2 devices, got %d", len(items))
		}
	})

	t.Run("filter by brand", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/devices?brand=Apple", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		var items []domain.Device
		_ = json.Unmarshal(rec.Body.Bytes(), &items)
		if len(items) != 1 || items[0].Brand != "Apple" {
			t.Errorf("expected 1 Apple device, got %+v", items)
		}
	})

	t.Run("filter by state", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/devices?state=in-use", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		var items []domain.Device
		_ = json.Unmarshal(rec.Body.Bytes(), &items)
		if len(items) != 1 || items[0].State != domain.StateInUse {
			t.Errorf("expected 1 in-use device, got %+v", items)
		}
	})
}

func TestDeviceHandler_Update(t *testing.T) {
	mux, repo, broker := setupTestServer()

	d1, _ := domain.NewDevice("iPad Air", "Apple", domain.StateAvailable)
	d2, _ := domain.NewDevice("ThinkPad X1", "Lenovo", domain.StateInUse)
	_ = repo.Create(context.Background(), d1)
	_ = repo.Create(context.Background(), d2)

	// Subscribe to verify state change notifications
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eventsCh, unsub := broker.Subscribe(ctx)
	defer unsub()

	t.Run("full update (PUT) available device and receive notification", func(t *testing.T) {
		payload := `{"name":"iPad Pro","brand":"Apple","state":"inactive"}`
		req := httptest.NewRequest(http.MethodPut, "/devices/"+d1.ID, bytes.NewBufferString(payload))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		select {
		case ev := <-eventsCh:
			if ev.DeviceID != d1.ID || ev.PreviousState != domain.StateAvailable || ev.NewState != domain.StateInactive {
				t.Errorf("unexpected event received: %+v", ev)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("expected notification event on state change")
		}
	})

	t.Run("partial update (PATCH) updating only state", func(t *testing.T) {
		payload := `{"state":"available"}`
		req := httptest.NewRequest(http.MethodPatch, "/devices/"+d2.ID, bytes.NewBufferString(payload))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		select {
		case ev := <-eventsCh:
			if ev.DeviceID != d2.ID || ev.PreviousState != domain.StateInUse || ev.NewState != domain.StateAvailable {
				t.Errorf("unexpected event received: %+v", ev)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("expected notification event on state change")
		}
	})

	t.Run("reject updating name or brand on in-use device (409 Conflict)", func(t *testing.T) {
		// Put d2 back to in-use
		inUseState := domain.StateInUse
		_, _, _, _ = d2.Update(nil, nil, &inUseState)
		_ = repo.Update(context.Background(), d2)

		payload := `{"name":"ThinkPad X2"}`
		req := httptest.NewRequest(http.MethodPatch, "/devices/"+d2.ID, bytes.NewBufferString(payload))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict when modifying in-use device name, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestDeviceHandler_Delete(t *testing.T) {
	mux, repo, _ := setupTestServer()

	d1, _ := domain.NewDevice("Mouse", "Logitech", domain.StateAvailable)
	d2, _ := domain.NewDevice("Server Rack", "Dell", domain.StateInUse)
	_ = repo.Create(context.Background(), d1)
	_ = repo.Create(context.Background(), d2)

	t.Run("delete available device returns 204 No Content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/devices/"+d1.ID, nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d", rec.Code)
		}
	})

	t.Run("reject delete on in-use device (409 Conflict)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/devices/"+d2.ID, nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict when deleting in-use device, got %d", rec.Code)
		}
	})
}

