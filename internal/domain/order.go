package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusPlaced           OrderStatus = "PLACED"
	StatusAccepted         OrderStatus = "ACCEPTED"
	StatusPreparing        OrderStatus = "PREPARING"
	StatusReady            OrderStatus = "READY"
	StatusOutForDelivery   OrderStatus = "OUT_FOR_DELIVERY"
	StatusDelivered        OrderStatus = "DELIVERED"
	StatusCancelledUser    OrderStatus = "CANCELLED_BY_USER"
	StatusCancelledAdmin   OrderStatus = "CANCELLED_BY_ADMIN"
)

func (s OrderStatus) IsTerminal() bool {
	return s == StatusDelivered || s == StatusCancelledUser || s == StatusCancelledAdmin
}

func (s OrderStatus) IsCancelled() bool {
	return s == StatusCancelledUser || s == StatusCancelledAdmin
}

// AllowedTransitions defines the valid next states from a given state.
var AllowedTransitions = map[OrderStatus][]OrderStatus{
	StatusPlaced:         {StatusAccepted, StatusCancelledUser, StatusCancelledAdmin},
	StatusAccepted:       {StatusPreparing, StatusCancelledAdmin},
	StatusPreparing:      {StatusReady},
	StatusReady:          {StatusOutForDelivery},
	StatusOutForDelivery: {StatusDelivered},
}

func (s OrderStatus) CanTransitionTo(next OrderStatus) bool {
	allowed, ok := AllowedTransitions[s]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == next {
			return true
		}
	}
	return false
}

type PaymentMode string

const (
	PaymentCOD    PaymentMode = "COD"
	PaymentOnline PaymentMode = "ONLINE"
)

type PaymentStatus string

const (
	PayPending  PaymentStatus = "PENDING"
	PayPaid     PaymentStatus = "PAID"
	PayFailed   PaymentStatus = "FAILED"
	PayRefunded PaymentStatus = "REFUNDED"
)

type Order struct {
	ID               uuid.UUID          `json:"id"`
	ShortCode        string             `json:"short_code"`
	UserID           uuid.UUID          `json:"user_id"`
	LocationID       uuid.UUID          `json:"location_id"`
	SlotID           uuid.UUID          `json:"slot_id"`
	Status           OrderStatus        `json:"status"`
	PaymentMode      PaymentMode        `json:"payment_mode"`
	PaymentStatus    PaymentStatus      `json:"payment_status"`
	SubtotalPaise    int64              `json:"subtotal_paise"`
	DeliveryFeePaise int64              `json:"delivery_fee_paise"`
	TotalPaise       int64              `json:"total_paise"`
	Notes            string             `json:"notes"`
	CancelDeadlineAt time.Time          `json:"cancel_deadline_at"`
	PlacedAt         time.Time          `json:"placed_at"`
	DeliveredAt      *time.Time         `json:"delivered_at"`
	CancelledAt      *time.Time         `json:"cancelled_at"`
	CancelReason     *string            `json:"cancel_reason"`
	UpdatedAt        time.Time          `json:"updated_at"`
	Items            []OrderItem        `json:"items"`
	Events           []OrderStatusEvent `json:"timeline"`
	Location         *Location          `json:"location"`
	Slot             *DeliverySlot      `json:"slot"`
}

// MarshalJSON includes the computed CanCancel/CanReview flags so every
// endpoint that returns an Order (list or single) carries them consistently.
func (o *Order) MarshalJSON() ([]byte, error) {
	type Alias Order
	return json.Marshal(&struct {
		*Alias
		CanCancel bool `json:"can_cancel"`
		CanReview bool `json:"can_review"`
	}{
		Alias:     (*Alias)(o),
		CanCancel: o.CanCancel(),
		CanReview: o.CanReview(),
	})
}

func (o *Order) CanCancel() bool {
	return o.Status == StatusPlaced && time.Now().Before(o.CancelDeadlineAt)
}

func (o *Order) CanReview() bool {
	return o.Status == StatusDelivered
}

type OrderItem struct {
	ID                 uuid.UUID `json:"id"`
	OrderID            uuid.UUID `json:"order_id"`
	MenuItemID         uuid.UUID `json:"menu_item_id"`
	NameSnapshot       string    `json:"name_snapshot"`
	PriceSnapshotPaise int64     `json:"price_snapshot_paise"`
	Quantity           int       `json:"quantity"`
	LineTotalPaise     int64     `json:"line_total_paise"`
}

type OrderStatusEvent struct {
	ID         uuid.UUID    `json:"id"`
	OrderID    uuid.UUID    `json:"order_id"`
	FromStatus *OrderStatus `json:"from_status"`
	ToStatus   OrderStatus  `json:"status"`
	ActorID    *uuid.UUID   `json:"actor_id"`
	Note       string       `json:"note"`
	CreatedAt  time.Time    `json:"at"`
}

type Location struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	DeliveryEnabled  bool      `json:"delivery_enabled"`
	DeliveryFeePaise int64     `json:"delivery_fee_paise"`
	SortOrder        int       `json:"sort_order"`
	CreatedAt        time.Time `json:"created_at"`
}

type DeliverySlot struct {
	ID            uuid.UUID `json:"id"`
	LocationID    uuid.UUID `json:"location_id"`
	SlotDate      time.Time `json:"slot_date"`
	StartTime     string    `json:"start_time"` // "13:00"
	EndTime       string    `json:"end_time"`   // "13:30"
	Capacity      int       `json:"capacity"`
	BookedCount   int       `json:"booked_count"`
	CutoffMinutes int       `json:"cutoff_minutes"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *DeliverySlot) SeatsLeft() int {
	return s.Capacity - s.BookedCount
}

// MarshalJSON includes the computed SeatsLeft so any endpoint returning a
// raw DeliverySlot (e.g. admin slot listings) carries it consistently with
// the customer-facing slot DTO.
func (s *DeliverySlot) MarshalJSON() ([]byte, error) {
	type Alias DeliverySlot
	return json.Marshal(&struct {
		*Alias
		SeatsLeft int `json:"seats_left"`
	}{
		Alias:     (*Alias)(s),
		SeatsLeft: s.SeatsLeft(),
	})
}

func (s *DeliverySlot) CutoffAt() time.Time {
	// Combine slot_date + start_time, subtract cutoff_minutes
	// StartTime is "HH:MM"
	var h, m int
	_, _ = parseTime(s.StartTime, &h, &m)
	slotStart := time.Date(
		s.SlotDate.Year(), s.SlotDate.Month(), s.SlotDate.Day(),
		h, m, 0, 0, time.UTC,
	)
	return slotStart.Add(-time.Duration(s.CutoffMinutes) * time.Minute)
}

func parseTime(t string, h, m *int) (int, int) {
	if len(t) < 5 {
		return 0, 0
	}
	*h = int(t[0]-'0')*10 + int(t[1]-'0')
	*m = int(t[3]-'0')*10 + int(t[4]-'0')
	return *h, *m
}
