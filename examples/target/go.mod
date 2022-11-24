module github.com/nordix/meridio/examples/target

go 1.18

require (
	github.com/nordix/meridio v0.8.0
	google.golang.org/grpc v1.49.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.1.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/genproto v0.0.0-20220908141613-51c1cc9bc6d0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace github.com/nordix/meridio => ../..
