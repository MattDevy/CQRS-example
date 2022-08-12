package billing

import (
	"context"

	"github.com/MattDevy/CQRS-example/pkg/reservations"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	"github.com/looplab/eventhorizon/repo/memory"
	"github.com/looplab/eventhorizon/repo/mongodb"
)

// Setup will initilize and register all the required billing CQRS commands, events, aggregates, projectors and sagas
func Setup(
	ctx context.Context,
	eventStore eh.EventStore,
	eventBus eh.EventBus,
	commandBus *bus.CommandHandler,
	billingRepo eh.ReadWriteRepo,
) {
	if memoryRepo := memory.IntoRepo(ctx, billingRepo); memoryRepo != nil {
		memoryRepo.SetEntityFactory(func() eh.Entity {
			return &BillingHistory{
				Bills: make(map[string]*Bill),
			}
		})
	}
	if mongoRepo := mongodb.IntoRepo(ctx, billingRepo); mongoRepo != nil {
		mongoRepo.SetEntityFactory(func() eh.Entity {
			return &BillingHistory{
				Bills: make(map[string]*Bill),
			}
		})
	}

	billingProjector := NewBillingHistoryProjector(billingRepo)
	eventBus.AddHandler(ctx, eh.MatchEvents{
		reservations.ReservationCreatedEvent,
		reservations.ReservationConfirmedEvent,
		reservations.ReservationDeclinedEvent,
		reservations.ReservationTimeChangedEvent,
		reservations.ReservationCancelledEvent,
	}, billingProjector)
}
