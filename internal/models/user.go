package models

import "time"

type UserAttribution struct {
	UTMSource   string `json:"utm_source,omitempty"`
	UTMMedium   string `json:"utm_medium,omitempty"`
	UTMCampaign string `json:"utm_campaign,omitempty"`
	UTMTerm     string `json:"utm_term,omitempty"`
	UTMContent  string `json:"utm_content,omitempty"`
}

type User struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Email          string          `json:"email"`
	ContactPhone   string          `json:"contact_phone,omitempty"`
	AcceptedTerms  bool            `json:"accepted_terms"`
	MarketingOptIn bool            `json:"marketing_opt_in"`
	PasswordHash   string          `json:"-"`
	Attribution    UserAttribution `json:"attribution,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
