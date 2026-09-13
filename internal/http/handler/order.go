package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/middleware"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/response"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

type OrderHandler struct {
	orderSvc *service.OrderService
	menuSvc  *service.MenuService
	locSvc   *service.LocationService
}

func NewOrderHandler(os *service.OrderService, ms *service.MenuService, ls *service.LocationService) *OrderHandler {
	return &OrderHandler{orderSvc: os, menuSvc: ms, locSvc: ls}
}

func (h *OrderHandler) ValidateCart(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items      []struct { MenuItemID string `json:"menu_item_id"`; Quantity int `json:"quantity"` } `json:"items"`
		LocationID string `json:"location_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	locID, err := uuid.Parse(body.LocationID)
	if err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"location_id": "invalid UUID"}))
		return
	}
	cartItems := make([]service.CartItem, 0, len(body.Items))
	for _, it := range body.Items {
		id, err := uuid.Parse(it.MenuItemID)
		if err != nil {
			continue
		}
		cartItems = append(cartItems, service.CartItem{MenuItemID: id, Quantity: it.Quantity})
	}
	result, err := h.menuSvc.ValidateCart(r.Context(), cartItems, locID)
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not validate cart"))
		return
	}

	loc, _ := h.locSvc.GetByID(r.Context(), locID)
	if loc != nil {
		result.DeliveryFeePaise = loc.DeliveryFeePaise
		result.TotalPaise = result.SubtotalPaise + loc.DeliveryFeePaise
		result.DeliveryEnabled = loc.DeliveryEnabled
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var body struct {
		Items      []struct { MenuItemID string `json:"menu_item_id"`; Quantity int `json:"quantity"` } `json:"items"`
		LocationID  string `json:"location_id"`
		SlotID      string `json:"slot_id"`
		Notes       string `json:"notes"`
		PaymentMode string `json:"payment_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	locID, _ := uuid.Parse(body.LocationID)
	slotID, _ := uuid.Parse(body.SlotID)
	cartItems := make([]service.CartItem, 0, len(body.Items))
	for _, it := range body.Items {
		id, err := uuid.Parse(it.MenuItemID)
		if err != nil {
			continue
		}
		cartItems = append(cartItems, service.CartItem{MenuItemID: id, Quantity: it.Quantity})
	}
	o, err := h.orderSvc.Create(r.Context(), uid, service.CreateOrderInput{
		Items: cartItems, LocationID: locID, SlotID: slotID,
		Notes: body.Notes, PaymentMode: domain.PaymentCOD,
	})
	if err != nil {
		h.handleOrderError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, orderSummaryDTO(o))
}

func (h *OrderHandler) ListMyOrders(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	activeOnly := r.URL.Query().Get("status") == "active"
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	orders, total, err := h.orderSvc.ListMyOrders(r.Context(), uid, activeOnly, page, pageSize)
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load orders"))
		return
	}
	response.JSONWithMeta(w, http.StatusOK, orders, response.Meta{
		Page: page, PageSize: pageSize, Total: total,
		TotalPages: response.TotalPages(total, pageSize),
	})
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("order"))
		return
	}
	uid := middleware.GetUserID(r.Context())
	role := middleware.GetUserRole(r.Context())
	o, err := h.orderSvc.GetOrder(r.Context(), id, uid, role)
	if errors.Is(err, domain.ErrNotFound) {
		errs.WriteError(w, r, errs.NotFound("order"))
		return
	}
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load order"))
		return
	}
	response.JSON(w, http.StatusOK, orderDetailDTO(o))
}

func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("order"))
		return
	}
	uid := middleware.GetUserID(r.Context())
	if err := h.orderSvc.CancelOrder(r.Context(), id, uid); err != nil {
		h.handleOrderError(w, r, err)
		return
	}
	response.NoContent(w)
}

// Admin

