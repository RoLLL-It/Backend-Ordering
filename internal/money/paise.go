// Package money provides integer-paise money helpers.
// NEVER use float64 for money. All values are int64 paise (1 rupee = 100 paise).
package money

import "fmt"

// FormatRupees formats an int64 paise value as a rupee string (e.g. ₹120.00).
func FormatRupees(paise int64) string {
	rupees := paise / 100
	paisa := paise % 100
	return fmt.Sprintf("₹%d.%02d", rupees, paisa)
}

// PaiseFromRupees converts a rupee float (display only) to integer paise.
// Use only when accepting input from trusted admin forms; never in order flow.
func PaiseFromRupees(rupees float64) int64 {
	return int64(rupees * 100)
}
