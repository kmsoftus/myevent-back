package models

import "time"

type Guest struct {
	ID            string     `json:"id"`
	EventID       string     `json:"event_id"`
	Name          string     `json:"name"`
	Email         string     `json:"email,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	InviteCode    string     `json:"invite_code"`
	QRCodeToken   string     `json:"qr_code_token"`
	MaxCompanions int        `json:"max_companions"`
	RSVPStatus    string     `json:"rsvp_status"`
	CheckedInAt   *time.Time `json:"checked_in_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
