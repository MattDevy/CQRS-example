package reservations

import (
	"context"
	"log"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	"github.com/looplab/eventhorizon/eventhandler/projector"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/looplab/eventhorizon/repo/memory"
	"github.com/looplab/eventhorizon/repo/mongodb"
)

// Setup will initialize and register all the commands, events, aggregates, projectors and sagas
func Setup(
	ctx context.Context,
	eventStore eh.EventStore,
	eventBus eh.EventBus,
	commandBus *bus.CommandHandler,
	reservationRepo eh.ReadWriteRepo,
) {

	// Set the EntityFactories for any memory or mongo repos
	if memoryRepo := memory.IntoRepo(ctx, reservationRepo); memoryRepo != nil {
		memoryRepo.SetEntityFactory(func() eh.Entity { return &Reservation{} })
	}
	if mongoRepo := mongodb.IntoRepo(ctx, reservationRepo); mongoRepo != nil {
		mongoRepo.SetEntityFactory(func() eh.Entity { return &Reservation{} })
	}

	// Register the projector with the eventBus
	reservationProjector := projector.NewEventHandler(NewReservationProjector(), reservationRepo)
	reservationProjector.SetEntityFactory(func() eh.Entity { return &Reservation{} })
	eventBus.AddHandler(ctx, eh.MatchEvents{
		ReservationCreatedEvent,
		ReservationConfirmedEvent,
		ReservationDeclinedEvent,
		ReservationTimeChangedEvent,
		ReservationCancelledEvent,
		ReservationBookingConflictedEvent,
	}, reservationProjector)

	// Create aggregate store
	aggregateStore, err := events.NewAggregateStore(eventStore)
	if err != nil {
		log.Fatalf("could not create aggregate store: %v", err)
	}

	// Register aggregate type and command handler
	commandHandler, err := aggregate.NewCommandHandler(ReservationAggregateType, aggregateStore)
	if err != nil {
		log.Fatalf("could not create command handler: %s", err)
	}

	// Handle specific commands
	commands := []eh.CommandType{
		CreateReservationCommand,
		ConfirmReservationCommand,
		DeclineReservationCommand,
		ChangeReservationTimeCommand,
		CancelReservationCommand,
	}
	for _, cmdType := range commands {
		if err := commandBus.SetHandler(commandHandler, cmdType); err != nil {
			log.Fatalf("could not set command handler: %v", err)
		}
	}

	// Add saga handler to automatically accept / decline reservations
	conflictSaga := saga.NewEventHandler(NewReservationConflictSaga(), commandBus)
	eventBus.AddHandler(ctx, eh.MatchEvents{
		ReservationCreatedEvent,
		ReservationTimeChangedEvent,
		ReservationCancelledEvent,
	}, conflictSaga)
}
