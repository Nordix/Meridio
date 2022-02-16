# Target-Client

Ambassador client testing application

## Default trench (namespace of the target)

Open a stream
```
./target-client open -t trench-a -c load-balancer -s stream-a
```

Close a stream
```
./target-client close -t trench-a -c load-balancer -s stream-a
```

Watch stream events (on each event the full list is sent with the status of each stream)
```
./target-client watch
```