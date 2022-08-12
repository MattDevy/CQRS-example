package billing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/MattDevy/CQRS-example/pkg/reservations"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

const (
	PricePerMinute float32 = 0.04
)

type Bill struct {
	ID      uuid.UUID
	Version int
	Minutes int
	Total   float32
}

type BillingHistory struct {
	ID           uuid.UUID
	Version      int
	User         string
	Bills        map[string]*Bill
	TotalMinutes int
	TotalPaid    float32
}

func (b *BillingHistory) EntityID() uuid.UUID {
	return b.ID
}

func (b *BillingHistory) AggregateVersion() int {
	return b.Version
}

// BillingHistoryProjector is the read model for a user's Billing History
type BillingHistoryProjector struct {
	Pending            map[uuid.UUID]*reservations.ReservationCreatedData
	UserBillingHistory map[string]uuid.UUID
	ReservationsUser   map[uuid.UUID]string
	repo               eh.ReadWriteRepo
	repoMu             sync.Mutex
}

// NewBillingHistoryProjector initializes a new BillingHistoryProjector, this method should be used
// when creating a new BillingHistoryProjector
func NewBillingHistoryProjector(repo eh.ReadWriteRepo) *BillingHistoryProjector {
	return &BillingHistoryProjector{
		Pending:            make(map[uuid.UUID]*reservations.ReservationCreatedData),
		UserBillingHistory: make(map[string]uuid.UUID),
		ReservationsUser:   make(map[uuid.UUID]string),
		repo:               repo,
	}
}

// HandlerType returns the EventHandlerType of the Projector
func (b *BillingHistoryProjector) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("BillingHistory")
}

// HandleEvent is the method that is called when any events that are registered are recieved from the event bus
func (b *BillingHistoryProjector) HandleEvent(ctx context.Context, event eh.Event) error {
	// One big hack
	b.repoMu.Lock()
	defer b.repoMu.Unlock()

	// Initialize the BillingHistory on fir
	var h *BillingHistory
	if event.EventType() == reservations.ReservationCreatedEvent {
		data, ok := event.Data().(*reservations.ReservationCreatedData)
		if !ok {
			return fmt.Errorf("projector: invalid event data type: %v", event.Data())
		}
		b.ReservationsUser[event.AggregateID()] = data.User
		b.UserBillingHistory[data.User] = uuid.New()
	}

	user, ok := b.ReservationsUser[event.AggregateID()]
	if !ok {
		return errors.New("No reservation found")
	}
	id, ok := b.UserBillingHistory[user]
	if !ok {
		return fmt.Errorf("No user %v found\n", user)
	}
	m, err := b.repo.Find(context.Background(), id)
	if errors.Is(err, eh.ErrEntityNotFound) {
		h = &BillingHistory{
			ID:    id,
			User:  user,
			Bills: map[string]*Bill{},
		}
	} else if err != nil {
		return err
	} else {
		var ok bool
		h, ok = m.(*BillingHistory)
		if !ok {
			return errors.New("projector: incorrect entity type")
		}
	}

	switch event.EventType() {
	case reservations.ReservationCreatedEvent:
		data, ok := event.Data().(*reservations.ReservationCreatedData)
		if !ok {
			return fmt.Errorf("projector: invalid event data type: %v", event.Data())
		}
		b.Pending[event.AggregateID()] = data
		h.Version++
	case reservations.ReservationConfirmedEvent:
		pending, ok := b.Pending[event.AggregateID()]
		if ok {
			bill, ok := h.Bills[thisMonth()]
			mins := int(math.Round(pending.EndTime.Sub(pending.StartTime).Minutes()))
			if !ok {
				bill = &Bill{
					ID: uuid.New(),
				}
				h.Bills[thisMonth()] = bill
			}
			bill.Minutes += mins
			bill.Total += float32(mins) * PricePerMinute
			bill.Version++

			h.TotalMinutes += mins
			h.TotalPaid += float32(mins) * PricePerMinute
			h.Version++
		} else {
			return errors.New("Event not found")
		}
	case reservations.ReservationDeclinedEvent:
		delete(b.Pending, event.AggregateID())
	case reservations.ReservationCancelledEvent:
		pending, ok := b.Pending[event.AggregateID()]
		if ok {
			bill, ok := h.Bills[thisMonth()]
			mins := int(math.Round(pending.EndTime.Sub(pending.StartTime).Minutes()))
			if ok {
				bill.Minutes -= mins
				bill.Total -= float32(mins) * PricePerMinute
				bill.Version++

				h.TotalMinutes -= mins
				h.TotalPaid -= float32(mins) * PricePerMinute
				h.Version++
			}
		}
	case reservations.ReservationTimeChangedEvent:
		data, ok := event.Data().(*reservations.ReservationTimeChangeData)
		if !ok {
			return fmt.Errorf("projector: invalid event data type: %v", event.Data())
		}
		pending, ok := b.Pending[event.AggregateID()]
		if ok {
			mins := int(math.Round(pending.EndTime.Sub(pending.StartTime).Minutes()))
			bill, ok := h.Bills[thisMonth()]
			if ok {
				bill.Minutes -= mins
				bill.Total -= float32(mins) * PricePerMinute
				bill.Version++

				h.TotalMinutes -= mins
				h.TotalPaid -= float32(mins) * PricePerMinute
				h.Version++
			}
			pending.StartTime = data.StartTime
			pending.EndTime = data.EndTime
		}
	default:
		return fmt.Errorf("could not handle event: %s", event)
	}
	if err := b.repo.Save(ctx, h); err != nil {
		return fmt.Errorf("projector: could not save: %w", err)
	}
	return nil
}

func thisMonth() string {
	year, month, _ := time.Now().Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).String()
}
