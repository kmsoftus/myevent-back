package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"myevent-back/internal/auth"
	"myevent-back/internal/config"
	"myevent-back/internal/mailer"
	"myevent-back/internal/notifier"
	"myevent-back/internal/repositories/memory"
	"myevent-back/internal/services"
	"myevent-back/internal/storage"
)

func TestPhaseOneFlow(t *testing.T) {
	router := newTestRouter(t)

	registerBody := map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	}
	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", registerBody)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	var authPayload struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Token string `json:"token"`
	}
	decodeBody(t, registerResponse, &authPayload)
	if authPayload.Token == "" {
		t.Fatal("expected token in register response")
	}

	meResponse := performJSONRequest(t, router, http.MethodGet, "/v1/auth/me", authPayload.Token, nil)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d", meResponse.Code)
	}

	createEventBody := map[string]any{
		"title":         "Casamento Ana & Joao",
		"slug":          "ana-joao",
		"type":          "casamento",
		"date":          "2026-10-12",
		"time":          "16:00",
		"location_name": "Espaco Bela Vista",
		"address":       "Rua Exemplo, 123",
	}
	createEventResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events", authPayload.Token, createEventBody)
	if createEventResponse.Code != http.StatusCreated {
		t.Fatalf("expected create event status 201, got %d", createEventResponse.Code)
	}

	var eventPayload struct {
		ID     string `json:"id"`
		Slug   string `json:"slug"`
		Status string `json:"status"`
	}
	decodeBody(t, createEventResponse, &eventPayload)
	if eventPayload.Status != "published" {
		t.Fatalf("expected published status, got %s", eventPayload.Status)
	}

	updateEventResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID, authPayload.Token, map[string]any{
		"description":     "Nosso grande dia",
		"pix_key":         "ana.pix@example.com",
		"pix_holder_name": "Ana Silva",
	})
	if updateEventResponse.Code != http.StatusOK {
		t.Fatalf("expected update event status 200, got %d", updateEventResponse.Code)
	}

	publishEventResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/status", authPayload.Token, map[string]any{
		"status": "published",
	})
	if publishEventResponse.Code != http.StatusOK {
		t.Fatalf("expected publish event status 200, got %d", publishEventResponse.Code)
	}

	publicEventResponse := performJSONRequest(t, router, http.MethodGet, "/v1/public/events/"+eventPayload.Slug, "", nil)
	if publicEventResponse.Code != http.StatusOK {
		t.Fatalf("expected public event status 200, got %d", publicEventResponse.Code)
	}

	var publicEventPayload struct {
		PixKey        string `json:"pix_key"`
		PixHolderName string `json:"pix_holder_name"`
	}
	decodeBody(t, publicEventResponse, &publicEventPayload)
	if publicEventPayload.PixKey != "ana.pix@example.com" || publicEventPayload.PixHolderName != "Ana Silva" {
		t.Fatalf("expected public event pix data, got %+v", publicEventPayload)
	}

	createGuestBody := map[string]any{
		"name":           "Maria",
		"email":          "maria@example.com",
		"phone":          "79999999999",
		"max_companions": 2,
	}
	createGuestResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/guests", authPayload.Token, createGuestBody)
	if createGuestResponse.Code != http.StatusCreated {
		t.Fatalf("expected create guest status 201, got %d", createGuestResponse.Code)
	}

	var guestPayload struct {
		ID            string `json:"id"`
		InviteCode    string `json:"invite_code"`
		QRCodeToken   string `json:"qr_code_token"`
		MaxCompanions int    `json:"max_companions"`
	}
	decodeBody(t, createGuestResponse, &guestPayload)
	if guestPayload.InviteCode == "" || guestPayload.QRCodeToken == "" {
		t.Fatal("expected guest identifiers to be generated")
	}

	updateGuestResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/guests/"+guestPayload.ID, authPayload.Token, map[string]any{
		"max_companions": 3,
	})
	if updateGuestResponse.Code != http.StatusOK {
		t.Fatalf("expected update guest status 200, got %d", updateGuestResponse.Code)
	}

	listGuestsResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/guests", authPayload.Token, nil)
	if listGuestsResponse.Code != http.StatusOK {
		t.Fatalf("expected list guests status 200, got %d", listGuestsResponse.Code)
	}

	var guestsPayload struct {
		Items      []map[string]any `json:"items"`
		Page       int              `json:"page"`
		PageSize   int              `json:"page_size"`
		Total      int              `json:"total"`
		TotalPages int              `json:"total_pages"`
	}
	decodeBody(t, listGuestsResponse, &guestsPayload)
	if len(guestsPayload.Items) != 1 {
		t.Fatalf("expected 1 guest, got %d", len(guestsPayload.Items))
	}
	if guestsPayload.Page != 1 || guestsPayload.PageSize != 100 || guestsPayload.Total != 1 || guestsPayload.TotalPages != 1 {
		t.Fatalf("unexpected guests pagination payload: %+v", guestsPayload)
	}

	submitRSVPResponse := performJSONRequest(t, router, http.MethodPost, "/v1/public/events/"+eventPayload.Slug+"/rsvp", "", map[string]any{
		"guest_identifier": guestPayload.InviteCode,
		"status":           "confirmed",
		"companions_count": 2,
		"message":          "Estaremos la!",
	})
	if submitRSVPResponse.Code != http.StatusOK {
		t.Fatalf("expected submit RSVP status 200, got %d", submitRSVPResponse.Code)
	}

	var rsvpPayload struct {
		GuestID         string `json:"guest_id"`
		GuestName       string `json:"guest_name"`
		Status          string `json:"status"`
		CompanionsCount int    `json:"companions_count"`
	}
	decodeBody(t, submitRSVPResponse, &rsvpPayload)
	if rsvpPayload.GuestID != guestPayload.ID {
		t.Fatalf("expected RSVP guest_id %s, got %s", guestPayload.ID, rsvpPayload.GuestID)
	}
	if rsvpPayload.Status != "confirmed" || rsvpPayload.CompanionsCount != 2 {
		t.Fatalf("unexpected RSVP payload: %+v", rsvpPayload)
	}

	listRSVPsResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/rsvps", authPayload.Token, nil)
	if listRSVPsResponse.Code != http.StatusOK {
		t.Fatalf("expected list RSVPs status 200, got %d", listRSVPsResponse.Code)
	}

	var rsvpsPayload struct {
		Items      []map[string]any `json:"items"`
		Page       int              `json:"page"`
		PageSize   int              `json:"page_size"`
		Total      int              `json:"total"`
		TotalPages int              `json:"total_pages"`
	}
	decodeBody(t, listRSVPsResponse, &rsvpsPayload)
	if len(rsvpsPayload.Items) != 1 {
		t.Fatalf("expected 1 RSVP, got %d", len(rsvpsPayload.Items))
	}
	if rsvpsPayload.Page != 1 || rsvpsPayload.PageSize != 100 || rsvpsPayload.Total != 1 || rsvpsPayload.TotalPages != 1 {
		t.Fatalf("unexpected RSVPs pagination payload: %+v", rsvpsPayload)
	}

	dashboardResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/dashboard", authPayload.Token, nil)
	if dashboardResponse.Code != http.StatusOK {
		t.Fatalf("expected dashboard status 200, got %d", dashboardResponse.Code)
	}

	var dashboardPayload struct {
		GuestsTotal         int `json:"guests_total"`
		GuestsConfirmed     int `json:"guests_confirmed"`
		GuestsPending       int `json:"guests_pending"`
		GuestsDeclined      int `json:"guests_declined"`
		CheckedInTotal      int `json:"checked_in_total"`
		GiftsTotal          int `json:"gifts_total"`
		GiftsConfirmed      int `json:"gifts_confirmed"`
		GiftsPendingPayment int `json:"gifts_pending_payment"`
	}
	decodeBody(t, dashboardResponse, &dashboardPayload)
	if dashboardPayload.GuestsTotal != 1 || dashboardPayload.GuestsConfirmed != 1 || dashboardPayload.GuestsPending != 0 {
		t.Fatalf("unexpected dashboard payload: %+v", dashboardPayload)
	}
	if dashboardPayload.GiftsTotal != 0 || dashboardPayload.GiftsConfirmed != 0 || dashboardPayload.GiftsPendingPayment != 0 {
		t.Fatalf("expected gift counters to remain zero in phase 2, got %+v", dashboardPayload)
	}

	qrPayloadResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/guests/"+guestPayload.ID+"/qrcode", authPayload.Token, nil)
	if qrPayloadResponse.Code != http.StatusOK {
		t.Fatalf("expected QR payload status 200, got %d", qrPayloadResponse.Code)
	}

	var qrPayload struct {
		GuestID     string `json:"guest_id"`
		QRCodeToken string `json:"qr_code_token"`
		CheckinURL  string `json:"checkin_url"`
	}
	decodeBody(t, qrPayloadResponse, &qrPayload)
	if qrPayload.GuestID != guestPayload.ID || qrPayload.QRCodeToken != guestPayload.QRCodeToken {
		t.Fatalf("unexpected QR payload: %+v", qrPayload)
	}
	if qrPayload.CheckinURL != "/checkin/"+guestPayload.QRCodeToken {
		t.Fatalf("expected checkin URL to embed QR token, got %s", qrPayload.CheckinURL)
	}
}

