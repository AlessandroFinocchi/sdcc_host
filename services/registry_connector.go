package services

import (
	"context"
	"fmt"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"log"
	"os"
	m "sdcc_host/model"
	uh "sdcc_host/utils"
	"strconv"
	"time"
)

const (
	//address = "0.0.0.0:50051" // For local testing
	address = "10.0.0.253:50051"
)

type RegistryConnectorClient struct {
	logger uh.MyLogger
}

func NewRegistryConnectorClient() (*RegistryConnectorClient, string) {
	currUUID, err := uuid.NewUUID()
	logging, errL := strconv.ParseBool(os.Getenv(m.LoggingEnv))
	if err != nil || errL != nil {
		log.Fatalf("Error in registry connector: %v", err)
	}

	fmt.Println("Current Host UUID: ", currUUID.String())

	return &RegistryConnectorClient{uh.NewMyLogger(logging)}, currUUID.String()
}

func (rc *RegistryConnectorClient) startHeartbeat(h pb.HeartbeatClient, ctx context.Context, currentNode *pb.Node) {
	for {
		time.Sleep(4 * time.Second)
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

	rc.logger.Log("Node list received:")
	for _, node := range nodeList.Nodes {
		rc.logger.Log(fmt.Sprintf("Node: %s %d:%d:%d", node.GetId(), node.GetMembershipPort(), node.GetVivaldiPort(), node.GetGossipPort()))
	}
	rc.logger.Log("")

	go rc.startHeartbeat(h, ctx, currentServerNode)

	return nodeList.Nodes
}
