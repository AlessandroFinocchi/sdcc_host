package main

import (
	"context"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	m "sdcc_host/model"
	s "sdcc_host/services"
)

func main() {
	ctx := context.Background()

	// Initialize Protocols
	rc, uniqueId := s.NewRegistryConnectorClient()
	membershipProtocol := s.NewMembershipProtocol()
	vivaldiGossip := s.NewVivaldiGossip()
	vivaldiProtocol := s.NewVivaldiProtocol(vivaldiGossip)

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
