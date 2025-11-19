module github.com/nordix/meridio/examples/target

go 1.24.0

require (
	github.com/nordix/meridio v0.8.0
	google.golang.org/grpc v1.71.1
)

require (
	golang.org/x/net v0.45.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/nordix/meridio => ../..
