module github.com/nordix/meridio/examples/target

go 1.24

require (
	github.com/nordix/meridio v0.8.0
	google.golang.org/grpc v1.71.1
)

require (
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/nordix/meridio => ../..