func TestPhaseThreeFlow(t *testing.T) {
	router := newTestRouter(t)

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-phase3@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	var authPayload struct {
		Token string `json:"token"`
	}
	decodeBody(t, registerResponse, &authPayload)

	createEventResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events", authPayload.Token, map[string]any{
		"title": "Festa da Maria",
		"slug":  "festa-da-maria",
		"date":  "2026-12-20",
		"time":  "18:00",
	})
	if createEventResponse.Code != http.StatusCreated {
		t.Fatalf("expected create event status 201, got %d", createEventResponse.Code)
	}

	var eventPayload struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	decodeBody(t, createEventResponse, &eventPayload)

	publishEventResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/status", authPayload.Token, map[string]any{
		"status": "published",
	})
	if publishEventResponse.Code != http.StatusOK {
		t.Fatalf("expected publish event status 200, got %d", publishEventResponse.Code)
	}

	createGuestResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/guests", authPayload.Token, map[string]any{
		"name": "Maria",
	})
	if createGuestResponse.Code != http.StatusCreated {
		t.Fatalf("expected create guest status 201, got %d", createGuestResponse.Code)
	}

	var guestPayload struct {
		ID          string `json:"id"`
		QRCodeToken string `json:"qr_code_token"`
	}
	decodeBody(t, createGuestResponse, &guestPayload)

	checkInResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/checkin", authPayload.Token, map[string]any{
		"qr_code_token": guestPayload.QRCodeToken,
	})
	if checkInResponse.Code != http.StatusOK {
		t.Fatalf("expected check-in status 200, got %d", checkInResponse.Code)
	}

	var checkInPayload struct {
		Success bool `json:"success"`
		Guest   struct {
			ID          string `json:"id"`
			CheckedInAt string `json:"checked_in_at"`
		} `json:"guest"`
	}
	decodeBody(t, checkInResponse, &checkInPayload)
	if !checkInPayload.Success || checkInPayload.Guest.ID != guestPayload.ID || checkInPayload.Guest.CheckedInAt == "" {
		t.Fatalf("unexpected check-in payload: %+v", checkInPayload)
	}

	duplicateCheckInResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/checkin", authPayload.Token, map[string]any{
		"guest_id": guestPayload.ID,
	})
	if duplicateCheckInResponse.Code != http.StatusOK {
		t.Fatalf("expected duplicate check-in status 200, got %d", duplicateCheckInResponse.Code)
	}

	var duplicateCheckInPayload struct {
		Success bool `json:"success"`
		Guest   struct {
			ID               string `json:"id"`
			AlreadyCheckedIn bool   `json:"already_checked_in"`
			CheckedInAt      string `json:"checked_in_at"`
		} `json:"guest"`
	}
	decodeBody(t, duplicateCheckInResponse, &duplicateCheckInPayload)
	if !duplicateCheckInPayload.Success || !duplicateCheckInPayload.Guest.AlreadyCheckedIn || duplicateCheckInPayload.Guest.ID != guestPayload.ID || duplicateCheckInPayload.Guest.CheckedInAt == "" {
		t.Fatalf("unexpected duplicate check-in payload: %+v", duplicateCheckInPayload)
	}

	listCheckInGuestsResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/checkin/guests", authPayload.Token, nil)
	if listCheckInGuestsResponse.Code != http.StatusOK {
		t.Fatalf("expected check-in guests status 200, got %d", listCheckInGuestsResponse.Code)
	}

	var checkInGuestsPayload struct {
		Items      []map[string]any `json:"items"`
		Page       int              `json:"page"`
		PageSize   int              `json:"page_size"`
		Total      int              `json:"total"`
		TotalPages int              `json:"total_pages"`
	}
	decodeBody(t, listCheckInGuestsResponse, &checkInGuestsPayload)
	if len(checkInGuestsPayload.Items) != 1 {
		t.Fatalf("expected 1 guest in check-in list, got %d", len(checkInGuestsPayload.Items))
	}
	if checkInGuestsPayload.Page != 1 || checkInGuestsPayload.PageSize != 100 || checkInGuestsPayload.Total != 1 || checkInGuestsPayload.TotalPages != 1 {
		t.Fatalf("unexpected check-in guests pagination payload: %+v", checkInGuestsPayload)
	}
	if checkInGuestsPayload.Items[0]["checked_in_at"] == nil {
		t.Fatalf("expected checked_in_at in guest list payload, got %+v", checkInGuestsPayload.Items[0])
	}

	createGiftResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/gifts", authPayload.Token, map[string]any{
		"title":             "Jogo de panelas",
		"value_cents":       25990,
		"allow_reservation": true,
		"allow_pix":         true,
	})
	if createGiftResponse.Code != http.StatusCreated {
		t.Fatalf("expected create gift status 201, got %d", createGiftResponse.Code)
	}

	var giftOnePayload struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeBody(t, createGiftResponse, &giftOnePayload)
	if giftOnePayload.Status != "available" {
		t.Fatalf("expected new gift to be available, got %s", giftOnePayload.Status)
	}

	createSecondGiftResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/gifts", authPayload.Token, map[string]any{
		"title":       "Liquidificador",
		"value_cents": 18990,
	})
	if createSecondGiftResponse.Code != http.StatusCreated {
		t.Fatalf("expected create second gift status 201, got %d", createSecondGiftResponse.Code)
	}

	var giftTwoPayload struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	decodeBody(t, createSecondGiftResponse, &giftTwoPayload)

	updateGiftResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/gifts/"+giftTwoPayload.ID, authPayload.Token, map[string]any{
		"title": "Liquidificador Pro",
	})
	if updateGiftResponse.Code != http.StatusOK {
		t.Fatalf("expected update gift status 200, got %d", updateGiftResponse.Code)
	}

	getGiftResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/gifts/"+giftTwoPayload.ID, authPayload.Token, nil)
	if getGiftResponse.Code != http.StatusOK {
		t.Fatalf("expected get gift status 200, got %d", getGiftResponse.Code)
	}

	var getGiftPayload struct {
		Title string `json:"title"`
	}
	decodeBody(t, getGiftResponse, &getGiftPayload)
	if getGiftPayload.Title != "Liquidificador Pro" {
		t.Fatalf("expected updated gift title, got %s", getGiftPayload.Title)
	}

	publicGiftsResponse := performJSONRequest(t, router, http.MethodGet, "/v1/public/events/"+eventPayload.Slug+"/gifts", "", nil)
	if publicGiftsResponse.Code != http.StatusOK {
		t.Fatalf("expected public gifts status 200, got %d", publicGiftsResponse.Code)
	}

	var publicGiftsPayload struct {
		Items []map[string]any `json:"items"`
	}
	decodeBody(t, publicGiftsResponse, &publicGiftsPayload)
	if len(publicGiftsPayload.Items) != 2 {
		t.Fatalf("expected 2 public gifts, got %d", len(publicGiftsPayload.Items))
	}

	reserveGiftResponse := performJSONRequest(t, router, http.MethodPost, "/v1/public/events/"+eventPayload.Slug+"/gifts/"+giftOnePayload.ID+"/reserve", "", map[string]any{
		"guest_name":    "Carlos",
		"guest_contact": "79999999999",
		"message":       "Vou dar esse presente",
	})
	if reserveGiftResponse.Code != http.StatusCreated {
		t.Fatalf("expected reserve gift status 201, got %d", reserveGiftResponse.Code)
	}

	var reservePayload struct {
		ID        string `json:"id"`
		GiftID    string `json:"gift_id"`
		GiftTitle string `json:"gift_title"`
		Type      string `json:"type"`
		Status    string `json:"status"`
	}
	decodeBody(t, reserveGiftResponse, &reservePayload)
	if reservePayload.GiftID != giftOnePayload.ID || reservePayload.Type != "reservation" || reservePayload.Status != "pending" {
		t.Fatalf("unexpected reserve payload: %+v", reservePayload)
	}

	duplicateReserveResponse := performJSONRequest(t, router, http.MethodPost, "/v1/public/events/"+eventPayload.Slug+"/gifts/"+giftOnePayload.ID+"/reserve", "", map[string]any{
		"guest_name": "Outro convidado",
	})
	if duplicateReserveResponse.Code != http.StatusConflict {
		t.Fatalf("expected duplicate reserve status 409, got %d", duplicateReserveResponse.Code)
	}

	listTransactionsResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/gift-transactions", authPayload.Token, nil)
	if listTransactionsResponse.Code != http.StatusOK {
		t.Fatalf("expected list transactions status 200, got %d", listTransactionsResponse.Code)
	}

	var transactionsPayload struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	decodeBody(t, listTransactionsResponse, &transactionsPayload)
	if len(transactionsPayload.Items) != 1 {
		t.Fatalf("expected 1 gift transaction, got %d", len(transactionsPayload.Items))
	}

	confirmTransactionResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/gift-transactions/"+reservePayload.ID+"/confirm", authPayload.Token, map[string]any{
		"status": "confirmed",
	})
	if confirmTransactionResponse.Code != http.StatusOK {
		t.Fatalf("expected confirm transaction status 200, got %d", confirmTransactionResponse.Code)
	}

	pixGiftResponse := performJSONRequest(t, router, http.MethodPost, "/v1/public/events/"+eventPayload.Slug+"/gifts/"+giftTwoPayload.ID+"/pix", "", map[string]any{
		"guest_name": "Fernanda",
		"message":    "Ja fiz o Pix",
	})
	if pixGiftResponse.Code != http.StatusCreated {
		t.Fatalf("expected pix gift status 201, got %d", pixGiftResponse.Code)
	}

	var pixPayload struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}
	decodeBody(t, pixGiftResponse, &pixPayload)
	if pixPayload.Type != "pix" || pixPayload.Status != "pending" {
		t.Fatalf("unexpected pix payload: %+v", pixPayload)
	}

	cancelTransactionResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/gift-transactions/"+pixPayload.ID+"/cancel", authPayload.Token, map[string]any{
		"status": "canceled",
	})
	if cancelTransactionResponse.Code != http.StatusOK {
		t.Fatalf("expected cancel transaction status 200, got %d", cancelTransactionResponse.Code)
	}

	listGiftsResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/gifts", authPayload.Token, nil)
	if listGiftsResponse.Code != http.StatusOK {
		t.Fatalf("expected list gifts status 200, got %d", listGiftsResponse.Code)
	}

	var giftsPayload []map[string]any
	decodeBody(t, listGiftsResponse, &giftsPayload)
	if len(giftsPayload) != 2 {
		t.Fatalf("expected 2 gifts, got %d", len(giftsPayload))
	}

	dashboardResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/dashboard", authPayload.Token, nil)
	if dashboardResponse.Code != http.StatusOK {
		t.Fatalf("expected dashboard status 200, got %d", dashboardResponse.Code)
	}

	var dashboardPayload struct {
		CheckedInTotal      int `json:"checked_in_total"`
		GiftsTotal          int `json:"gifts_total"`
		GiftsConfirmed      int `json:"gifts_confirmed"`
		GiftsPendingPayment int `json:"gifts_pending_payment"`
	}
	decodeBody(t, dashboardResponse, &dashboardPayload)
	if dashboardPayload.CheckedInTotal != 1 || dashboardPayload.GiftsTotal != 2 || dashboardPayload.GiftsConfirmed != 1 || dashboardPayload.GiftsPendingPayment != 0 {
		t.Fatalf("unexpected phase 3 dashboard payload: %+v", dashboardPayload)
	}

	deleteGiftResponse := performJSONRequest(t, router, http.MethodDelete, "/v1/events/"+eventPayload.ID+"/gifts/"+giftTwoPayload.ID, authPayload.Token, nil)
	if deleteGiftResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete gift status 204, got %d", deleteGiftResponse.Code)
	}
}

