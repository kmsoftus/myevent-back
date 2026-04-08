package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"myevent-back/internal/models"
	"myevent-back/internal/repositories"
)

type Store struct {
	mu sync.RWMutex

	users       map[string]*models.User
	userByEmail map[string]string

	passwordResetTokens         map[string]*models.PasswordResetToken
	passwordResetTokenByHash    map[string]string
	passwordResetTokenIDsByUser map[string]map[string]struct{}
	pushDeviceTokens            map[string]*models.PushDeviceToken
	pushDeviceTokenIDsByUser    map[string]map[string]struct{}

	events      map[string]*models.Event
	eventBySlug map[string]string

	guests          map[string]*models.Guest
	guestIDsByEvent map[string]map[string]struct{}
	guestByInvite   map[string]string
	guestByQRToken  map[string]string

	rsvps          map[string]*models.RSVP
	rsvpByGuest    map[string]string
	rsvpIDsByEvent map[string]map[string]struct{}

	gifts          map[string]*models.Gift
	giftIDsByEvent map[string]map[string]struct{}

	giftTransactions          map[string]*models.GiftTransaction
	giftTransactionIDsByEvent map[string]map[string]struct{}
}

func NewStore() *Store {
	return &Store{
		users:                       make(map[string]*models.User),
		userByEmail:                 make(map[string]string),
		passwordResetTokens:         make(map[string]*models.PasswordResetToken),
		passwordResetTokenByHash:    make(map[string]string),
		passwordResetTokenIDsByUser: make(map[string]map[string]struct{}),
		pushDeviceTokens:            make(map[string]*models.PushDeviceToken),
		pushDeviceTokenIDsByUser:    make(map[string]map[string]struct{}),
		events:                      make(map[string]*models.Event),
		eventBySlug:                 make(map[string]string),
		guests:                      make(map[string]*models.Guest),
		guestIDsByEvent:             make(map[string]map[string]struct{}),
		guestByInvite:               make(map[string]string),
		guestByQRToken:              make(map[string]string),
		rsvps:                       make(map[string]*models.RSVP),
		rsvpByGuest:                 make(map[string]string),
		rsvpIDsByEvent:              make(map[string]map[string]struct{}),
		gifts:                       make(map[string]*models.Gift),
		giftIDsByEvent:              make(map[string]map[string]struct{}),
		giftTransactions:            make(map[string]*models.GiftTransaction),
		giftTransactionIDsByEvent:   make(map[string]map[string]struct{}),
	}
}

func (s *Store) Users() repositories.UserRepository {
	return &userRepository{store: s}
}

func (s *Store) Events() repositories.EventRepository {
	return &eventRepository{store: s}
}

func (s *Store) PasswordResetTokens() repositories.PasswordResetTokenRepository {
	return &passwordResetTokenRepository{store: s}
}

func (s *Store) Guests() repositories.GuestRepository {
	return &guestRepository{store: s}
}

func (s *Store) PushDeviceTokens() repositories.PushDeviceTokenRepository {
	return &pushDeviceTokenRepository{store: s}
}

func (s *Store) RSVPs() repositories.RSVPRepository {
	return &rsvpRepository{store: s}
}

func (s *Store) Gifts() repositories.GiftRepository {
	return &giftRepository{store: s}
}

func (s *Store) GiftTransactions() repositories.GiftTransactionRepository {
	return &giftTransactionRepository{store: s}
}

type userRepository struct {
	store *Store
}

func (r *userRepository) Create(_ context.Context, user *models.User) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	email := strings.ToLower(user.Email)
	if _, exists := r.store.userByEmail[email]; exists {
		return repositories.ErrConflict
	}

	r.store.users[user.ID] = cloneUser(user)
	r.store.userByEmail[email] = user.ID
	return nil
}

