package reservations

import (
	"time"

	eh "github.com/looplab/eventhorizon"
)

func init() {
	eh.RegisterEventData(ReservationCreatedEvent, func() eh.EventData {
		return &ReservationCreatedData{}
	})
	eh.RegisterEventData(ReservationConfirmedEvent, func() eh.EventData {
		return &ReservationConfirmedData{}
	})
	eh.RegisterEventData(ReservationDeclinedEvent, func() eh.EventData {
		return &ReservationDeclinedData{}
	})
	eh.RegisterEventData(ReservationTimeChangedEvent, func() eh.EventData {
		return &ReservationTimeChangeData{}
	})
	eh.RegisterEventData(ReservationCancelledEvent, func() eh.EventData {
		return &ReservationCancelledData{}
	})
}

const (
	ReservationCreatedEvent           eh.EventType = "ReservationCreated"
	ReservationConfirmedEvent         eh.EventType = "ReservationConfirmed"
	ReservationDeclinedEvent          eh.EventType = "ReservationDeclined"
	ReservationTimeChangedEvent       eh.EventType = "ReservationTimeChanged"
	ReservationCancelledEvent         eh.EventType = "ReservationCancelled"
	ReservationBookingConflictedEvent eh.EventType = "ReservationBookingConflicted"
)

type ReservationCreatedData struct {
	RoomID    int
	Name      string
	User      string
	StartTime time.Time
	EndTime   time.Time
}

type ReservationConfirmedData struct {
	User string
}

type ReservationDeclinedData struct {
	User    string
	Message string
}

type ReservationTimeChangeData struct {
	User      string
	StartTime time.Time
	EndTime   time.Time
}

type ReservationCancelledData struct {
	User string
}