func TestPhaseFourFlow(t *testing.T) {
	router := newTestRouter(t)

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-phase4@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	var authPayload struct {
		Token string `json:"token"`
	}
	decodeBody(t, registerResponse, &authPayload)

	createEventResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events", authPayload.Token, map[string]any{
		"title": "Sunset Party",
		"slug":  "sunset-party",
		"theme": "party",
	})
	if createEventResponse.Code != http.StatusCreated {
		t.Fatalf("expected create event status 201, got %d", createEventResponse.Code)
	}

	var eventPayload struct {
		ID              string `json:"id"`
		Slug            string `json:"slug"`
		Theme           string `json:"theme"`
		PrimaryColor    string `json:"primary_color"`
		SecondaryColor  string `json:"secondary_color"`
		BackgroundColor string `json:"background_color"`
		TextColor       string `json:"text_color"`
	}
	decodeBody(t, createEventResponse, &eventPayload)
	if eventPayload.Theme != "party" || eventPayload.PrimaryColor != "#db2777" || eventPayload.BackgroundColor != "#1f2937" {
		t.Fatalf("unexpected party theme palette: %+v", eventPayload)
	}

	updateEventResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID, authPayload.Token, map[string]any{
		"theme": "minimal",
	})
	if updateEventResponse.Code != http.StatusOK {
		t.Fatalf("expected update event status 200, got %d", updateEventResponse.Code)
	}

	var updatedEventPayload struct {
		Theme           string `json:"theme"`
		PrimaryColor    string `json:"primary_color"`
		SecondaryColor  string `json:"secondary_color"`
		BackgroundColor string `json:"background_color"`
		TextColor       string `json:"text_color"`
	}
	decodeBody(t, updateEventResponse, &updatedEventPayload)
	if updatedEventPayload.Theme != "minimal" || updatedEventPayload.PrimaryColor != "#1f2937" || updatedEventPayload.BackgroundColor != "#ffffff" {
		t.Fatalf("expected minimal theme palette after theme switch, got %+v", updatedEventPayload)
	}

	uploadResponse := performMultipartRequest(t, router, "/v1/uploads", authPayload.Token, "events/covers", "cover.png", samplePNG)
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d", uploadResponse.Code)
	}

	var uploadPayload struct {
		URL string `json:"url"`
		Key string `json:"key"`
	}
	decodeBody(t, uploadResponse, &uploadPayload)
	if !strings.HasPrefix(uploadPayload.Key, "events/covers/") {
		t.Fatalf("expected upload key to use covers folder, got %s", uploadPayload.Key)
	}

	uploadedFileResponse := performJSONRequest(t, router, http.MethodGet, strings.TrimPrefix(uploadPayload.URL, "http://localhost:8080"), "", nil)
	if uploadedFileResponse.Code != http.StatusOK {
		t.Fatalf("expected uploaded file to be served locally, got %d", uploadedFileResponse.Code)
	}

	updateCoverResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID, authPayload.Token, map[string]any{
		"cover_image_url": uploadPayload.URL,
	})
	if updateCoverResponse.Code != http.StatusOK {
		t.Fatalf("expected cover image update status 200, got %d", updateCoverResponse.Code)
	}

	invalidGiftResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/gifts", authPayload.Token, map[string]any{
		"title":         "Gift URL invalido",
		"external_link": "nota-url",
	})
	if invalidGiftResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid gift url status 400, got %d", invalidGiftResponse.Code)
	}

	publishEventResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/status", authPayload.Token, map[string]any{
		"status": "published",
	})
	if publishEventResponse.Code != http.StatusOK {
		t.Fatalf("expected publish event status 200, got %d", publishEventResponse.Code)
	}

	createGiftResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/gifts", authPayload.Token, map[string]any{
		"title":             "Poltrona",
		"image_url":         uploadPayload.URL,
		"external_link":     "https://example.com/poltrona",
		"allow_reservation": true,
	})
	if createGiftResponse.Code != http.StatusCreated {
		t.Fatalf("expected create gift status 201, got %d", createGiftResponse.Code)
	}

	var giftPayload struct {
		ID string `json:"id"`
	}
	decodeBody(t, createGiftResponse, &giftPayload)

	reserveResponse := performJSONRequest(t, router, http.MethodPost, "/v1/public/events/"+eventPayload.Slug+"/gifts/"+giftPayload.ID+"/reserve", "", map[string]any{
		"guest_name": "Carlos",
	})
	if reserveResponse.Code != http.StatusCreated {
		t.Fatalf("expected reserve response 201, got %d", reserveResponse.Code)
	}

	var transactionPayload struct {
		ID string `json:"id"`
	}
	decodeBody(t, reserveResponse, &transactionPayload)

	confirmResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/gift-transactions/"+transactionPayload.ID+"/confirm", authPayload.Token, map[string]any{
		"status": "confirmed",
	})
	if confirmResponse.Code != http.StatusOK {
		t.Fatalf("expected confirm response 200, got %d", confirmResponse.Code)
	}

	cancelConfirmedResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID+"/gift-transactions/"+transactionPayload.ID+"/cancel", authPayload.Token, map[string]any{
		"status": "canceled",
	})
	if cancelConfirmedResponse.Code != http.StatusConflict {
		t.Fatalf("expected cancel confirmed transaction status 409, got %d", cancelConfirmedResponse.Code)
	}

	deleteUploadResponse := performJSONRequest(t, router, http.MethodDelete, "/v1/uploads", authPayload.Token, map[string]any{
		"key": uploadPayload.Key,
	})
	if deleteUploadResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete upload status 204, got %d", deleteUploadResponse.Code)
	}
}