func (r *userRepository) GetByID(_ context.Context, id string) (*models.User, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	user, ok := r.store.users[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneUser(user), nil
}

func (r *userRepository) GetByEmail(_ context.Context, email string) (*models.User, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	id, ok := r.store.userByEmail[strings.ToLower(email)]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneUser(r.store.users[id]), nil
}

func (r *userRepository) Delete(_ context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	user, ok := r.store.users[id]
	if !ok {
		return repositories.ErrNotFound
	}

	delete(r.store.users, id)
	delete(r.store.userByEmail, strings.ToLower(user.Email))

	if tokenIDs, ok := r.store.passwordResetTokenIDsByUser[id]; ok {
		for tokenID := range tokenIDs {
			if token, exists := r.store.passwordResetTokens[tokenID]; exists {
				delete(r.store.passwordResetTokenByHash, token.TokenHash)
				delete(r.store.passwordResetTokens, tokenID)
			}
		}
		delete(r.store.passwordResetTokenIDsByUser, id)
	}

	if tokenIDs, ok := r.store.pushDeviceTokenIDsByUser[id]; ok {
		for tokenID := range tokenIDs {
			delete(r.store.pushDeviceTokens, tokenID)
		}
		delete(r.store.pushDeviceTokenIDsByUser, id)
	}

	for eventID, event := range r.store.events {
		if event.UserID == id {
			r.store.deleteEventLocked(eventID)
		}
	}

	return nil
}

func (r *userRepository) UpdateProfile(_ context.Context, id, name, contactPhone, profilePhotoURL string, updatedAt time.Time) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	user, ok := r.store.users[id]
	if !ok {
		return repositories.ErrNotFound
	}

	user.Name = name
	user.ContactPhone = contactPhone
	user.ProfilePhotoURL = profilePhotoURL
	user.UpdatedAt = updatedAt
	return nil
}

func (r *userRepository) UpdatePassword(_ context.Context, id, passwordHash string, updatedAt time.Time) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	user, ok := r.store.users[id]
	if !ok {
		return repositories.ErrNotFound
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = updatedAt
	return nil
}

func (r *passwordResetTokenRepository) Create(_ context.Context, token *models.PasswordResetToken) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, exists := r.store.passwordResetTokens[token.ID]; exists {
		return repositories.ErrConflict
	}
	if _, exists := r.store.passwordResetTokenByHash[token.TokenHash]; exists {
		return repositories.ErrConflict
	}

	r.store.passwordResetTokens[token.ID] = clonePasswordResetToken(token)
	r.store.passwordResetTokenByHash[token.TokenHash] = token.ID

	if _, ok := r.store.passwordResetTokenIDsByUser[token.UserID]; !ok {
		r.store.passwordResetTokenIDsByUser[token.UserID] = make(map[string]struct{})
	}
	r.store.passwordResetTokenIDsByUser[token.UserID][token.ID] = struct{}{}

	return nil
}

func (r *passwordResetTokenRepository) DeleteActiveByUserID(_ context.Context, userID string, now time.Time) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	tokenIDs := r.store.passwordResetTokenIDsByUser[userID]
	for tokenID := range tokenIDs {
		token, ok := r.store.passwordResetTokens[tokenID]
		if !ok {
			delete(tokenIDs, tokenID)
			continue
		}
		if token.UsedAt == nil && token.ExpiresAt.After(now) {
			delete(r.store.passwordResetTokenByHash, token.TokenHash)
			delete(r.store.passwordResetTokens, tokenID)
			delete(tokenIDs, tokenID)
		}
	}

	if len(tokenIDs) == 0 {
		delete(r.store.passwordResetTokenIDsByUser, userID)
	}

	return nil
}

func (r *passwordResetTokenRepository) Consume(_ context.Context, tokenHash string, now time.Time) (*models.PasswordResetToken, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	tokenID, ok := r.store.passwordResetTokenByHash[tokenHash]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	token, ok := r.store.passwordResetTokens[tokenID]
	if !ok {
		delete(r.store.passwordResetTokenByHash, tokenHash)
		return nil, repositories.ErrNotFound
	}

	if token.UsedAt != nil || !token.ExpiresAt.After(now) {
		delete(r.store.passwordResetTokenByHash, tokenHash)
		return nil, repositories.ErrNotFound
	}

	usedAt := now
	token.UsedAt = &usedAt
	delete(r.store.passwordResetTokenByHash, tokenHash)

	return clonePasswordResetToken(token), nil
}

