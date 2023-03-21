module github.com/nordix/meridio/examples/target

go 1.20

require (
	github.com/nordix/meridio v0.8.0
	google.golang.org/grpc v1.53.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230223222841-637eb2293923 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/nordix/meridio => ../..
