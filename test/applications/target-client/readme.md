# Target-Client

Ambassador client testing application

## Default trench (namespace of the target)

Connect to a conduit
```
./target-client connect -ns load-balancer
```

Disconnect from a conduit
```
./target-client disconnect -ns load-balancer
```

Request a stream
```
./target-client request -ns load-balancer
```

Close a stream
```
./target-client close -ns load-balancer
```

## Specific Trench 

Connect to a conduit
```
./target-client connect -ns load-balancer -t trench-a
```

Disconnect from a conduit
```
./target-client disconnect -ns load-balancer -t trench-a
```

Request a stream
```
./target-client request -ns load-balancer -t trench-a
```

Close a stream
```
./target-client close -ns load-balancer -t trench-a
```