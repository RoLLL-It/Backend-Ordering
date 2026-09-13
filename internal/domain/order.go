package domain

import (
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
	ID                uuid.UUID
	ShortCode         string
	UserID            uuid.UUID
	LocationID        uuid.UUID
	SlotID            uuid.UUID
	Status            OrderStatus
	PaymentMode       PaymentMode
	PaymentStatus     PaymentStatus
	SubtotalPaise     int64
	DeliveryFeePaise  int64
	TotalPaise        int64
	Notes             string
	CancelDeadlineAt  time.Time
	PlacedAt          time.Time
	DeliveredAt       *time.Time
	CancelledAt       *time.Time
	CancelReason      *string
	UpdatedAt         time.Time
	Items             []OrderItem
	Events            []OrderStatusEvent
	Location          *Location
	Slot              *DeliverySlot
}

func (o *Order) CanCancel() bool {
	return o.Status == StatusPlaced && time.Now().Before(o.CancelDeadlineAt)
}

func (o *Order) CanReview() bool {
	return o.Status == StatusDelivered
}

type OrderItem struct {
	ID                  uuid.UUID
	OrderID             uuid.UUID
	MenuItemID          uuid.UUID
	NameSnapshot        string
	PriceSnapshotPaise  int64
	Quantity            int
	LineTotalPaise      int64
}

type OrderStatusEvent struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	FromStatus *OrderStatus
	ToStatus   OrderStatus
	ActorID    *uuid.UUID
	Note       string
	CreatedAt  time.Time
}

type Location struct {
	ID              uuid.UUID
	Code            string
	Name            string
	DeliveryEnabled bool
	DeliveryFeePaise int64
	SortOrder       int
	CreatedAt       time.Time
}

type DeliverySlot struct {
	ID            uuid.UUID
	LocationID    uuid.UUID
	SlotDate      time.Time
	StartTime     string // "13:00"
	EndTime       string // "13:30"
	Capacity      int
	BookedCount   int
	CutoffMinutes int
	IsActive      bool
	CreatedAt     time.Time
}

func (s *DeliverySlot) SeatsLeft() int {
	return s.Capacity - s.BookedCount
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
