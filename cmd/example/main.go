package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	gcpEventBus "github.com/looplab/eventhorizon/eventbus/gcp"
	tracingEventBus "github.com/looplab/eventhorizon/eventbus/tracing"
	mongoEventStore "github.com/looplab/eventhorizon/eventstore/mongodb"
	tracingEventStore "github.com/looplab/eventhorizon/eventstore/tracing"
	"github.com/looplab/eventhorizon/middleware/eventhandler/observer"
	mongoRepo "github.com/looplab/eventhorizon/repo/mongodb"
	tracingRepo "github.com/looplab/eventhorizon/repo/tracing"
	"github.com/looplab/eventhorizon/repo/version"

	"github.com/MattDevy/CQRS-example/pkg/billing"
	"github.com/MattDevy/CQRS-example/pkg/reservations"
	"github.com/MattDevy/CQRS-example/pkg/tracing"
	ctracing "github.com/looplab/eventhorizon/middleware/commandhandler/tracing"
)

func main() {
	// Connect to localhost if not running inside docker
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8085")
	}

	// Connect to localhost if not running inside docker
	tracingURL := os.Getenv("TRACING_URL")
	if tracingURL == "" {
		tracingURL = "localhost"
	}

	// Set up tracing
	tracing.InitOpenCensus(tracingURL, "receiver")
	traceCloser, err := tracing.NewTracer("reservations", tracingURL)
	if err != nil {
		log.Fatal("could not create tracer: ", err)
	}
	defer func() {
		if err := traceCloser.Close(); err != nil {
			log.Printf("Could not close tracer\n")
		}
	}()

	// Configuration vars
	var (
		GCPProject         = "test"
		GCPAppID           = "reservations"
		PubSubCommandTopic = reservations.RoomCommandsTopic
		MongoURL           = "mongodb://localhost:27017"
		MongoDB            = "reservations"
	)

	// Create the pub sub event bus
	eventBus := NewPubSubEventBus(GCPProject, GCPAppID)
	// Wrap the event bus to add tracing.
	eventBus = tracingEventBus.NewEventBus(eventBus)

	// Create the event store.
	eventStore := NewMongoEventStore(eventBus, MongoURL, MongoDB)
	eventStore = tracingEventStore.NewEventStore(eventStore)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add an event logger as an observer.
	eventLogger := &EventLogger{}
	if err := eventBus.AddHandler(ctx, eh.MatchAll{},
		eh.UseEventHandlerMiddleware(eventLogger,
			observer.NewMiddleware(observer.NamedGroup("reservations")),
		),
	); err != nil {
		log.Fatal("could not add event logger: ", err)
	}

	// Create mongo projection repos
	reservationRepo := NewMongoRepo(MongoURL, MongoDB, "reservations")
	billingRepo := NewMongoRepo(MongoURL, MongoDB, "billing")

	// Create the command bus to handle all commands
	commandBus := bus.NewCommandHandler()
	// Add tracing middleware to init tracing spans, and the logging middleware.
	commandHandler := eh.UseCommandHandlerMiddleware(commandBus,
		ctracing.NewMiddleware(),
		CommandLogger,
	)

	// Set up models, commands etc....
	billing.Setup(ctx, eventStore, eventBus, commandBus, billingRepo)
	reservations.Setup(ctx, eventStore, eventBus, commandBus, reservationRepo)

	wg := sync.WaitGroup{}

	// Handle incoming commands
	commandChan := NewCommandChannel(&wg, commandHandler)
	GetCommandsFromPubSub(ctx, GCPProject, GCPAppID, PubSubCommandTopic, commandChan)

	// Wait for everything to complete
	eventBus.Wait()
	wg.Done()

}

func NewPubSubEventBus(project, appID string) eh.EventBus {
	eventBus, err := gcpEventBus.NewEventBus(project, appID)
	if err != nil {
		log.Fatal("could not create event bus: ", err)
	}
	go func() {
		for err := range eventBus.Errors() {
			log.Print("eventbus:", err)
		}
	}()
	return eventBus
}

func NewMongoEventStore(eventBus eh.EventBus, url, database string) eh.EventStore {
	eventStore, err := mongoEventStore.NewEventStore(url, database,
		mongoEventStore.WithEventHandler(eventBus), // Add the event bus as a handler after save.
	)
	if err != nil {
		log.Fatal("could not create event store: ", err)
	}
	return eventStore
}

func NewMongoRepo(url, database, colleciton string) eh.ReadWriteRepo {
	repo, err := mongoRepo.NewRepo(url, database, colleciton)
	if err != nil {
		log.Fatal("could not create invitation repository: ", err)
	}
	return tracingRepo.NewRepo(version.NewRepo(repo))
}

func NewCommandChannel(wg *sync.WaitGroup, commandHandler eh.CommandHandler) chan<- eh.Command {
	commandChan := make(chan eh.Command, 300)
	go func() {
		wg.Add(1)
		defer wg.Done()
		for cmd := range commandChan {
			if err := commandHandler.HandleCommand(context.Background(), cmd); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}()
	return commandChan
}

func GetCommandsFromPubSub(ctx context.Context, project, appID, topicName string, commandChan chan<- eh.Command) {
	client, err := pubsub.NewClient(context.Background(), project)
	if err != nil {
		log.Fatal(err)
	}

	topic := client.Topic(topicName)
	if exists, err := topic.Exists(context.Background()); err != nil {
		log.Fatal(err)
	} else if !exists {
		topic, err = client.CreateTopicWithConfig(context.Background(), topicName, &pubsub.TopicConfig{})
		if err != nil {
			log.Fatal(err)
		}
	}

	sub := client.Subscription("test")
	if exists, err := sub.Exists(context.Background()); !exists && err == nil {
		sub, err = client.CreateSubscription(ctx, "test", pubsub.SubscriptionConfig{
			Topic: topic,
		})
	}
	if err != nil {
		log.Fatalln(err)
	}

	sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		commandType, ok := msg.Attributes["CommandType"]
		if !ok {
			fmt.Println("No command type set")
			return
		}
		cmd, err := eh.CreateCommand(eh.CommandType(commandType))
		if err != nil {
			fmt.Println("unknown command type")
			return
		}

		if err := json.Unmarshal(msg.Data, cmd); err != nil {
			fmt.Println("Bad command")
			return
		}

		commandChan <- cmd
	})
}

// EventLogger is a simple event handler for logging all events.
type EventLogger struct{}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (l *EventLogger) HandlerType() eh.EventHandlerType {
	return "logger"
}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (l *EventLogger) HandleEvent(ctx context.Context, event eh.Event) error {
	log.Printf("EVENT: %s\n", event)
	return nil
}

// CommandLogger is an example of a function based logging middleware.
func CommandLogger(h eh.CommandHandler) eh.CommandHandler {
	return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
		log.Printf("CMD: %#v\n", cmd)
		return h.HandleCommand(ctx, cmd)
	})
}