func TestAuthResponsesAreLocalizedAndDetailed(t *testing.T) {
	router, passwordResetSender, _, _, _ := newTestRouterWithDeps(t)

	invalidRegisterResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "email-invalido",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if invalidRegisterResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid register status 400, got %d", invalidRegisterResponse.Code)
	}

	var invalidRegisterPayload struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
	}
	decodeBody(t, invalidRegisterResponse, &invalidRegisterPayload)
	if invalidRegisterPayload.Message != "Informe um e-mail valido." {
		t.Fatalf("expected localized register message, got %q", invalidRegisterPayload.Message)
	}
	if invalidRegisterPayload.Code != "auth_email_invalid" {
		t.Fatalf("expected auth_email_invalid code, got %q", invalidRegisterPayload.Code)
	}
	if len(invalidRegisterPayload.Details) != 1 || invalidRegisterPayload.Details[0].Field != "email" {
		t.Fatalf("expected email field detail, got %+v", invalidRegisterPayload.Details)
	}

	missingTermsResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":     "Kaleb",
		"email":    "kaleb-sem-termos@example.com",
		"password": "Senha123",
	})
	if missingTermsResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected missing terms status 400, got %d", missingTermsResponse.Code)
	}

	var missingTermsPayload struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Details []struct {
			Field string `json:"field"`
		} `json:"details"`
	}
	decodeBody(t, missingTermsResponse, &missingTermsPayload)
	if missingTermsPayload.Message != "Voce precisa aceitar os Termos de Uso e a Politica de Privacidade." {
		t.Fatalf("expected missing terms message, got %q", missingTermsPayload.Message)
	}
	if missingTermsPayload.Code != "auth_terms_required" {
		t.Fatalf("expected auth_terms_required code, got %q", missingTermsPayload.Code)
	}
	if len(missingTermsPayload.Details) != 1 || missingTermsPayload.Details[0].Field != "accepted_terms" {
		t.Fatalf("expected accepted_terms field detail, got %+v", missingTermsPayload.Details)
	}

	firstRegisterResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-auth@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if firstRegisterResponse.Code != http.StatusCreated {
		t.Fatalf("expected first register status 201, got %d", firstRegisterResponse.Code)
	}

	duplicateRegisterResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-auth@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if duplicateRegisterResponse.Code != http.StatusConflict {
		t.Fatalf("expected duplicate register status 409, got %d", duplicateRegisterResponse.Code)
	}

	var duplicateRegisterPayload struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	decodeBody(t, duplicateRegisterResponse, &duplicateRegisterPayload)
	if duplicateRegisterPayload.Message != "Este e-mail ja esta cadastrado." {
		t.Fatalf("expected duplicate email message, got %q", duplicateRegisterPayload.Message)
	}
	if duplicateRegisterPayload.Code != "auth_email_already_registered" {
		t.Fatalf("expected auth_email_already_registered code, got %q", duplicateRegisterPayload.Code)
	}

	invalidLoginResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email":    "kaleb-auth@example.com",
		"password": "senha-errada",
	})
	if invalidLoginResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid login status 401, got %d", invalidLoginResponse.Code)
	}

	var invalidLoginPayload struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	decodeBody(t, invalidLoginResponse, &invalidLoginPayload)
	if invalidLoginPayload.Message != "E-mail ou senha invalidos." {
		t.Fatalf("expected invalid credentials message, got %q", invalidLoginPayload.Message)
	}
	if invalidLoginPayload.Code != "auth_invalid_credentials" {
		t.Fatalf("expected auth_invalid_credentials code, got %q", invalidLoginPayload.Code)
	}

	forgotPasswordResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/forgot-password", "", map[string]any{
		"email": "kaleb-auth@example.com",
	})
	if forgotPasswordResponse.Code != http.StatusOK {
		t.Fatalf("expected forgot password status 200, got %d", forgotPasswordResponse.Code)
	}

	var forgotPasswordPayload struct {
		Message string `json:"message"`
	}
	decodeBody(t, forgotPasswordResponse, &forgotPasswordPayload)
	if forgotPasswordPayload.Message == "" {
		t.Fatal("expected forgot password message")
	}
	if len(passwordResetSender.messages) != 1 {
		t.Fatalf("expected 1 password reset email, got %d", len(passwordResetSender.messages))
	}

	resetURL, err := url.Parse(passwordResetSender.messages[0].ResetURL)
	if err != nil {
		t.Fatalf("parse reset url: %v", err)
	}

	resetToken := strings.TrimPrefix(resetURL.EscapedPath(), "/redefinir-senha/")
	if resetToken == "" {
		t.Fatal("expected reset token in reset URL")
	}

	resetPasswordResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/reset-password", "", map[string]any{
		"token":    resetToken,
		"password": "NovaSenha9",
	})
	if resetPasswordResponse.Code != http.StatusOK {
		t.Fatalf("expected reset password status 200, got %d", resetPasswordResponse.Code)
	}

	reuseResetResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/reset-password", "", map[string]any{
		"token":    resetToken,
		"password": "OutraSenha9",
	})
	if reuseResetResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected reused reset token status 400, got %d", reuseResetResponse.Code)
	}

	oldPasswordLoginResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email":    "kaleb-auth@example.com",
		"password": "Senha123",
	})
	if oldPasswordLoginResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected old password login to fail, got %d", oldPasswordLoginResponse.Code)
	}

	newPasswordLoginResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email":    "kaleb-auth@example.com",
		"password": "NovaSenha9",
	})
	if newPasswordLoginResponse.Code != http.StatusOK {
		t.Fatalf("expected login with new password to succeed, got %d", newPasswordLoginResponse.Code)
	}
}