func (h *OrderHandler) AdminListOrders(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	pageSize := 20
	var status *domain.OrderStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.OrderStatus(s)
		status = &st
	}
	orders, total, err := h.orderSvc.AdminList(r.Context(), status, nil, nil, page, pageSize)
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load orders"))
		return
	}
	response.JSONWithMeta(w, http.StatusOK, orders, response.Meta{
		Page: page, PageSize: pageSize, Total: total, TotalPages: response.TotalPages(total, pageSize),
	})
}

func (h *OrderHandler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("order"))
		return
	}
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	actorID := middleware.GetUserID(r.Context())
	if err := h.orderSvc.AdminUpdateStatus(r.Context(), id, domain.OrderStatus(body.Status), actorID, body.Note); err != nil {
		h.handleOrderError(w, r, err)
		return
	}
	response.NoContent(w)
}

func (h *OrderHandler) AdminCancelOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("order"))
		return
	}
	actorID := middleware.GetUserID(r.Context())
	if err := h.orderSvc.AdminUpdateStatus(r.Context(), id, domain.StatusCancelledAdmin, actorID, "Cancelled by admin"); err != nil {
		h.handleOrderError(w, r, err)
		return
	}
	response.NoContent(w)
}

func (h *OrderHandler) handleOrderError(w http.ResponseWriter, r *http.Request, err error) {
	if items, ok := service.IsItemsUnavailable(err); ok {
		errs.WriteError(w, r, errs.ItemsUnavailable(items))
		return
	}
	switch {
	case errors.Is(err, domain.ErrSlotFull):
		errs.WriteError(w, r, errs.SlotFull())
	case errors.Is(err, domain.ErrSlotExpired):
		errs.WriteError(w, r, errs.SlotExpired())
	case errors.Is(err, domain.ErrDeliveryDisabled):
		errs.WriteError(w, r, errs.DeliveryDisabled())
	case errors.Is(err, domain.ErrInvalidTransition):
		errs.WriteError(w, r, errs.InvalidTransition())
	case errors.Is(err, domain.ErrCancelWindowPassed):
		errs.WriteError(w, r, errs.CancelWindowPassed())
	case errors.Is(err, domain.ErrNotFound):
		errs.WriteError(w, r, errs.NotFound("order"))
	default:
		errs.WriteError(w, r, errs.Internal("an error occurred"))
	}
}

func orderSummaryDTO(o *domain.Order) map[string]any {
	return map[string]any{
		"id": o.ID, "short_code": o.ShortCode, "status": o.Status,
		"total_paise": o.TotalPaise, "cancel_deadline_at": o.CancelDeadlineAt,
	}
}

func orderDetailDTO(o *domain.Order) map[string]any {
	dto := map[string]any{
		"id": o.ID, "short_code": o.ShortCode, "status": o.Status,
		"subtotal_paise": o.SubtotalPaise, "delivery_fee_paise": o.DeliveryFeePaise, "total_paise": o.TotalPaise,
		"payment_mode": o.PaymentMode, "payment_status": o.PaymentStatus,
		"notes": o.Notes, "can_cancel": o.CanCancel(), "can_review": o.CanReview(),
		"cancel_deadline_at": o.CancelDeadlineAt, "placed_at": o.PlacedAt,
		"delivered_at": o.DeliveredAt, "cancelled_at": o.CancelledAt,
		"items": o.Items,
	}
	if o.Location != nil {
		dto["location"] = map[string]any{"code": o.Location.Code, "name": o.Location.Name}
	}
	if o.Slot != nil {
		dto["slot"] = map[string]any{
			"start_time": o.Slot.StartTime, "end_time": o.Slot.EndTime,
			"slot_date": o.Slot.SlotDate.Format("2006-01-02"),
		}
	}
	events := make([]map[string]any, 0, len(o.Events))
	for _, e := range o.Events {
		events = append(events, map[string]any{"status": e.ToStatus, "at": e.CreatedAt})
	}
	dto["timeline"] = events
	return dto
}
