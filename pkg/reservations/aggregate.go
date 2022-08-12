package reservations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/fsm"
)

func init() {
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return NewReservationAggregate(id)
	})
}

const ReservationAggregateType eh.AggregateType = "Reservation"

var _ = eh.Aggregate(&ReservationAggregate{})

// ReservationAggregate is the write-model, and is used with event sourcing
// It uses a Finite State Machine (FSM) to determine the state when events are applied
type ReservationAggregate struct {
	*events.AggregateBase

	name      string
	startTime time.Time
	endTime   time.Time
	user      string

	created bool

	state *fsm.FSM

	err error
}

// NewReservationAggregate returns an initiliazed ReservationAggregate, this should alkways be used to create the aggregate
func NewReservationAggregate(id uuid.UUID) *ReservationAggregate {
	return &ReservationAggregate{
		AggregateBase: events.NewAggregateBase(ReservationAggregateType, id),
		state: fsm.NewFSM("pending",
			fsm.Events{
				{Name: "confirmed", Src: []string{"pending"}, Dst: "confirmed"},
				{Name: "declined", Src: []string{"pending"}, Dst: "declined"},
				{Name: "cancelled", Src: []string{"pending", "confirmed"}, Dst: "cancelled"},
				{Name: "changed", Src: []string{"pending", "declined", "cancelled", "confirmed"}, Dst: "pending"},
			},
			fsm.Callbacks{},
		),
	}
}

// HandleCommand is called whenever the commandBus recieves a command for which this aggregate is registered
func (r *ReservationAggregate) HandleCommand(ctx context.Context, cmd eh.Command) error {
	switch cmd := cmd.(type) {
	case *CreateReservation:
		if !r.created {
			r.AppendEvent(ReservationCreatedEvent, &ReservationCreatedData{
				RoomID:    cmd.RoomID,
				Name:      cmd.Name,
				User:      cmd.User,
				StartTime: cmd.StartTime,
				EndTime:   cmd.EndTime,
			}, time.Now())
		} else {
			// TODO if table already reserved, raise a conflict event
			// TODO error
		}
	case *ConfirmReservation:
		if r.state.Is("pending") {
			r.AppendEvent(ReservationConfirmedEvent, &ReservationConfirmedData{
				User: cmd.User,
			}, time.Now())
		} else {
			// TODO error
		}
	case *DeclineReservation:
		if r.state.Is("pending") {
			r.AppendEvent(ReservationDeclinedEvent, &ReservationDeclinedData{
				User:    cmd.User,
				Message: cmd.Message,
			}, time.Now())
		} else {
			// TODO error
		}
	case *CancelReservation:
		if r.state.Is("pending") || r.state.Is("confirmed") {
			r.AppendEvent(ReservationCancelledEvent, &ReservationCancelledData{
				User: cmd.User,
			}, time.Now())
		} else {
			// TODO error
		}
	case *ChangeReservationTime:
		if cmd.EndTime.After(cmd.StartTime) {
			r.AppendEvent(ReservationTimeChangedEvent, &ReservationTimeChangeData{
				User:      cmd.User,
				StartTime: cmd.StartTime,
				EndTime:   cmd.EndTime,
			}, time.Now())
		} else {
			// TODO error
		}

	}
	return nil
}

// ApplyEvent is called whenever an event is recieved on the eventBus
func (r *ReservationAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	fmt.Println("Recieved event")
	switch event.EventType() {
	case ReservationCreatedEvent:
		r.created = true
		if data, ok := event.Data().(*ReservationCreatedData); ok {
			r.name = data.Name
			r.startTime = data.StartTime
			r.endTime = data.EndTime
			r.user = data.User
		}
	case ReservationConfirmedEvent:
		r.state.Event("confirmed")
	case ReservationDeclinedEvent:
		r.state.Event("declined")
	case ReservationTimeChangedEvent:
		r.state.Event("changed")
		if data, ok := event.Data().(*ReservationTimeChangeData); ok {
			r.startTime = data.StartTime
			r.endTime = data.EndTime
		}
	case ReservationCancelledEvent:
		r.state.Event("cancelled")
	case ReservationBookingConflictedEvent:
		r.err = errors.New("Room already booked")
	}

	return nil
}