func TestRegisterSendsTelegramNotification(t *testing.T) {
	router, _, registrationSender, store, _ := newTestRouterWithDeps(t)

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":             "Kaleb",
		"email":            "kaleb-telegram@example.com",
		"password":         "Senha123",
		"contact_phone":    "(11) 99999-9999",
		"accepted_terms":   true,
		"marketing_opt_in": true,
		"utm_source":       "google",
		"utm_medium":       "cpc",
		"utm_campaign":     "casamento-2026",
	})
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	if len(registrationSender.messages) != 1 {
		t.Fatalf("expected 1 registration notification, got %d", len(registrationSender.messages))
	}

	message := registrationSender.messages[0]
	if message.Name != "Kaleb" {
		t.Fatalf("expected registration name Kaleb, got %q", message.Name)
	}
	if message.Email != "kaleb-telegram@example.com" {
		t.Fatalf("expected registration email to match, got %q", message.Email)
	}
	if message.ContactPhone != "(11) 99999-9999" {
		t.Fatalf("expected registration contact phone to match, got %q", message.ContactPhone)
	}
	if message.UserID == "" {
		t.Fatal("expected registration notification to include user ID")
	}
	if message.CreatedAt.IsZero() {
		t.Fatal("expected registration notification to include created_at")
	}
	if message.Attribution.UTMSource != "google" || message.Attribution.UTMMedium != "cpc" || message.Attribution.UTMCampaign != "casamento-2026" {
		t.Fatalf("expected registration notification attribution, got %+v", message.Attribution)
	}

	user, err := store.Users().GetByEmail(context.Background(), "kaleb-telegram@example.com")
	if err != nil {
		t.Fatalf("load saved user: %v", err)
	}
	if user.ContactPhone != "(11) 99999-9999" {
		t.Fatalf("expected saved user contact phone, got %q", user.ContactPhone)
	}
	if !user.AcceptedTerms {
		t.Fatal("expected accepted_terms to be saved")
	}
	if !user.MarketingOptIn {
		t.Fatal("expected marketing_opt_in to be saved")
	}
	if user.Attribution.UTMSource != "google" || user.Attribution.UTMMedium != "cpc" || user.Attribution.UTMCampaign != "casamento-2026" {
		t.Fatalf("expected saved user attribution, got %+v", user.Attribution)
	}
}

