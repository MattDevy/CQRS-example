package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/MattDevy/CQRS-example/pkg/reservations"
	"github.com/MattDevy/CQRS-example/pkg/tracing"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func main() {
	// Connect to localhost if not running inside docker
	tracingURL := os.Getenv("TRACING_URL")
	if tracingURL == "" {
		tracingURL = "localhost"
	}

	tracing.InitOpenCensus(tracingURL, "writer")

	options := []option.ClientOption{
		option.WithEndpoint("localhost:8085"),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()),
	}
	client, err := reservations.NewClient("test", options...)
	if err != nil {
		log.Fatalln(err)
	}

	waitEnter()

	mattReservationID := uuid.New()

	// Create a reservation
	var cmd eh.Command
	cmd = &reservations.CreateReservation{
		ID:        mattReservationID,
		Name:      "My new event",
		User:      "Matt",
		RoomID:    3,
		StartTime: time.Now().Add(30 * time.Minute),
		EndTime:   time.Now().Add(1 * time.Hour),
	}
	if err := client.SendCommand(context.Background(), cmd); err != nil {
		log.Fatalln(err)
	}

	waitEnter()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := time.Now().Add(2 * time.Hour)

	// Move the reservation
	cmd = &reservations.ChangeReservationTime{
		ID:        mattReservationID,
		User:      "Matt",
		StartTime: startTime,
		EndTime:   endTime,
	}
	if err := client.SendCommand(context.Background(), cmd); err != nil {
		log.Fatalln(err)
	}

	waitEnter()

	// Create a clashing reservation for room 3
	cmd = &reservations.CreateReservation{
		ID:        uuid.New(),
		Name:      "Joyce's birthday bash",
		RoomID:    3,
		User:      "Joyce",
		StartTime: startTime,
		EndTime:   endTime,
	}
	if err := client.SendCommand(context.Background(), cmd); err != nil {
		log.Fatalln(err)
	}

	waitEnter()

	// Cancel inital reservation
	cmd = &reservations.CancelReservation{
		ID:   mattReservationID,
		User: "Matt",
	}
	if err := client.SendCommand(context.Background(), cmd); err != nil {
		log.Fatalln(err)
	}

}

// waitEnter will wait until the user presses the enter key
func waitEnter() {
	var null string

	fmt.Println("\nHit enter to continue...")
	fmt.Scanln(&null)
}