func (r *pushDeviceTokenRepository) Upsert(_ context.Context, token *models.PushDeviceToken) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.users[token.UserID]; !ok {
		return repositories.ErrNotFound
	}

	if existing, ok := r.store.pushDeviceTokens[token.Token]; ok {
		if existing.UserID != token.UserID {
			if tokenIDs := r.store.pushDeviceTokenIDsByUser[existing.UserID]; tokenIDs != nil {
				delete(tokenIDs, existing.Token)
				if len(tokenIDs) == 0 {
					delete(r.store.pushDeviceTokenIDsByUser, existing.UserID)
				}
			}
			existing.CreatedAt = token.CreatedAt
		}

		existing.UserID = token.UserID
		existing.Platform = token.Platform
		existing.UpdatedAt = token.UpdatedAt
		existing.LastSeenAt = token.LastSeenAt

		if _, ok := r.store.pushDeviceTokenIDsByUser[token.UserID]; !ok {
			r.store.pushDeviceTokenIDsByUser[token.UserID] = make(map[string]struct{})
		}
		r.store.pushDeviceTokenIDsByUser[token.UserID][token.Token] = struct{}{}
		return nil
	}

	copy := clonePushDeviceToken(token)
	r.store.pushDeviceTokens[token.Token] = copy
	if _, ok := r.store.pushDeviceTokenIDsByUser[token.UserID]; !ok {
		r.store.pushDeviceTokenIDsByUser[token.UserID] = make(map[string]struct{})
	}
	r.store.pushDeviceTokenIDsByUser[token.UserID][token.Token] = struct{}{}
	return nil
}

func (r *pushDeviceTokenRepository) ListByUserID(_ context.Context, userID string) ([]*models.PushDeviceToken, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	tokenIDs := r.store.pushDeviceTokenIDsByUser[userID]
	tokens := make([]*models.PushDeviceToken, 0, len(tokenIDs))
	for tokenID := range tokenIDs {
		token := r.store.pushDeviceTokens[tokenID]
		if token == nil {
			continue
		}
		tokens = append(tokens, clonePushDeviceToken(token))
	}

	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].UpdatedAt.After(tokens[j].UpdatedAt)
	})

	return tokens, nil
}

func (r *pushDeviceTokenRepository) DeleteByToken(_ context.Context, token string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	current, ok := r.store.pushDeviceTokens[token]
	if !ok {
		return nil
	}

	delete(r.store.pushDeviceTokens, token)
	if tokenIDs := r.store.pushDeviceTokenIDsByUser[current.UserID]; tokenIDs != nil {
		delete(tokenIDs, token)
		if len(tokenIDs) == 0 {
			delete(r.store.pushDeviceTokenIDsByUser, current.UserID)
		}
	}

	return nil
}

func (r *pushDeviceTokenRepository) ListAll(_ context.Context) ([]*models.PushDeviceToken, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	tokens := make([]*models.PushDeviceToken, 0, len(r.store.pushDeviceTokens))
	for _, t := range r.store.pushDeviceTokens {
		clone := *t
		tokens = append(tokens, &clone)
	}
	return tokens, nil
}

type eventRepository struct {
	store *Store
}

type passwordResetTokenRepository struct {
	store *Store
}

type pushDeviceTokenRepository struct {
	store *Store
}

func (r *eventRepository) Create(_ context.Context, event *models.Event) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	slug := strings.ToLower(event.Slug)
	if _, exists := r.store.eventBySlug[slug]; exists {
		return repositories.ErrConflict
	}

	r.store.events[event.ID] = cloneEvent(event)
	r.store.eventBySlug[slug] = event.ID
	return nil
}

func (r *eventRepository) ListByUserID(_ context.Context, userID string) ([]*models.Event, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	events := make([]*models.Event, 0)
	for _, event := range r.store.events {
		if event.UserID == userID {
			events = append(events, cloneEvent(event))
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})

	return events, nil
}

func (r *eventRepository) GetByID(_ context.Context, id string) (*models.Event, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	event, ok := r.store.events[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneEvent(event), nil
}

func (r *eventRepository) GetBySlug(_ context.Context, slug string) (*models.Event, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	id, ok := r.store.eventBySlug[strings.ToLower(slug)]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneEvent(r.store.events[id]), nil
}

func (r *eventRepository) Update(_ context.Context, event *models.Event) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	current, ok := r.store.events[event.ID]
	if !ok {
		return repositories.ErrNotFound
	}

	oldSlug := strings.ToLower(current.Slug)
	newSlug := strings.ToLower(event.Slug)
	if ownerID, exists := r.store.eventBySlug[newSlug]; exists && ownerID != event.ID {
		return repositories.ErrConflict
	}

	delete(r.store.eventBySlug, oldSlug)
	r.store.eventBySlug[newSlug] = event.ID
	r.store.events[event.ID] = cloneEvent(event)
	return nil
}

