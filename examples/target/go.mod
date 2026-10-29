module github.com/nordix/meridio/examples/target

go 1.26.9

require (
	github.com/nordix/meridio v0.8.0
	google.golang.org/grpc v1.83.2
)

require (
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260825221802-da73d73af1c5 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/nordix/meridio => ../..
