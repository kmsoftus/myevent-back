package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"myevent-back/internal/dto"
	"myevent-back/internal/models"
	"myevent-back/internal/repositories/memory"
)

func TestEventServiceListByUserFiltersByQueryAndStatus(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewEventService(store.Events())

	now := time.Now().UTC()
	seedEvent(t, store, &models.Event{
		ID:        "event-1",
		UserID:    "user-1",
		Title:     "Casamento na Praia",
		Slug:      "casamento-na-praia",
		Status:    "published",
		Date:      "2026-09-20",
		CreatedAt: now.Add(-4 * time.Hour),
		UpdatedAt: now.Add(-4 * time.Hour),
	})
	seedEvent(t, store, &models.Event{
		ID:        "event-2",
		UserID:    "user-1",
		Title:     "Casamento Civil",
		Slug:      "casamento-civil",
		Status:    "draft",
		Date:      "2026-10-12",
		CreatedAt: now.Add(-3 * time.Hour),
		UpdatedAt: now.Add(-3 * time.Hour),
	})
	seedEvent(t, store, &models.Event{
		ID:        "event-3",
		UserID:    "user-1",
		Title:     "Cha de Bebe",
		Slug:      "cha-de-bebe",
		Status:    "published",
		Date:      "2026-11-02",
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
	})

	events, err := service.ListByUser(ctx, "user-1", dto.ListEventsRequest{
		Query:  "casamento",
		Status: "published",
		Sort:   "date_desc",
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "event-1" {
		t.Fatalf("expected event-1, got %s", events[0].ID)
	}
}

func TestEventServiceListByUserSortsByDate(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewEventService(store.Events())

	now := time.Now().UTC()
	seedEvent(t, store, &models.Event{
		ID:        "event-1",
		UserID:    "user-1",
		Title:     "Evento A",
		Slug:      "evento-a",
		Status:    "published",
		Date:      "2026-08-20",
		CreatedAt: now.Add(-3 * time.Hour),
		UpdatedAt: now.Add(-3 * time.Hour),
	})
	seedEvent(t, store, &models.Event{
		ID:        "event-2",
		UserID:    "user-1",
		Title:     "Evento B",
		Slug:      "evento-b",
		Status:    "published",
		Date:      "2026-05-01",
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
	})
	seedEvent(t, store, &models.Event{
		ID:        "event-3",
		UserID:    "user-1",
		Title:     "Evento C",
		Slug:      "evento-c",
		Status:    "published",
		Date:      "2026-12-15",
		CreatedAt: now.Add(-1 * time.Hour),
		UpdatedAt: now.Add(-1 * time.Hour),
	})

	eventsAsc, err := service.ListByUser(ctx, "user-1", dto.ListEventsRequest{
		Sort: "date_asc",
	})
	if err != nil {
		t.Fatalf("list events asc: %v", err)
	}
	if len(eventsAsc) != 3 {
		t.Fatalf("expected 3 events in asc sort, got %d", len(eventsAsc))
	}
	if eventsAsc[0].ID != "event-2" || eventsAsc[1].ID != "event-1" || eventsAsc[2].ID != "event-3" {
		t.Fatalf("unexpected asc order: %s, %s, %s", eventsAsc[0].ID, eventsAsc[1].ID, eventsAsc[2].ID)
	}

	eventsDesc, err := service.ListByUser(ctx, "user-1", dto.ListEventsRequest{
		Sort: "date_desc",
	})
	if err != nil {
		t.Fatalf("list events desc: %v", err)
	}
	if len(eventsDesc) != 3 {
		t.Fatalf("expected 3 events in desc sort, got %d", len(eventsDesc))
	}
	if eventsDesc[0].ID != "event-3" || eventsDesc[1].ID != "event-1" || eventsDesc[2].ID != "event-2" {
		t.Fatalf("unexpected desc order: %s, %s, %s", eventsDesc[0].ID, eventsDesc[1].ID, eventsDesc[2].ID)
	}
}

func TestEventServiceListByUserRejectsInvalidStatus(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewEventService(store.Events())

	_, err := service.ListByUser(ctx, "user-1", dto.ListEventsRequest{
		Status: "invalid",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestEventServiceListByUserRejectsInvalidSort(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := NewEventService(store.Events())

	_, err := service.ListByUser(ctx, "user-1", dto.ListEventsRequest{
		Sort: "created_at_desc",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func seedEvent(t *testing.T, store *memory.Store, event *models.Event) {
	t.Helper()
	if err := store.Events().Create(context.Background(), event); err != nil {
		t.Fatalf("create event %s: %v", event.ID, err)
	}
}
