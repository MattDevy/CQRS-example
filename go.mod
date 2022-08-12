module github.com/MattDevy/CQRS-example

go 1.16

require (
	cloud.google.com/go/pubsub v1.12.0
	contrib.go.opencensus.io/exporter/zipkin v0.1.2
	github.com/google/uuid v1.2.0
	github.com/looplab/eventhorizon v0.14.3
	github.com/looplab/fsm v0.2.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	go.opencensus.io v0.23.0
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	google.golang.org/api v0.49.0
	google.golang.org/genproto v0.0.0-20210629135825-364e77e5a69d // indirect
	google.golang.org/grpc v1.38.0
)