func TestUpdateProfileUpdatesNameContactPhoneAndProfilePhoto(t *testing.T) {
	router := newTestRouter(t)

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-profile@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	var authPayload struct {
		Token string `json:"token"`
	}
	decodeBody(t, registerResponse, &authPayload)

	updateResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/auth/me", authPayload.Token, map[string]any{
		"name":              "Kaleb Moura",
		"contact_phone":     "11999998888",
		"profile_photo_url": "https://cdn.example.com/users/profiles/kaleb.jpg",
	})
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update profile status 200, got %d", updateResponse.Code)
	}

	var updatePayload struct {
		Message string `json:"message"`
		User    struct {
			Name            string `json:"name"`
			ContactPhone    string `json:"contact_phone"`
			ProfilePhotoURL string `json:"profile_photo_url"`
		} `json:"user"`
	}
	decodeBody(t, updateResponse, &updatePayload)
	if updatePayload.Message != "Dados atualizados com sucesso." {
		t.Fatalf("expected update message, got %q", updatePayload.Message)
	}
	if updatePayload.User.Name != "Kaleb Moura" {
		t.Fatalf("expected updated name, got %q", updatePayload.User.Name)
	}
	if updatePayload.User.ContactPhone != "(11) 99999-8888" {
		t.Fatalf("expected formatted contact phone, got %q", updatePayload.User.ContactPhone)
	}
	if updatePayload.User.ProfilePhotoURL != "https://cdn.example.com/users/profiles/kaleb.jpg" {
		t.Fatalf("expected updated profile photo url, got %q", updatePayload.User.ProfilePhotoURL)
	}

	meResponse := performJSONRequest(t, router, http.MethodGet, "/v1/auth/me", authPayload.Token, nil)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d", meResponse.Code)
	}

	var mePayload struct {
		Name            string `json:"name"`
		ContactPhone    string `json:"contact_phone"`
		ProfilePhotoURL string `json:"profile_photo_url"`
	}
	decodeBody(t, meResponse, &mePayload)
	if mePayload.Name != "Kaleb Moura" {
		t.Fatalf("expected persisted updated name, got %q", mePayload.Name)
	}
	if mePayload.ContactPhone != "(11) 99999-8888" {
		t.Fatalf("expected persisted formatted contact phone, got %q", mePayload.ContactPhone)
	}
	if mePayload.ProfilePhotoURL != "https://cdn.example.com/users/profiles/kaleb.jpg" {
		t.Fatalf("expected persisted profile photo url, got %q", mePayload.ProfilePhotoURL)
	}
}

