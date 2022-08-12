# CQRS-example

## Description
This repo is a demo for showing how CQRS can be useful in certain situations, and to give insights into how a CQRS eco system might look (albeit as a pretty monolithic code base here)

This repo includes
- A server to handle commands, emit events and execute business logic
- Multiple views of the data depending on concern (billing, reservations)
- And of course, tracing (visit http://localhost:16686)
## Getting started
```sh
docker-compose up -d
go run ./cmd/example

# New tab
go run ./cmd/writer
# hit enter a bunch of times, have a look at the mongo db data between each enter press to understand the examples
```

In the ./cmd/example tab you will see logging output.

## Use mongo to see data

```sh
mongo
> use reservations
> db.events.find().pretty()
...
> db.reservations.find().pretty()
...
> db.billing.find().pretty()
```

## Tidy-up
You must tidy-up between runs of the example, this is because shortcuts have been taken to throw together a demo. And certain bits of critical state are not persisted
```sh
docker-compose down
```