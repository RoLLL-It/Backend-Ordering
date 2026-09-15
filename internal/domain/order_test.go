package domain

import (
	"testing"
	"time"
)

func TestOrderCanCancel(t *testing.T) {
	t.Run("placed and within deadline", func(t *testing.T) {
		o := &Order{
			Status:           StatusPlaced,
			CancelDeadlineAt: time.Now().Add(1 * time.Minute),
		}
		if !o.CanCancel() {
			t.Error("expected CanCancel=true")
		}
	})

	t.Run("placed but deadline passed", func(t *testing.T) {
		o := &Order{
			Status:           StatusPlaced,
			CancelDeadlineAt: time.Now().Add(-1 * time.Minute),
		}
		if o.CanCancel() {
			t.Error("expected CanCancel=false when deadline passed")
		}
	})

	t.Run("wrong status", func(t *testing.T) {
		o := &Order{
			Status:           StatusAccepted,
			CancelDeadlineAt: time.Now().Add(1 * time.Minute),
		}
		if o.CanCancel() {
			t.Error("expected CanCancel=false for non-PLACED status")
		}
	})
}

func TestOrderCanReview(t *testing.T) {
	statuses := []struct {
		status   OrderStatus
		expected bool
	}{
		{StatusDelivered, true},
		{StatusPlaced, false},
		{StatusAccepted, false},
		{StatusPreparing, false},
		{StatusReady, false},
		{StatusOutForDelivery, false},
		{StatusCancelledUser, false},
		{StatusCancelledAdmin, false},
	}
	for _, tc := range statuses {
		o := &Order{Status: tc.status}
		if got := o.CanReview(); got != tc.expected {
			t.Errorf("CanReview() for %s: got %v, want %v", tc.status, got, tc.expected)
		}
	}
}

func TestCanTransitionTo(t *testing.T) {
	valid := []struct{ from, to OrderStatus }{
		{StatusPlaced, StatusAccepted},
		{StatusPlaced, StatusCancelledUser},
		{StatusPlaced, StatusCancelledAdmin},
		{StatusAccepted, StatusPreparing},
		{StatusAccepted, StatusCancelledAdmin},
		{StatusPreparing, StatusReady},
		{StatusReady, StatusOutForDelivery},
		{StatusOutForDelivery, StatusDelivered},
	}
	for _, tc := range valid {
		if !tc.from.CanTransitionTo(tc.to) {
			t.Errorf("expected valid transition %s -> %s", tc.from, tc.to)
		}
	}

	invalid := []struct{ from, to OrderStatus }{
		{StatusPlaced, StatusPreparing},
		{StatusPlaced, StatusDelivered},
		{StatusAccepted, StatusDelivered},
		{StatusPreparing, StatusAccepted},
		{StatusDelivered, StatusPlaced},
		{StatusCancelledUser, StatusPlaced},
		{StatusCancelledAdmin, StatusPlaced},
	}
	for _, tc := range invalid {
		if tc.from.CanTransitionTo(tc.to) {
			t.Errorf("expected invalid transition %s -> %s", tc.from, tc.to)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	terminal := []OrderStatus{StatusDelivered, StatusCancelledUser, StatusCancelledAdmin}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("expected %s to be terminal", s)
		}
	}

	nonTerminal := []OrderStatus{StatusPlaced, StatusAccepted, StatusPreparing, StatusReady, StatusOutForDelivery}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("expected %s to be non-terminal", s)
		}
	}
}

func TestIsCancelled(t *testing.T) {
	if !StatusCancelledUser.IsCancelled() {
		t.Error("CANCELLED_BY_USER should be cancelled")
	}
	if !StatusCancelledAdmin.IsCancelled() {
		t.Error("CANCELLED_BY_ADMIN should be cancelled")
	}
	if StatusDelivered.IsCancelled() {
		t.Error("DELIVERED should not be cancelled")
	}
	if StatusPlaced.IsCancelled() {
		t.Error("PLACED should not be cancelled")
	}
}