func TestCreateEventRejectsPastDate(t *testing.T) {
	router := newTestRouter(t)

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-past-date@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	var authPayload struct {
		Token string `json:"token"`
	}
	decodeBody(t, registerResponse, &authPayload)

	yesterday := time.Now().In(time.FixedZone("America/Sao_Paulo", -3*60*60)).AddDate(0, 0, -1).Format("2006-01-02")
	createEventResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events", authPayload.Token, map[string]any{
		"title":         "Evento no passado",
		"slug":          "evento-no-passado",
		"date":          yesterday,
		"location_name": "Salao Antigo",
	})
	if createEventResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected create event status 400 for past date, got %d", createEventResponse.Code)
	}

	var payload struct {
		Message string `json:"message"`
	}
	decodeBody(t, createEventResponse, &payload)
	if payload.Message != "A data do evento nao pode ser anterior a hoje." {
		t.Fatalf("expected past date message, got %q", payload.Message)
	}
}

func TestDeleteAccountRemovesManagedUploadsAndUserData(t *testing.T) {
	router, _, _, _, uploadDir := newTestRouterWithDeps(t)

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-delete@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	var authPayload struct {
		Token string `json:"token"`
	}
	decodeBody(t, registerResponse, &authPayload)

	uploadResponse := performMultipartRequest(t, router, "/v1/uploads", authPayload.Token, "events/covers", "cover.png", samplePNG)
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d", uploadResponse.Code)
	}

	var uploadPayload struct {
		Key string `json:"key"`
		URL string `json:"url"`
	}
	decodeBody(t, uploadResponse, &uploadPayload)

	profileUploadResponse := performMultipartRequest(t, router, "/v1/uploads", authPayload.Token, "users/profiles", "profile.png", samplePNG)
	if profileUploadResponse.Code != http.StatusCreated {
		t.Fatalf("expected profile upload status 201, got %d", profileUploadResponse.Code)
	}

	var profileUploadPayload struct {
		Key string `json:"key"`
		URL string `json:"url"`
	}
	decodeBody(t, profileUploadResponse, &profileUploadPayload)

	updateProfileResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/auth/me", authPayload.Token, map[string]any{
		"name":              "Kaleb",
		"profile_photo_url": profileUploadPayload.URL,
	})
	if updateProfileResponse.Code != http.StatusOK {
		t.Fatalf("expected profile update status 200, got %d", updateProfileResponse.Code)
	}

	createEventResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events", authPayload.Token, map[string]any{
		"title":           "Evento para excluir",
		"slug":            "evento-para-excluir",
		"cover_image_url": uploadPayload.URL,
	})
	if createEventResponse.Code != http.StatusCreated {
		t.Fatalf("expected create event status 201, got %d", createEventResponse.Code)
	}

	var eventPayload struct {
		ID string `json:"id"`
	}
	decodeBody(t, createEventResponse, &eventPayload)

	createGiftResponse := performJSONRequest(t, router, http.MethodPost, "/v1/events/"+eventPayload.ID+"/gifts", authPayload.Token, map[string]any{
		"title":     "Presente com a mesma imagem",
		"image_url": uploadPayload.URL,
	})
	if createGiftResponse.Code != http.StatusCreated {
		t.Fatalf("expected create gift status 201, got %d", createGiftResponse.Code)
	}

	assetPath := filepath.Join(uploadDir, filepath.FromSlash(uploadPayload.Key))
	if _, err := os.Stat(assetPath); err != nil {
		t.Fatalf("expected uploaded asset to exist before account deletion: %v", err)
	}
	profileAssetPath := filepath.Join(uploadDir, filepath.FromSlash(profileUploadPayload.Key))
	if _, err := os.Stat(profileAssetPath); err != nil {
		t.Fatalf("expected profile upload asset to exist before account deletion: %v", err)
	}

	deleteAccountResponse := performJSONRequest(t, router, http.MethodDelete, "/v1/auth/me", authPayload.Token, map[string]any{
		"email": "kaleb-delete@example.com",
	})
	if deleteAccountResponse.Code != http.StatusOK {
		t.Fatalf("expected delete account status 200, got %d", deleteAccountResponse.Code)
	}

	if _, err := os.Stat(assetPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed upload to be deleted, got err=%v", err)
	}
	if _, err := os.Stat(profileAssetPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed profile upload to be deleted, got err=%v", err)
	}

	meResponse := performJSONRequest(t, router, http.MethodGet, "/v1/auth/me", authPayload.Token, nil)
	if meResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected deleted account token to be rejected by /me, got %d", meResponse.Code)
	}

	loginResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", "", map[string]any{
		"email":    "kaleb-delete@example.com",
		"password": "Senha123",
	})
	if loginResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected deleted account login to fail, got %d", loginResponse.Code)
	}
}

func TestDeleteAccountRequiresMatchingConfirmationEmail(t *testing.T) {
	router := newTestRouter(t)

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-delete-confirm@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	var authPayload struct {
		Token string `json:"token"`
	}
	decodeBody(t, registerResponse, &authPayload)

	deleteAccountResponse := performJSONRequest(t, router, http.MethodDelete, "/v1/auth/me", authPayload.Token, map[string]any{
		"email": "outro-email@example.com",
	})
	if deleteAccountResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected delete account status 400, got %d", deleteAccountResponse.Code)
	}

	var deletePayload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
	}
	decodeBody(t, deleteAccountResponse, &deletePayload)
	if deletePayload.Code != "auth_delete_email_mismatch" {
		t.Fatalf("expected auth_delete_email_mismatch code, got %q", deletePayload.Code)
	}
	if deletePayload.Message != "O e-mail digitado nao confere com a conta." {
		t.Fatalf("expected mismatch message, got %q", deletePayload.Message)
	}
	if len(deletePayload.Details) != 1 || deletePayload.Details[0].Field != "email" {
		t.Fatalf("expected email field detail, got %+v", deletePayload.Details)
	}

	meResponse := performJSONRequest(t, router, http.MethodGet, "/v1/auth/me", authPayload.Token, nil)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("expected account to remain active after invalid confirmation, got %d", meResponse.Code)
	}
}