func (r *eventRepository) Delete(_ context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	return r.store.deleteEventLocked(id)
}

type guestRepository struct {
	store *Store
}

type rsvpRepository struct {
	store *Store
}

type giftRepository struct {
	store *Store
}

type giftTransactionRepository struct {
	store *Store
}

func (r *guestRepository) Create(_ context.Context, guest *models.Guest) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	inviteCode := strings.ToUpper(strings.TrimSpace(guest.InviteCode))
	if _, exists := r.store.guests[guest.ID]; exists {
		return repositories.ErrConflict
	}
	if _, exists := r.store.guestByInvite[inviteCode]; exists {
		return repositories.ErrConflict
	}
	if _, exists := r.store.guestByQRToken[guest.QRCodeToken]; exists {
		return repositories.ErrConflict
	}

	r.store.guests[guest.ID] = cloneGuest(guest)
	r.store.guestByInvite[inviteCode] = guest.ID
	r.store.guestByQRToken[guest.QRCodeToken] = guest.ID

	if _, ok := r.store.guestIDsByEvent[guest.EventID]; !ok {
		r.store.guestIDsByEvent[guest.EventID] = make(map[string]struct{})
	}
	r.store.guestIDsByEvent[guest.EventID][guest.ID] = struct{}{}

	return nil
}

func (r *guestRepository) ListByEventID(_ context.Context, eventID string) ([]*models.Guest, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	guestIDs := r.store.guestIDsByEvent[eventID]
	guests := make([]*models.Guest, 0, len(guestIDs))
	for guestID := range guestIDs {
		guests = append(guests, cloneGuest(r.store.guests[guestID]))
	}

	sort.Slice(guests, func(i, j int) bool {
		return guests[i].CreatedAt.Before(guests[j].CreatedAt)
	})

	return guests, nil
}

func (r *guestRepository) CountByEventID(_ context.Context, eventID string) (int, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	return len(r.store.guestIDsByEvent[eventID]), nil
}

func (r *guestRepository) ListByEventIDPaged(_ context.Context, eventID string, limit, offset int) ([]*models.Guest, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	guestIDs := r.store.guestIDsByEvent[eventID]
	guests := make([]*models.Guest, 0, len(guestIDs))
	for guestID := range guestIDs {
		guests = append(guests, cloneGuest(r.store.guests[guestID]))
	}

	sort.Slice(guests, func(i, j int) bool {
		return guests[i].CreatedAt.Before(guests[j].CreatedAt)
	})

	if offset >= len(guests) {
		return []*models.Guest{}, nil
	}
	end := offset + limit
	if end > len(guests) {
		end = len(guests)
	}
	return guests[offset:end], nil
}

