package reservations

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
)

const ReservationConflictSagaType saga.Type = "ReservationConflictSaga"

type reservation struct {
	startTime time.Time
	endTime   time.Time
}

type room struct {
	reservations map[uuid.UUID]*reservation
}

// ReservationConflictSaga contains the business logic to Confirm/Decline reservations based on clashes
// WARNING not persisten, will not retain state after program restart
// This is a prototype after all...
type ReservationConflictSaga struct {
	reservedRooms   map[int]room
	reservedRoomsMu sync.RWMutex
}

func NewReservationConflictSaga() *ReservationConflictSaga {
	r := &ReservationConflictSaga{
		reservedRooms:   make(map[int]room),
		reservedRoomsMu: sync.RWMutex{},
	}
	for i := 1; i < 7; i++ {
		r.reservedRooms[i] = room{
			reservations: make(map[uuid.UUID]*reservation),
		}
	}
	return r
}

func (r *ReservationConflictSaga) SagaType() saga.Type {
	return ReservationConflictSagaType
}

// RunSaga recieves events and takes actions depending on business rules
// There are lots of bugs here, so don't look too closely!
func (r *ReservationConflictSaga) RunSaga(ctx context.Context, event eh.Event, h eh.CommandHandler) error {
	switch event.EventType() {
	case ReservationCreatedEvent:
		if data, ok := event.Data().(*ReservationCreatedData); ok {
			r.reservedRoomsMu.RLock()
			rroom, ok := r.reservedRooms[data.RoomID]
			if !ok {
				h.HandleCommand(ctx, &DeclineReservation{
					ID:      event.AggregateID(),
					User:    "Scheduler",
					Message: "Room does not exist.",
				})
			}

			// Already exists
			_, ok = rroom.reservations[event.AggregateID()]
			if ok {
				// TODO probably want to check stuff here
				r.reservedRoomsMu.RUnlock()
				return nil
			}
			r.reservedRoomsMu.RUnlock()
			r.reservedRoomsMu.Lock()
			rroom, ok = r.reservedRooms[data.RoomID]
			if !ok {
				h.HandleCommand(ctx, &DeclineReservation{
					ID:      event.AggregateID(),
					User:    "Scheduler",
					Message: "Room does not exist.",
				})
			}

			var clash bool
			for _, res := range rroom.reservations {
				if timeIntersect(res.startTime, res.endTime, data.StartTime, data.EndTime) {
					// The reservation clashes
					h.HandleCommand(ctx, &DeclineReservation{
						ID:      event.AggregateID(),
						User:    "Scheduler",
						Message: "Room occupied.",
					})
					clash = true
					break
				}
			}

			if !clash {
				r.reservedRooms[data.RoomID].reservations[event.AggregateID()] = &reservation{
					startTime: data.StartTime,
					endTime:   data.EndTime,
				}
				h.HandleCommand(ctx, &ConfirmReservation{
					ID:   event.AggregateID(),
					User: "Scheduler",
				})
			}
			r.reservedRoomsMu.Unlock()
		}
	case ReservationTimeChangedEvent:
		if data, ok := event.Data().(*ReservationTimeChangeData); ok {
			r.reservedRoomsMu.Lock()
			var roomNumber int
			for roomno, rrom := range r.reservedRooms {
				if _, ok := rrom.reservations[event.AggregateID()]; ok {
					delete(r.reservedRooms[roomno].reservations, event.AggregateID())
					roomNumber = roomno
					break
				}
			}
			var clash bool
			for _, res := range r.reservedRooms[roomNumber].reservations {
				if timeIntersect(res.startTime, res.endTime, data.StartTime, data.EndTime) {
					// The reservation clashes
					h.HandleCommand(ctx, &DeclineReservation{
						ID:      event.AggregateID(),
						User:    "Scheduler",
						Message: "Room occupied",
					})
					clash = true
					break
				}
			}

			if !clash {
				r.reservedRooms[roomNumber].reservations[event.AggregateID()] = &reservation{
					startTime: data.StartTime,
					endTime:   data.EndTime,
				}
				h.HandleCommand(ctx, &ConfirmReservation{
					ID:   event.AggregateID(),
					User: "Scheduler",
				})
			}
			r.reservedRoomsMu.Unlock()
		}
	case ReservationCancelledEvent:
		r.reservedRoomsMu.Lock()
		for roomno, rrom := range r.reservedRooms {
			if _, ok := rrom.reservations[event.AggregateID()]; ok {
				delete(r.reservedRooms[roomno].reservations, event.AggregateID())
				break
			}
		}
		r.reservedRoomsMu.Unlock()

	}
	return nil
}

func afterEquals(time1, time2 time.Time) bool {
	return time1.After(time2) || time1.Equal(time2)
}

func timeIntersect(start1, end1, start2, end2 time.Time) bool {
	return (afterEquals(start2, start1) && start2.Before(end1)) ||
		(afterEquals(start1, start2) && start1.Before(end2))
}
