package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"
	"time"

	pb "MandatoryActivity4/MandatoryActivity4/Node.go"

	"google.golang.org/grpc"
)

type Node struct {
	pb.UnimplementedNodeServiceServer
	id       string
	address  string
	nextNode string
	hasToken bool
	mu       sync.Mutex
}

func (n *Node) RequestToken(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	log.Printf("Node %s received a token request", n.id)

	if n.hasToken {
		log.Printf("Node %s has the token", n.id)
		n.enterCriticalSection()
		n.hasToken = false
		log.Printf("Node %s is passing the token", n.id)
		n.passToken()
	} else {
		log.Printf("Node %s does not have the token", n.id)
	}

	return &pb.Response{Message: "Token requested"}, nil
}

func (n *Node) ReleaseToken(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	log.Printf("Node %s received a token release request", n.id)

	n.hasToken = true
	log.Printf("Node %s has accepted the token", n.id)
	return &pb.Response{Message: "Token released"}, nil
}

func (n *Node) enterCriticalSection() {
	log.Printf("Node %s entering critical section", n.id)
	// Emulate critical section
	time.Sleep(2 * time.Second)
	log.Printf("Node %s leaving critical section", n.id)
}

func (n *Node) passToken() {
	log.Printf("Node %s attempting to pass the token to %s", n.id, n.nextNode)
	conn, err := grpc.Dial(n.nextNode, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to next node: %v", err)
	}
	defer conn.Close()

	client := pb.NewNodeServiceClient(conn)
	_, err = client.ReleaseToken(context.Background(), &pb.Request{NodeId: n.id})
	if err != nil {
		log.Fatalf("Failed to pass token: %v", err)
	}
	log.Printf("Node %s successfully passed the token to %s", n.id, n.nextNode)
}

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("Usage: %s <node_id> <address> <next_node_address>", os.Args[0])
	}

	// Create or open the log file
	logFile, err := os.OpenFile("node.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Set log output to the file
	log.SetOutput(logFile)

	node := &Node{
		id:       os.Args[1],
		address:  os.Args[2],
		nextNode: os.Args[3],
		hasToken: false,
	}

	if node.id == "1" {
		node.hasToken = true
	}

	lis, err := net.Listen("tcp", node.address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterNodeServiceServer(grpcServer, node)

	log.Printf("Node %s listening on %s", node.id, node.address)

	// Start a goroutine to request the token periodically
	go func() {
		for {
			time.Sleep(5 * time.Second) // Adjust the interval as needed
			if !node.hasToken {
				conn, err := grpc.Dial(node.nextNode, grpc.WithInsecure())
				if err != nil {
					log.Printf("Failed to connect to next node: %v", err)
					continue
				}
				defer conn.Close()

				client := pb.NewNodeServiceClient(conn)
				_, err = client.RequestToken(context.Background(), &pb.Request{NodeId: node.id})
				if err != nil {
					log.Printf("Failed to request token: %v", err)
				}
			}
		}
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