func (r *guestRepository) GetByID(_ context.Context, id string) (*models.Guest, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	guest, ok := r.store.guests[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneGuest(guest), nil
}

func (r *guestRepository) GetByInviteCode(_ context.Context, inviteCode string) (*models.Guest, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	id, ok := r.store.guestByInvite[strings.ToUpper(strings.TrimSpace(inviteCode))]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneGuest(r.store.guests[id]), nil
}

func (r *guestRepository) GetByShortCode(_ context.Context, eventID, shortCode string) (*models.Guest, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	guestIDs := r.store.guestIDsByEvent[eventID]
	for guestID := range guestIDs {
		g := r.store.guests[guestID]
		if g.ShortCode == shortCode {
			return cloneGuest(g), nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (r *guestRepository) SearchByName(_ context.Context, eventID, query string, limit int) ([]*models.Guest, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	queryLower := strings.ToLower(query)
	guestIDs := r.store.guestIDsByEvent[eventID]
	var results []*models.Guest
	for guestID := range guestIDs {
		g := r.store.guests[guestID]
		if strings.Contains(strings.ToLower(g.Name), queryLower) {
			results = append(results, cloneGuest(g))
			if len(results) >= limit {
				break
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results, nil
}

func (r *guestRepository) GetByQRCodeToken(_ context.Context, qrCodeToken string) (*models.Guest, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	id, ok := r.store.guestByQRToken[strings.TrimSpace(qrCodeToken)]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneGuest(r.store.guests[id]), nil
}

func (r *guestRepository) Update(_ context.Context, guest *models.Guest) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	current, ok := r.store.guests[guest.ID]
	if !ok {
		return repositories.ErrNotFound
	}

	inviteCode := strings.ToUpper(strings.TrimSpace(guest.InviteCode))
	if ownerID, exists := r.store.guestByInvite[inviteCode]; exists && ownerID != guest.ID {
		return repositories.ErrConflict
	}
	if ownerID, exists := r.store.guestByQRToken[guest.QRCodeToken]; exists && ownerID != guest.ID {
		return repositories.ErrConflict
	}

	delete(r.store.guestByInvite, strings.ToUpper(strings.TrimSpace(current.InviteCode)))
	delete(r.store.guestByQRToken, current.QRCodeToken)
	r.store.guestByInvite[inviteCode] = guest.ID
	r.store.guestByQRToken[guest.QRCodeToken] = guest.ID
	r.store.guests[guest.ID] = cloneGuest(guest)

	return nil
}

func (r *guestRepository) Delete(_ context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	guest, ok := r.store.guests[id]
	if !ok {
		return repositories.ErrNotFound
	}

	delete(r.store.guests, id)
	delete(r.store.guestByInvite, strings.ToUpper(strings.TrimSpace(guest.InviteCode)))
	delete(r.store.guestByQRToken, guest.QRCodeToken)
	if rsvpID, ok := r.store.rsvpByGuest[id]; ok {
		delete(r.store.rsvpByGuest, id)
		delete(r.store.rsvps, rsvpID)
		if rsvpIDs, ok := r.store.rsvpIDsByEvent[guest.EventID]; ok {
			delete(rsvpIDs, rsvpID)
			if len(rsvpIDs) == 0 {
				delete(r.store.rsvpIDsByEvent, guest.EventID)
			}
		}
	}

	if guestIDs, ok := r.store.guestIDsByEvent[guest.EventID]; ok {
		delete(guestIDs, guest.ID)
		if len(guestIDs) == 0 {
			delete(r.store.guestIDsByEvent, guest.EventID)
		}
	}

	return nil
}

func (r *rsvpRepository) Upsert(_ context.Context, rsvp *models.RSVP) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.guests[rsvp.GuestID]; !ok {
		return repositories.ErrNotFound
	}

	if existingID, ok := r.store.rsvpByGuest[rsvp.GuestID]; ok && existingID != rsvp.ID {
		rsvp.ID = existingID
	}
	if existing, ok := r.store.rsvps[rsvp.ID]; ok {
		rsvp.CreatedAt = existing.CreatedAt
	}

	r.store.rsvps[rsvp.ID] = cloneRSVP(rsvp)
	r.store.rsvpByGuest[rsvp.GuestID] = rsvp.ID

	if _, ok := r.store.rsvpIDsByEvent[rsvp.EventID]; !ok {
		r.store.rsvpIDsByEvent[rsvp.EventID] = make(map[string]struct{})
	}
	r.store.rsvpIDsByEvent[rsvp.EventID][rsvp.ID] = struct{}{}

	return nil
}

func (r *rsvpRepository) GetByGuestID(_ context.Context, guestID string) (*models.RSVP, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	rsvpID, ok := r.store.rsvpByGuest[guestID]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	rsvp, ok := r.store.rsvps[rsvpID]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return cloneRSVP(rsvp), nil
}

func (r *rsvpRepository) ListByEventID(_ context.Context, eventID string) ([]*models.RSVP, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	rsvpIDs := r.store.rsvpIDsByEvent[eventID]
	rsvps := make([]*models.RSVP, 0, len(rsvpIDs))
	for rsvpID := range rsvpIDs {
		rsvps = append(rsvps, cloneRSVP(r.store.rsvps[rsvpID]))
	}

	sort.Slice(rsvps, func(i, j int) bool {
		return rsvps[i].RespondedAt.After(rsvps[j].RespondedAt)
	})

	return rsvps, nil
}

func (r *rsvpRepository) CountByEventID(_ context.Context, eventID string) (int, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	return len(r.store.rsvpIDsByEvent[eventID]), nil
}

func (r *rsvpRepository) ListByEventIDPaged(_ context.Context, eventID string, limit, offset int) ([]*models.RSVP, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	rsvpIDs := r.store.rsvpIDsByEvent[eventID]
	rsvps := make([]*models.RSVP, 0, len(rsvpIDs))
	for rsvpID := range rsvpIDs {
		rsvps = append(rsvps, cloneRSVP(r.store.rsvps[rsvpID]))
	}

	sort.Slice(rsvps, func(i, j int) bool {
		return rsvps[i].RespondedAt.After(rsvps[j].RespondedAt)
	})

	if offset >= len(rsvps) {
		return []*models.RSVP{}, nil
	}
	end := offset + limit
	if end > len(rsvps) {
		end = len(rsvps)
	}
	return rsvps[offset:end], nil
}

func (r *giftRepository) Create(_ context.Context, gift *models.Gift) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, exists := r.store.gifts[gift.ID]; exists {
		return repositories.ErrConflict
	}

	r.store.gifts[gift.ID] = cloneGift(gift)
	if _, ok := r.store.giftIDsByEvent[gift.EventID]; !ok {
		r.store.giftIDsByEvent[gift.EventID] = make(map[string]struct{})
	}
	r.store.giftIDsByEvent[gift.EventID][gift.ID] = struct{}{}

	return nil
}

func (r *giftRepository) ListByEventID(_ context.Context, eventID string) ([]*models.Gift, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	giftIDs := r.store.giftIDsByEvent[eventID]
	gifts := make([]*models.Gift, 0, len(giftIDs))
	for giftID := range giftIDs {
		gifts = append(gifts, cloneGift(r.store.gifts[giftID]))
	}

	sort.Slice(gifts, func(i, j int) bool {
		return gifts[i].CreatedAt.Before(gifts[j].CreatedAt)
	})

	return gifts, nil
}

func (r *giftRepository) CountByEventID(_ context.Context, eventID string) (int, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	return len(r.store.giftIDsByEvent[eventID]), nil
}

func (r *giftRepository) ListByEventIDPaged(_ context.Context, eventID string, limit, offset int) ([]*models.Gift, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	giftIDs := r.store.giftIDsByEvent[eventID]
	gifts := make([]*models.Gift, 0, len(giftIDs))
	for giftID := range giftIDs {
		gifts = append(gifts, cloneGift(r.store.gifts[giftID]))
	}

	sort.Slice(gifts, func(i, j int) bool {
		return gifts[i].CreatedAt.Before(gifts[j].CreatedAt)
	})

	if offset >= len(gifts) {
		return []*models.Gift{}, nil
	}
	end := offset + limit
	if end > len(gifts) {
		end = len(gifts)
	}
	return gifts[offset:end], nil
}

func (r *giftRepository) GetByID(_ context.Context, id string) (*models.Gift, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	gift, ok := r.store.gifts[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneGift(gift), nil
}

func (r *giftRepository) Update(_ context.Context, gift *models.Gift) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.gifts[gift.ID]; !ok {
		return repositories.ErrNotFound
	}

	r.store.gifts[gift.ID] = cloneGift(gift)
	return nil
}

func (r *giftRepository) Delete(_ context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	gift, ok := r.store.gifts[id]
	if !ok {
		return repositories.ErrNotFound
	}

	delete(r.store.gifts, id)
	if giftIDs, ok := r.store.giftIDsByEvent[gift.EventID]; ok {
		delete(giftIDs, gift.ID)
		if len(giftIDs) == 0 {
			delete(r.store.giftIDsByEvent, gift.EventID)
		}
	}

	return nil
}

func (r *giftTransactionRepository) Create(_ context.Context, transaction *models.GiftTransaction) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, exists := r.store.giftTransactions[transaction.ID]; exists {
		return repositories.ErrConflict
	}

	r.store.giftTransactions[transaction.ID] = cloneGiftTransaction(transaction)
	if _, ok := r.store.giftTransactionIDsByEvent[transaction.EventID]; !ok {
		r.store.giftTransactionIDsByEvent[transaction.EventID] = make(map[string]struct{})
	}
	r.store.giftTransactionIDsByEvent[transaction.EventID][transaction.ID] = struct{}{}

	return nil
}

func (r *giftTransactionRepository) ListByEventID(_ context.Context, eventID string) ([]*models.GiftTransaction, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	transactionIDs := r.store.giftTransactionIDsByEvent[eventID]
	transactions := make([]*models.GiftTransaction, 0, len(transactionIDs))
	for transactionID := range transactionIDs {
		transactions = append(transactions, cloneGiftTransaction(r.store.giftTransactions[transactionID]))
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].CreatedAt.After(transactions[j].CreatedAt)
	})

	return transactions, nil
}

func (r *giftTransactionRepository) GetByID(_ context.Context, id string) (*models.GiftTransaction, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	transaction, ok := r.store.giftTransactions[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}

	return cloneGiftTransaction(transaction), nil
}

func (r *giftTransactionRepository) Update(_ context.Context, transaction *models.GiftTransaction) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.giftTransactions[transaction.ID]; !ok {
		return repositories.ErrNotFound
	}

	r.store.giftTransactions[transaction.ID] = cloneGiftTransaction(transaction)
	return nil
}

