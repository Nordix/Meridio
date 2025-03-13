module github.com/nordix/meridio/examples/target

go 1.24

require (
	github.com/nordix/meridio v0.8.0
	google.golang.org/grpc v1.60.1
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231030173426-d783a09b4405 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/nordix/meridio => ../..
