package main

import (
  "errors"
  "fmt"
  "log"
  "net"
  "os"
  "strings"
  "time"

  "golang.org/x/net/context"
  "google.golang.org/grpc"
  pb "github.com/ace0/keystoneLight/keystone"
)

const (
  host = "localhost"
  startingPort = 1989
  portRange = 100
)

func failOnErr(msg string, err error) {
  if err != nil {
    log.Fatalf(msg, err)
  } 
}

// Keystone server
type node struct{
  address string
  dstore map[string]string
  peers map[string]pb.KeystoneClient
}

// Handle client reads
func (n *node) Read(ctx context.Context, in *pb.Key) (*pb.Value, error) {
  log.Printf("Received client read: %v", in)
  return &pb.Value{Value: n.dstore[in.Key]}, nil
}

// Handle client writes
func (n *node) Write(ctx context.Context, in *pb.KeyValue) (*pb.Nothing, error) {
  log.Printf("Received client write: %v", in)

  // Write this value to all of our peers
  for addr, peer := range n.peers {
    // At least one entry (ourself) will be nil
    if peer != nil {
      ctx, cancel := context.WithTimeout(context.Background(), time.Second)
      defer cancel()

      log.Printf("Writing to %v", addr)

      _, err := peer.ServerWrite(ctx, in)

      // If the peer server is no longer connected, remove it from the list of
      // peers and continue
      if (err != nil && strings.Contains(err.Error(), "connect: connection refused")) {
        log.Printf("Peer %v is disconnected. Dropping peer.")
        delete(n.peers, addr)
      } else if err != nil {
        log.Printf("Failed to write to %v: %v", peer, err)
        return &pb.Nothing{}, err 
      }
    }
  }

  // Update the local data store and report success
  n.dstore[in.Key] = in.Value
  return &pb.Nothing{}, nil
}

// Handle writes from another node
func (n *node) ServerWrite(ctx context.Context, in *pb.KeyValue) (*pb.Nothing, error) {
  log.Printf("Received node write: %v", in)
  n.dstore[in.Key] = in.Value
  return &pb.Nothing{}, nil  
}

// Get information about the current state of the cluster
func (n *node) GetClusterInfo(ctx context.Context, in *pb.Nothing) (*pb.ClusterInfo, error) {
  log.Println("Received cluster info request")
  return &pb.ClusterInfo{
    Dstore: n.dstore,
    Peers: keys(n.peers),
  }, nil
}

// Another node is joining our cluster
func (n *node) Join(ctx context.Context, in *pb.ServerInfo) (*pb.Nothing, error) {
  // Connect to this node and store it in our list of peers
  client, err := connectPeer(in.Address)
  if err != nil {
    log.Printf("Failed to connect to peer (%v): %v", in.Address, err)
    return &pb.Nothing{}, err
  }
  log.Printf("Adding peer node %v", in.Address)
  n.peers[in.Address] = client
  return &pb.Nothing{}, nil
}

// Join an existing cluster
func (n *node) joinCluster(address string) {
  // Connect to a node and fetch cluster state
  client, err := connectPeer(address)  
  failOnErr(fmt.Sprintf("Failed to connect to %v", address), err)
  // defer conn.Close()
  n.peers[address] = client

  ctx, cancel := context.WithTimeout(context.Background(), time.Second)
  defer cancel()

  info, err := client.GetClusterInfo(ctx, &pb.Nothing{})
  failOnErr("Could not get cluster info", err)
  log.Printf("Connecting to cluster: %v", info)

  // Copy the existing data store
  for k,v := range info.Dstore {
    n.dstore[k] = v
  }

  // Connect to each node in turn and store the client connections
  for _,addr := range info.Peers {
    // In case we're listed as a peer in the network, don't connect to 
    // ourselves
    if addr == n.address {
      continue
    }

    // Connect to any peers we're not already connected to
    if _, ok := n.peers[addr]; !ok {
      client, err = connectPeer(addr)
      failOnErr(fmt.Sprintf("Failed connect to peer: %v", addr), err)
      n.peers[addr] = client
    }
    // Call Join() on this peer so it knows about us
    log.Printf("Contacting node %v", addr)
    n.peers[addr].Join(ctx, &pb.ServerInfo{Address:n.address})
  }

  log.Printf("Cluster joined. Peers: %v", n.peers)
}

// Connect to a peer node, create a client, and get a context for requests
func connectPeer(address string) (pb.KeystoneClient, error) {
  // Open a TCP connection
  conn, err := grpc.Dial(address, grpc.WithInsecure())
  if err != nil {
    return nil,  err
  }

  // Create a client object
  client := pb.NewKeystoneClient(conn)
  return client, nil
}

// Fetch the keys from a map
func keys(m map[string]pb.KeystoneClient) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}

// Opens a listener on an unused port
func listenerOnUnusedPort(host string, start, max int) (string, net.Listener, error) {
  // Search this port range
  for port := start; port < start + max; port += 1 {
    address :=  fmt.Sprintf("%v:%d", host, port)
    lis, err := net.Listen("tcp", address)

    // Success!
    if err == nil {
      return address, lis, nil
    }

    // Bail out on any error except "address in use"
    if ! strings.Contains(err.Error(), "bind: address already in use") {
      return "", nil, err
    }
  }
  // We failed
  return "", nil, errors.New("Failed to find unused port in range")
}

// Run a Keystone Light node
func main() {
  // Open a listener
  address, lis, err := listenerOnUnusedPort(host, startingPort, portRange)
  failOnErr("Failed to create listener", err)

  n := node{
      address:address,
      dstore: make(map[string]string),
      peers: map[string]pb.KeystoneClient{address:nil},
    }

  // If we're passed an address on the cli, connect to that address and 
  // join the cluster
  if len(os.Args) > 1 {
    n.joinCluster(os.Args[1])
  }

  // Create a server and start handling requests
  s := grpc.NewServer()
  pb.RegisterKeystoneServer(s, &n)

  log.Printf("Keystone Light node listening (%v): "+
    "Always smooth, even when you're not.", address)

  err = s.Serve(lis)
  failOnErr("Failed to start server", err)
}
