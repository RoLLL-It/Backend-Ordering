package domain

import (
	"testing"
	"time"
)

func TestSlotSeatsLeft(t *testing.T) {
	s := &DeliverySlot{Capacity: 10, BookedCount: 3}
	if got := s.SeatsLeft(); got != 7 {
		t.Errorf("SeatsLeft() = %d, want 7", got)
	}

	full := &DeliverySlot{Capacity: 5, BookedCount: 5}
	if got := full.SeatsLeft(); got != 0 {
		t.Errorf("SeatsLeft() on full slot = %d, want 0", got)
	}
}

func TestSlotCutoffAt(t *testing.T) {
	// SlotDate: today, StartTime: "14:30", CutoffMinutes: 30
	// CutoffAt should be today at 14:00 UTC
	today := time.Now().UTC().Truncate(24 * time.Hour) // midnight UTC
	slot := &DeliverySlot{
		SlotDate:      today,
		StartTime:     "14:30",
		CutoffMinutes: 30,
	}
	cutoff := slot.CutoffAt()

	expectedHour := 14
	expectedMin := 0
	if cutoff.Hour() != expectedHour || cutoff.Minute() != expectedMin {
		t.Errorf("CutoffAt() = %v, want %02d:%02d UTC", cutoff, expectedHour, expectedMin)
	}
}

func TestSlotCutoffAt_ZeroMinutes(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	slot := &DeliverySlot{
		SlotDate:      today,
		StartTime:     "10:00",
		CutoffMinutes: 0,
	}
	cutoff := slot.CutoffAt()
	if cutoff.Hour() != 10 || cutoff.Minute() != 0 {
		t.Errorf("CutoffAt() with 0 minutes = %v, want 10:00 UTC", cutoff)
	}
}
