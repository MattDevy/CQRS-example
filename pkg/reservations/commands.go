package reservations

import (
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

func init() {
	eh.RegisterCommand(func() eh.Command { return &CreateReservation{} })
	eh.RegisterCommand(func() eh.Command { return &ConfirmReservation{} })
	eh.RegisterCommand(func() eh.Command { return &DeclineReservation{} })
	eh.RegisterCommand(func() eh.Command { return &ChangeReservationTime{} })
	eh.RegisterCommand(func() eh.Command { return &CancelReservation{} })
}

const (
	CreateReservationCommand     eh.CommandType = "CreateReservation"
	ConfirmReservationCommand    eh.CommandType = "ConfirmReservation"
	DeclineReservationCommand    eh.CommandType = "DeclineReservation"
	ChangeReservationTimeCommand eh.CommandType = "ChangeReservationTime"
	CancelReservationCommand     eh.CommandType = "CancelReservation"
)

// CreateReservation is the command to create a reservation
// It contains all the information needed to create a reservation, no field can be empty
type CreateReservation struct {
	ID        uuid.UUID
	Name      string
	User      string
	RoomID    int
	StartTime time.Time
	EndTime   time.Time
}

func (c CreateReservation) AggregateID() uuid.UUID          { return c.ID }
func (c CreateReservation) AggregateType() eh.AggregateType { return ReservationAggregateType }
func (c CreateReservation) CommandType() eh.CommandType     { return CreateReservationCommand }

// Confirm is the command to confirm a reservation
// It contains all the information needed to confirm a reservation, no field can be empty
type ConfirmReservation struct {
	ID   uuid.UUID
	User string
}

func (c ConfirmReservation) AggregateID() uuid.UUID          { return c.ID }
func (c ConfirmReservation) AggregateType() eh.AggregateType { return ReservationAggregateType }
func (c ConfirmReservation) CommandType() eh.CommandType     { return ConfirmReservationCommand }

// DeclineReservation is the command to decline a reservation
// It contains all the information needed to decline a reservation, no field can be empty
type DeclineReservation struct {
	ID      uuid.UUID
	User    string
	Message string
}

func (d DeclineReservation) AggregateID() uuid.UUID          { return d.ID }
func (d DeclineReservation) AggregateType() eh.AggregateType { return ReservationAggregateType }
func (d DeclineReservation) CommandType() eh.CommandType     { return DeclineReservationCommand }

// ChangeReservationTime is the command to change the time of a reservation
// It contains all the information needed to change a reservation time, no field can be empty
type ChangeReservationTime struct {
	ID        uuid.UUID
	User      string
	StartTime time.Time
	EndTime   time.Time
}

func (c ChangeReservationTime) AggregateID() uuid.UUID          { return c.ID }
func (c ChangeReservationTime) AggregateType() eh.AggregateType { return ReservationAggregateType }
func (c ChangeReservationTime) CommandType() eh.CommandType     { return ChangeReservationTimeCommand }

// CancelReservation is the command to cancel a reservation
// It contains all the information needed to cancel a reservation, no field can be empty
type CancelReservation struct {
	ID   uuid.UUID
	User string
}

func (c CancelReservation) AggregateID() uuid.UUID          { return c.ID }
func (c CancelReservation) AggregateType() eh.AggregateType { return ReservationAggregateType }
func (c CancelReservation) CommandType() eh.CommandType     { return CancelReservationCommand }
