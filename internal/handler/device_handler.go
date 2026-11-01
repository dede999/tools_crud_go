package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dede999/tools_crud_go/internal/domain"
	"github.com/dede999/tools_crud_go/internal/repository"
)

type DeviceHandler struct {
	repo repository.DeviceRepository
}

func NewDeviceHandler(repo repository.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

func (h *DeviceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /devices", h.Create)
	mux.HandleFunc("GET /devices/{id}", h.GetByID)
	mux.HandleFunc("GET /devices", h.List)
	mux.HandleFunc("PUT /devices/{id}", h.UpdateFull)
	mux.HandleFunc("PATCH /devices/{id}", h.UpdatePartial)
	mux.HandleFunc("DELETE /devices/{id}", h.Delete)
}

type createDeviceRequest struct {
	Name  string             `json:"name"`
	Brand string             `json:"brand"`
	State domain.DeviceState `json:"state"`
}

type updateDeviceRequest struct {
	Name  *string             `json:"name"`
	Brand *string             `json:"brand"`
	State *domain.DeviceState `json:"state"`
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	device, err := domain.NewDevice(req.Name, req.Brand, req.State)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	if err := h.repo.Create(r.Context(), device); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create device")
		return
	}

	respondJSON(w, http.StatusCreated, device)
}

func (h *DeviceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "device id is required")
		return
	}

	device, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get device")
		return
	}

	respondJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var brand *string
	if b := query.Get("brand"); b != "" {
		brand = &b
	}

	var state *domain.DeviceState
	if s := query.Get("state"); s != "" {
		st := domain.DeviceState(s)
		if !domain.IsValidState(st) {
			respondError(w, http.StatusBadRequest, "invalid state filter: "+s)
			return
		}
		state = &st
	}

	devices, err := h.repo.List(r.Context(), brand, state)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list devices")
		return
	}

	respondJSON(w, http.StatusOK, devices)
}

func (h *DeviceHandler) UpdateFull(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "device id is required")
		return
	}

	var req createDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	device, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to retrieve device")
		return
	}

	_, _, _, err = device.Update(&req.Name, &req.Brand, &req.State)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	if err := h.repo.Update(r.Context(), device); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update device")
		return
	}

	respondJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) UpdatePartial(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "device id is required")
		return
	}

	var req updateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	device, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to retrieve device")
		return
	}

	_, _, _, err = device.Update(req.Name, req.Brand, req.State)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	if err := h.repo.Update(r.Context(), device); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update device")
		return
	}

	respondJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "device id is required")
		return
	}

	device, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to retrieve device")
		return
	}

	if err := device.CanDelete(); err != nil {
		respondDomainError(w, err)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete device")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func respondDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCannotUpdateInUseDevice),
		errors.Is(err, domain.ErrCannotDeleteInUseDevice):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidName),
		errors.Is(err, domain.ErrInvalidBrand),
		errors.Is(err, domain.ErrNameTooLong),
		errors.Is(err, domain.ErrBrandTooLong),
		errors.Is(err, domain.ErrInvalidCharacters),
		errors.Is(err, domain.ErrInvalidState):
		respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrDeviceNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
