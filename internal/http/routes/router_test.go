package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"myevent-back/internal/auth"
	"myevent-back/internal/config"
	"myevent-back/internal/repositories/memory"
	"myevent-back/internal/services"
	"myevent-back/internal/storage"
)

func TestPhaseOneFlow(t *testing.T) {
	router := newTestRouter(t)

	registerBody := map[string]any{
		"name":     "Kaleb",
		"email":    "kaleb@example.com",
		"password": "12345678",
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
	if eventPayload.Status != "draft" {
		t.Fatalf("expected draft status, got %s", eventPayload.Status)
	}

	updateEventResponse := performJSONRequest(t, router, http.MethodPatch, "/v1/events/"+eventPayload.ID, authPayload.Token, map[string]any{
		"description": "Nosso grande dia",
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

	var guestsPayload []map[string]any
	decodeBody(t, listGuestsResponse, &guestsPayload)
	if len(guestsPayload) != 1 {
		t.Fatalf("expected 1 guest, got %d", len(guestsPayload))
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

	var rsvpsPayload []map[string]any
	decodeBody(t, listRSVPsResponse, &rsvpsPayload)
	if len(rsvpsPayload) != 1 {
		t.Fatalf("expected 1 RSVP, got %d", len(rsvpsPayload))
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
		"name":     "Kaleb",
		"email":    "kaleb-phase3@example.com",
		"password": "12345678",
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
	if duplicateCheckInResponse.Code != http.StatusConflict {
		t.Fatalf("expected duplicate check-in status 409, got %d", duplicateCheckInResponse.Code)
	}

	listCheckInGuestsResponse := performJSONRequest(t, router, http.MethodGet, "/v1/events/"+eventPayload.ID+"/checkin/guests", authPayload.Token, nil)
	if listCheckInGuestsResponse.Code != http.StatusOK {
		t.Fatalf("expected check-in guests status 200, got %d", listCheckInGuestsResponse.Code)
	}

	var checkInGuestsPayload []map[string]any
	decodeBody(t, listCheckInGuestsResponse, &checkInGuestsPayload)
	if len(checkInGuestsPayload) != 1 {
		t.Fatalf("expected 1 guest in check-in list, got %d", len(checkInGuestsPayload))
	}
	if checkInGuestsPayload[0]["checked_in_at"] == nil {
		t.Fatalf("expected checked_in_at in guest list payload, got %+v", checkInGuestsPayload[0])
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

	var publicGiftsPayload []map[string]any
	decodeBody(t, publicGiftsResponse, &publicGiftsPayload)
	if len(publicGiftsPayload) != 2 {
		t.Fatalf("expected 2 public gifts, got %d", len(publicGiftsPayload))
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

	var transactionsPayload []map[string]any
	decodeBody(t, listTransactionsResponse, &transactionsPayload)
	if len(transactionsPayload) != 1 {
		t.Fatalf("expected 1 gift transaction, got %d", len(transactionsPayload))
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
		"name":     "Kaleb",
		"email":    "kaleb-phase4@example.com",
		"password": "12345678",
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

func newTestRouter(t *testing.T) http.Handler {
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

	authService := services.NewAuthService(store.Users(), jwtManager)
	eventService := services.NewEventService(store.Events())
	guestService := services.NewGuestService(store.Events(), store.Guests())
	rsvpService := services.NewRSVPService(store.Events(), store.Guests(), store.RSVPs())
	checkInService := services.NewCheckInService(store.Events(), store.Guests())
	giftService := services.NewGiftService(store.Events(), store.Gifts())
	giftTransactionService := services.NewGiftTransactionService(store.Events(), store.Gifts(), store.GiftTransactions())
	dashboardService := services.NewDashboardService(store.Events(), store.Guests(), store.Gifts())
	uploadService := services.NewUploadService(localStorage, cfg.UploadMaxSizeBytes)

	return NewRouter(cfg, jwtManager, authService, eventService, guestService, rsvpService, checkInService, giftService, giftTransactionService, dashboardService, uploadService)
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
