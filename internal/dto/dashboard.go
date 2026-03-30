package dto

type DashboardResponse struct {
	GuestsTotal         int `json:"guests_total"`
	GuestsConfirmed     int `json:"guests_confirmed"`
	GuestsPending       int `json:"guests_pending"`
	GuestsDeclined      int `json:"guests_declined"`
	CheckedInTotal      int `json:"checked_in_total"`
	GiftsTotal          int `json:"gifts_total"`
	GiftsConfirmed      int `json:"gifts_confirmed"`
	GiftsPendingPayment int `json:"gifts_pending_payment"`
}
