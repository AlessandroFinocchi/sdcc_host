package main

import (
	"context"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	"log"
	"os"
	m "sdcc_host/model"
	s "sdcc_host/services"
	"sdcc_host/vivaldi"
)

func main() {
	// Create/Truncate result file
	file, err := os.Create("/data/results.csv") // Write the file to /data (mapped to a volume)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	_, err = file.WriteString("Time, Error\n")
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
	_ = file.Close()

	ctx := context.Background()

	// Initialize Protocols
	rc, uniqueId := s.NewRegistryConnectorClient()
	filter := vivaldi.NewFilter()
	membershipProtocol := s.NewMembershipProtocol(filter)
	vivaldiGossip := s.NewVivaldiGossip(filter)
	vivaldiProtocol := s.NewVivaldiProtocol(vivaldiGossip, filter)

	// Start Protocols and get address infos
	membershipServerIp, membershipServerPort := membershipProtocol.StartServer()
	vivaldiServerIp, vivaldiServerPort := vivaldiProtocol.StartServer()
	gossipServerip, gossipServerPort := vivaldiGossip.StartServer()

	// Init current server node
	currentServerNode := &pb.Node{
		Id:             uniqueId,
		MembershipIp:   membershipServerIp,
		MembershipPort: membershipServerPort,
		VivaldiIp:      vivaldiServerIp,
		VivaldiPort:    vivaldiServerPort,
		GossipIp:       gossipServerip,
		GossipPort:     gossipServerPort,
	}

	// Connect to Registry
	startingNodeList := rc.Connect(ctx, currentServerNode)

	// Init partial view
	pView := m.NewPartialView(currentServerNode, startingNodeList)
	membershipProtocol.SetPartialView(pView)
	vivaldiProtocol.SetPartialView(pView)
	vivaldiGossip.SetPartialView(pView)

	// Start client protocols
	go membershipProtocol.StartClient()
	go vivaldiProtocol.StartClient()
	go vivaldiGossip.StartClient()

	select {}
}
