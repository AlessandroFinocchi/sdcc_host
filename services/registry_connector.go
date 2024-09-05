package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"os"
	uh "sdcc_host/utils"
	"time"
)

const (
	address = "0.0.0.0:50051" // For local testing
	//address = "10.0.0.253:50051"
)

type RegistryConnectorClient struct {
}

func NewRegistryConnectorClient() (*RegistryConnectorClient, string) {
	currUUID, err := uuid.NewUUID()
	if err != nil {
		log.Fatalf("Could not generate UUID: %v", err)
	}

	fmt.Println("Current Host UUID: ", currUUID.String())

	return &RegistryConnectorClient{}, currUUID.String()
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := os.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Load client's certificate and private key
	clientCert, err := tls.LoadX509KeyPair("cert/client-cert.pem", "cert/client-key.pem")
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
	}

	return credentials.NewTLS(config), nil
}

func (rc *RegistryConnectorClient) startHeartbeat(h pb.HeartbeatClient, ctx context.Context, currentNode *pb.Node) {
	for {
		time.Sleep(2 * time.Second)
		ctxT, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := h.Beat(ctxT, &pb.Node{
			Id:             currentNode.GetId(),
			MembershipIp:   currentNode.GetMembershipIp(),
			MembershipPort: currentNode.GetMembershipPort(),
			VivaldiIp:      currentNode.GetVivaldiIp(),
			VivaldiPort:    currentNode.GetVivaldiPort(),
			GossipIp:       currentNode.GetGossipIp(),
			GossipPort:     currentNode.GetGossipPort()})
		if err != nil {
			log.Fatalf("Could not send heartbeat: %v", err)
		}
		cancel()
		//fmt.Println("Beaten registry")
	}
}

func (rc *RegistryConnectorClient) Connect(ctx context.Context, currentServerNode *pb.Node) []*pb.Node {
	tlsCredentials, err := uh.LoadClientTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(tlsCredentials))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	c := pb.NewConnectorClient(conn)
	h := pb.NewHeartbeatClient(conn)

	nodeList, err := c.Connect(ctx, currentServerNode)
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}

	fmt.Println("Node list received:")
	for _, node := range nodeList.Nodes {
		fmt.Println("Node: ", node.GetId(), " ", node.GetMembershipPort(), ":", node.GetVivaldiPort(), ":", node.GetGossipPort())
	}
	fmt.Println()

	go rc.startHeartbeat(h, ctx, currentServerNode)

	return nodeList.Nodes
}