func cloneUser(user *models.User) *models.User {
	copy := *user
	return &copy
}

func cloneEvent(event *models.Event) *models.Event {
	copy := *event
	return &copy
}

func cloneGuest(guest *models.Guest) *models.Guest {
	copy := *guest
	if guest.CheckedInAt != nil {
		checkedInAt := *guest.CheckedInAt
		copy.CheckedInAt = &checkedInAt
	}
	return &copy
}

func cloneRSVP(rsvp *models.RSVP) *models.RSVP {
	copy := *rsvp
	if rsvp.CompanionNames != nil {
		copy.CompanionNames = append([]string(nil), rsvp.CompanionNames...)
	}
	return &copy
}

func cloneGift(gift *models.Gift) *models.Gift {
	copy := *gift
	if gift.ValueCents != nil {
		value := *gift.ValueCents
		copy.ValueCents = &value
	}
	return &copy
}

func cloneGiftTransaction(transaction *models.GiftTransaction) *models.GiftTransaction {
	copy := *transaction
	if transaction.ConfirmedAt != nil {
		confirmedAt := *transaction.ConfirmedAt
		copy.ConfirmedAt = &confirmedAt
	}
	return &copy
}

func clonePasswordResetToken(token *models.PasswordResetToken) *models.PasswordResetToken {
	copy := *token
	if token.UsedAt != nil {
		usedAt := *token.UsedAt
		copy.UsedAt = &usedAt
	}
	return &copy
}

