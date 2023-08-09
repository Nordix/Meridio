module github.com/nordix/meridio/examples/target

go 1.21

require (
	github.com/nordix/meridio v0.8.0
	google.golang.org/grpc v1.57.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230525234030-28d5490b6b19 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace github.com/nordix/meridio => ../..
