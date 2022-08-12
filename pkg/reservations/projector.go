package reservations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/projector"
)

type ReservationStatus string

const (
	StatusPending   ReservationStatus = "pending"
	StatusDeclined  ReservationStatus = "declined"
	StatusConfirmed ReservationStatus = "confirmed"
	StatusCancelled ReservationStatus = "cancelled"
)

// Reservation is the read-model
type Reservation struct {
	ID        uuid.UUID
	Version   int
	Name      string
	Creator   string
	RoomID    int
	StartTime time.Time
	EndTime   time.Time
	Status    ReservationStatus
}

func (r *Reservation) EntityID() uuid.UUID {
	return r.ID
}

func (r *Reservation) AggregateVersion() int {
	return r.Version
}

// ReservationProjector is the projector for the read-model
type ReservationProjector struct{}

func NewReservationProjector() *ReservationProjector {
	return &ReservationProjector{}
}

func (p *ReservationProjector) ProjectorType() projector.Type {
	return projector.Type(ReservationAggregateType.String())
}

// Project is called each time an event related to a specific AggregateID come from the eventBus
// A new ReservationProjector is generated per AggregateID
func (p *ReservationProjector) Project(ctx context.Context, event eh.Event, entity eh.Entity) (eh.Entity, error) {
	r, ok := entity.(*Reservation)
	if !ok {
		return nil, errors.New("model is of incorrect type")
	}

	switch event.EventType() {
	case ReservationCreatedEvent:
		data, ok := event.Data().(*ReservationCreatedData)
		if !ok {
			return nil, fmt.Errorf("projector: invalid event data type: %v", event.Data())
		}
		r.ID = event.AggregateID()
		r.Name = data.Name
		r.Creator = data.User
		r.StartTime = data.StartTime
		r.EndTime = data.EndTime
		r.Status = StatusPending
		r.RoomID = data.RoomID
	case ReservationConfirmedEvent:
		r.Status = StatusConfirmed
	case ReservationDeclinedEvent:
		r.Status = StatusDeclined
	case ReservationTimeChangedEvent:
		data, ok := event.Data().(*ReservationTimeChangeData)
		if !ok {
			return nil, fmt.Errorf("projector: invalid event data type: %v", event.Data())
		}
		r.Status = StatusPending
		r.StartTime = data.StartTime
		r.EndTime = data.EndTime
	case ReservationCancelledEvent:
		r.Status = StatusCancelled
	case ReservationBookingConflictedEvent:
		r.Status = StatusDeclined
	default:
		return nil, fmt.Errorf("could not handle event: %s", event)
	}
	r.Version++
	return r, nil
}