func TestRegisterDeviceToken(t *testing.T) {
	router, _, _, store, _ := newTestRouterWithDeps(t)

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/v1/auth/register", "", map[string]any{
		"name":           "Kaleb",
		"email":          "kaleb-device-token@example.com",
		"password":       "Senha123",
		"accepted_terms": true,
	})
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResponse.Code)
	}

	var registerPayload struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	decodeBody(t, registerResponse, &registerPayload)

	deviceToken := strings.Repeat("a", 32)
	registerTokenResponse := performJSONRequest(
		t,
		router,
		http.MethodPost,
		"/v1/notifications/device-token",
		registerPayload.Token,
		map[string]any{
			"token":    deviceToken,
			"platform": "android",
		},
	)
	if registerTokenResponse.Code != http.StatusCreated {
		t.Fatalf("expected register device token status 201, got %d", registerTokenResponse.Code)
	}

	tokens, err := store.PushDeviceTokens().ListByUserID(context.Background(), registerPayload.User.ID)
	if err != nil {
		t.Fatalf("list user device tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 device token, got %d", len(tokens))
	}
	if tokens[0].Token != deviceToken {
		t.Fatalf("expected token %q, got %q", deviceToken, tokens[0].Token)
	}
	if tokens[0].Platform != "android" {
		t.Fatalf("expected platform android, got %q", tokens[0].Platform)
	}
}

func newTestRouter(t *testing.T) http.Handler {
	router, _, _, _, _ := newTestRouterWithDeps(t)
	return router
}

func newTestRouterWithDeps(t *testing.T) (http.Handler, *capturePasswordResetSender, *captureRegistrationSender, *memory.Store, string) {
	t.Helper()

	uploadDir := t.TempDir()

	cfg := config.Config{
		AppEnv:             "test",
		AppPort:            "8080",
		AppBaseURL:         "http://localhost:8080",
		FrontendURL:        "http://localhost:3000",
		JWTSecret:          "test-secret",
		JWTExpiresIn:       time.Hour,
		CORSAllowedOrigins: []string{"http://localhost:3000"},
		LocalUploadDir:     uploadDir,
		UploadMaxSizeBytes: 10 << 20,
	}

	localStorage, err := storage.NewLocalStorage(uploadDir, "http://localhost:8080/uploads")
	if err != nil {
		t.Fatalf("create local storage: %v", err)
	}

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiresIn)
	store := memory.NewStore()
	passwordResetSender := &capturePasswordResetSender{}
	registrationSender := &captureRegistrationSender{}
	organizerPushSender := &captureOrganizerPushSender{}
	organizerNotificationService := services.NewOrganizerNotificationService(store.PushDeviceTokens(), organizerPushSender, store.Notifications())

	authService := services.NewAuthService(
		store.Users(),
		store.PasswordResetTokens(),
		jwtManager,
		time.Hour,
		"http://localhost:3000/redefinir-senha",
		passwordResetSender,
		registrationSender,
	)
	eventService := services.NewEventService(store.Events())
	guestService := services.NewGuestService(store.Events(), store.Guests())
	rsvpService := services.NewRSVPService(store.Events(), store.Guests(), store.RSVPs(), 0, organizerNotificationService)
	checkInService := services.NewCheckInService(store.Events(), store.Guests(), store.RSVPs())
	giftService := services.NewGiftService(store.Events(), store.Gifts())
	giftTransactionService := services.NewGiftTransactionService(store.Events(), store.Gifts(), store.GiftTransactions(), time.Hour, organizerNotificationService)
	dashboardService := services.NewDashboardService(store.Users(), store.Events(), store.Guests(), store.Gifts())
	uploadService := services.NewUploadService(localStorage, cfg.UploadMaxSizeBytes)
	accountService := services.NewAccountService(store.Users(), store.Events(), store.Gifts(), uploadService)

	return NewRouter(cfg, nil, localStorage, jwtManager, authService, accountService, eventService, guestService, rsvpService, checkInService, giftService, giftTransactionService, dashboardService, uploadService, nil, organizerNotificationService, store.Users()), passwordResetSender, registrationSender, store, uploadDir
}

func performJSONRequest(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		requestBody = bytes.NewReader(payload)
	}

	request := httptest.NewRequest(method, path, requestBody)
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeBody(t *testing.T, response *httptest.ResponseRecorder, dst any) {
	t.Helper()

	if err := json.Unmarshal(response.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func performMultipartRequest(t *testing.T, router http.Handler, path, token, folder, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("folder", folder); err != nil {
		t.Fatalf("write folder field: %v", err)
	}

	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, err := io.Copy(fileWriter, bytes.NewReader(content)); err != nil {
		t.Fatalf("write file content: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, path, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

var samplePNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x03, 0x01, 0x01, 0x00, 0xc9, 0xfe, 0x92,
	0xef, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
	0x44, 0xae, 0x42, 0x60, 0x82,
}

type capturePasswordResetSender struct {
	messages []mailer.PasswordResetMessage
}

func (s *capturePasswordResetSender) SendPasswordReset(_ context.Context, message mailer.PasswordResetMessage) error {
	s.messages = append(s.messages, message)
	return nil
}

type captureRegistrationSender struct {
	messages []notifier.NewRegistrationMessage
}

func (s *captureRegistrationSender) SendNewRegistration(_ context.Context, message notifier.NewRegistrationMessage) error {
	s.messages = append(s.messages, message)
	return nil
}

type captureOrganizerPushSender struct{}

func (captureOrganizerPushSender) SendToDevice(_ context.Context, _ string, _ notifier.OrganizerPushMessage) error {
	return nil
}