func clonePushDeviceToken(token *models.PushDeviceToken) *models.PushDeviceToken {
	copy := *token
	return &copy
}

func (s *Store) deleteEventLocked(id string) error {
	event, ok := s.events[id]
	if !ok {
		return repositories.ErrNotFound
	}

	delete(s.events, id)
	delete(s.eventBySlug, strings.ToLower(event.Slug))

	if guestIDs, ok := s.guestIDsByEvent[id]; ok {
		for guestID := range guestIDs {
			guest := s.guests[guestID]
			delete(s.guestByInvite, strings.ToUpper(strings.TrimSpace(guest.InviteCode)))
			delete(s.guestByQRToken, guest.QRCodeToken)
			delete(s.guests, guestID)
			if rsvpID, ok := s.rsvpByGuest[guestID]; ok {
				delete(s.rsvpByGuest, guestID)
				delete(s.rsvps, rsvpID)
				if rsvpIDs, ok := s.rsvpIDsByEvent[id]; ok {
					delete(rsvpIDs, rsvpID)
					if len(rsvpIDs) == 0 {
						delete(s.rsvpIDsByEvent, id)
					}
				}
			}
		}
		delete(s.guestIDsByEvent, id)
	}

	if giftIDs, ok := s.giftIDsByEvent[id]; ok {
		for giftID := range giftIDs {
			delete(s.gifts, giftID)
		}
		delete(s.giftIDsByEvent, id)
	}

	if transactionIDs, ok := s.giftTransactionIDsByEvent[id]; ok {
		for transactionID := range transactionIDs {
			delete(s.giftTransactions, transactionID)
		}
		delete(s.giftTransactionIDsByEvent, id)
	}

	return nil
}
