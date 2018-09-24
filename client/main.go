package main

import (
	"log"
	"fmt"
	"time"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
  pb "github.com/ace0/keystoneLight/keystone"
)

type client struct{
	kc pb.KeystoneClient
}

func failOnErr(msg string, err error) {
	if err != nil {
		log.Fatalf(msg, err)
	}	
}

// Wrapper so we can test many quick reads/writes
func (c *client) readAndPrint(key string) {
	// 1 second timeout on requests
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Read from the server and print the response
	response, err := c.kc.Read(ctx, &pb.Key{Key:key})
	failOnErr("Failed to read", err)
	log.Printf("Read:  %v=%v", key, response.Value)
}

func (c *client) writeAndPrint(key, value string) {
	// 1 second timeout on requests
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Read from the server and print the response
	_, err := c.kc.Write(ctx, &pb.KeyValue{Key:key, Value:value})
	failOnErr("Failed to write", err)
	log.Printf("Wrote: %v=%v", key, value)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: client server:port KEY [VALUE]")
		fmt.Println(" KEY only:  read KEY")
		fmt.Println(" KEY/VALUE: write KEY:VALUE")
		return
	}

	// Parse args
	server,key,value := os.Args[1],os.Args[2],""
	if len(os.Args) > 3 {
		value = os.Args[3]
	}

	// Connect to a server
	conn, err := grpc.Dial(server, grpc.WithInsecure())
	failOnErr("Failed to connect", err)
	defer conn.Close()
	client := client{kc: pb.NewKeystoneClient(conn)}

	// Read or write
	if len(value) > 0 {
		client.writeAndPrint(key, value)
	} else {
		client.readAndPrint(key)
	}
}
