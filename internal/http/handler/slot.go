package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/response"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

type SlotHandler struct {
	slotRepo *repo.SlotRepo
	locSvc   *service.LocationService
	settings *service.SettingsService
}

func NewSlotHandler(sr *repo.SlotRepo, ls *service.LocationService, ss *service.SettingsService) *SlotHandler {
	return &SlotHandler{slotRepo: sr, locSvc: ls, settings: ss}
}

func (h *SlotHandler) GetLocations(w http.ResponseWriter, r *http.Request) {
	locs, err := h.locSvc.List(r.Context())
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load locations"))
		return
	}
	response.JSON(w, http.StatusOK, locs)
}

func (h *SlotHandler) ListSlots(w http.ResponseWriter, r *http.Request) {
	locIDStr := r.URL.Query().Get("location_id")
	locID, err := uuid.Parse(locIDStr)
	if err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"location_id": "required"}))
		return
	}
	dateStr := r.URL.Query().Get("date")
	var date time.Time
	if dateStr == "" {
		date = time.Now().UTC().Truncate(24 * time.Hour)
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			errs.WriteError(w, r, errs.Validation(map[string]string{"date": "invalid format, use YYYY-MM-DD"}))
			return
		}
	}

	slots, err := h.slotRepo.ListByLocationAndDate(r.Context(), locID, date)
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load slots"))
		return
	}

	loc, _ := h.locSvc.GetByID(r.Context(), locID)
	deliveryEnabled, _ := h.settings.GetBool(r.Context(), "delivery_enabled")

	dtos := make([]map[string]any, 0, len(slots))
	for _, s := range slots {
		dto := slotDTO(s, loc, deliveryEnabled)
		dtos = append(dtos, dto)
	}
	response.JSON(w, http.StatusOK, dtos)
}

func slotDTO(s *domain.DeliverySlot, loc *domain.Location, globalDelivery bool) map[string]any {
	available := true
	var reason *string
	if !globalDelivery || (loc != nil && !loc.DeliveryEnabled) {
		available = false
		r := "DELIVERY_DISABLED"
		reason = &r
	} else if s.BookedCount >= s.Capacity {
		available = false
		r := "FULL"
		reason = &r
	} else if time.Now().After(s.CutoffAt()) {
		available = false
		r := "CUTOFF_PASSED"
		reason = &r
	}
	return map[string]any{
		"id": s.ID, "start_time": s.StartTime, "end_time": s.EndTime,
		"capacity": s.Capacity, "booked_count": s.BookedCount, "seats_left": s.SeatsLeft(),
		"is_available": available, "unavailable_reason": reason,
	}
}

// Admin slot handlers

func (h *SlotHandler) AdminListSlots(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	date := time.Now().UTC().Truncate(24 * time.Hour)
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			errs.WriteError(w, r, errs.Validation(map[string]string{"date": "invalid"}))
			return
		}
	}
	slots, err := h.slotRepo.ListForAdmin(r.Context(), date)
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load slots"))
		return
	}
	response.JSON(w, http.StatusOK, slots)
}

func (h *SlotHandler) AdminCreateSlot(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LocationID    string `json:"location_id"`
		SlotDate      string `json:"slot_date"`
		StartTime     string `json:"start_time"`
		EndTime       string `json:"end_time"`
		Capacity      int    `json:"capacity"`
		CutoffMinutes int    `json:"cutoff_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	locID, _ := uuid.Parse(body.LocationID)
	date, err := time.Parse("2006-01-02", body.SlotDate)
	if err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"slot_date": "invalid"}))
		return
	}
	s := &domain.DeliverySlot{
		ID: uuid.New(), LocationID: locID, SlotDate: date,
		StartTime: body.StartTime, EndTime: body.EndTime,
		Capacity: body.Capacity, CutoffMinutes: body.CutoffMinutes, IsActive: true,
	}
	if err := h.slotRepo.Create(r.Context(), s); err != nil {
		errs.WriteError(w, r, errs.Internal("could not create slot"))
		return
	}
	response.JSON(w, http.StatusCreated, s)
}

func (h *SlotHandler) AdminUpdateSlot(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("slot"))
		return
	}
	var body struct {
		Capacity      int  `json:"capacity"`
		CutoffMinutes int  `json:"cutoff_minutes"`
		IsActive      bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	s := &domain.DeliverySlot{ID: id, Capacity: body.Capacity, CutoffMinutes: body.CutoffMinutes, IsActive: body.IsActive}
	if err := h.slotRepo.Update(r.Context(), s); err != nil {
		errs.WriteError(w, r, errs.Internal("could not update slot"))
		return
	}
	response.JSON(w, http.StatusOK, s)
}

func (h *SlotHandler) AdminUpdateLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("location"))
		return
	}
	var body struct {
		DeliveryEnabled bool `json:"delivery_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	if err := h.locSvc.SetDelivery(r.Context(), id, body.DeliveryEnabled); err != nil {
		errs.WriteError(w, r, errs.Internal("could not update location"))
		return
	}
	response.NoContent(w)
}

func (h *SlotHandler) AdminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	// Accept mixed JSON value types (bool, string, number) since settings are
	// a heterogeneous key/value bag (delivery_enabled: bool, announcement: string, etc).
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	for k, v := range body {
		var strVal string
		switch val := v.(type) {
		case bool:
			strVal = strconv.FormatBool(val)
		case string:
			strVal = val
		case float64:
			strVal = strconv.FormatFloat(val, 'f', -1, 64)
		default:
			errs.WriteError(w, r, errs.Validation(map[string]string{k: "unsupported value type"}))
			return
		}
		if err := h.settings.Set(r.Context(), k, strVal); err != nil {
			errs.WriteError(w, r, errs.Internal("could not update settings"))
			return
		}
	}
	response.NoContent(w)
}
