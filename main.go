package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	file1, err := os.Create("results2.txt") // Write the file to /data (mapped to a volume)
	if err != nil {
		fmt.Println("Error opening file:", err)
		for {
			time.Sleep(20 * time.Second)
		}
	}
	_ = file1.Close()
	fmt.Println("Successfully created to output.txt")

	file, err := os.Open("results.txt") // Write the file to /data (mapped to a volume)
	if err != nil {
		fmt.Println("Error opening file:", err)
		for {
			time.Sleep(20 * time.Second)
		}
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	_, err = file.WriteString("Hello, Docker!\n")
	if err != nil {
		fmt.Println("Error writing to file:", err)
		for {
			time.Sleep(20 * time.Second)
		}
	}

	fmt.Println("Successfully wrote to output.txt")
	for {
		time.Sleep(20 * time.Second)
	}
	//ctx := context.Background()
	//
	//// Initialize Protocols
	//rc, uniqueId := s.NewRegistryConnectorClient()
	//filter := vivaldi.NewFilter()
	//membershipProtocol := s.NewMembershipProtocol(filter)
	//vivaldiGossip := s.NewVivaldiGossip(filter)
	//vivaldiProtocol := s.NewVivaldiProtocol(vivaldiGossip, filter)
	//
	//// Start Protocols and get address infos
	//membershipServerIp, membershipServerPort := membershipProtocol.StartServer()
	//vivaldiServerIp, vivaldiServerPort := vivaldiProtocol.StartServer()
	//gossipServerip, gossipServerPort := vivaldiGossip.StartServer()
	//
	//// Init current server node
	//currentServerNode := &pb.Node{
	//	Id:             uniqueId,
	//	MembershipIp:   membershipServerIp,
	//	MembershipPort: membershipServerPort,
	//	VivaldiIp:      vivaldiServerIp,
	//	VivaldiPort:    vivaldiServerPort,
	//	GossipIp:       gossipServerip,
	//	GossipPort:     gossipServerPort,
	//}
	//
	//// Connect to Registry
	//startingNodeList := rc.Connect(ctx, currentServerNode)
	//
	//// Init partial view
	//pView := m.NewPartialView(currentServerNode, startingNodeList)
	//membershipProtocol.SetPartialView(pView)
	//vivaldiProtocol.SetPartialView(pView)
	//vivaldiGossip.SetPartialView(pView)
	//
	//// Start client protocols
	//go membershipProtocol.StartClient()
	//go vivaldiProtocol.StartClient()
	//go vivaldiGossip.StartClient()
	//
	//select {}
}
