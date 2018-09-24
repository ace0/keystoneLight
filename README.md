# Keystone Light
Keystone is a toy version of a clustered key-value service. New server nodes can connect to any existing node to clone clustere state and join the cluster. Clients can issue reads and write to any server and state is replicated to all active nodes in the cluster.

## Quickstart

<!-- Prerequisites:
go get -u google.golang.org/grpc
go get -u github.com/golang/protobuf/protoc-gen-go
 -->

Install this package:
```
go get -u github.com/ace0/keystoneLight
```

Start a single-server cluster:
```
go run go/src/github.com/ace0/keystoneLight/server/main.go
```

In separate terminals, add two more nodes to the cluster:
```
go run go/src/github.com/ace0/keystoneLight/server/main.go localhost:1989
go run go/src/github.com/ace0/keystoneLight/server/main.go localhost:1989
```

Connect with a client and read/write values from separate nodes:
```
go run go/src/github.com/ace0/keystoneLight/client/main.go localhost:1989 keith stone
go run go/src/github.com/ace0/keystoneLight/client/main.go localhost:1991 keith
go run go/src/github.com/ace0/keystoneLight/client/main.go localhost:1990 keith

go run go/src/github.com/ace0/keystoneLight/client/main.go localhost:1990 keith stoooone
go run go/src/github.com/ace0/keystoneLight/client/main.go localhost:1989 keith
```

Kill the first node and continue reading/write
```
go run go/src/github.com/ace0/keystoneLight/client/main.go localhost:1990 "So Smooth" "You know it"
go run go/src/github.com/ace0/keystoneLight/client/main.go localhost:1991 "So Smooth"
```

## Limitations

This is a toy service and some limitations apply.

No locking. Data races may exist within a server if simultaneous write requests are received.

Network segmentation. If the network is segmented, the cluster's state will drift if writes are received. There is no anti-entropy or reconcilliation when cluster nodes re-connect.

Consistency is not guaranteed if there are simultanous writes to the same key. The current implementation can be thought of as read quorum = 1 and write quorum = n (because nodes issue writes to all known peers); however, two-phase commit is not used, so this does not guarantee consistency. (Adding 2PC would achieve this at slight overhead.)
