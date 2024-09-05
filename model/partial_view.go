package model

import (
	"fmt"
	cm "github.com/AlessandroFinocchi/sdcc_common/model"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	"github.com/AlessandroFinocchi/sdcc_common/utils"
	u "github.com/AlessandroFinocchi/sdcc_common/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"math/rand"
	"sort"
	"sync"
)

type PartialView struct {
	ViewSize          int
	descList          DescriptorList
	currentServerNode *pb.Node // The node that is running the membership and vivaldi protocols
	healers           int
	swappers          int
	mu                *sync.RWMutex
	r                 *rand.Rand
}

func NewPartialView(currentServerNode *pb.Node, nodeList []*pb.Node) *PartialView {
	var healers, swappers int
	viewSize, err := u.ReadConfigInt("config.ini", "membership", "c")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	viewSelection := u.ReadConfigString("config.ini", "membership", "view_selection")

	switch viewSelection {
	case "blind":
		healers = 0
		swappers = 0
	case "healer":
		healers = viewSize / 2
		swappers = 0
	case "swapper":
		healers = 0
		swappers = viewSize / 2
	default:
		fmt.Println("Invalid view selection strategy. Using default values.")
		healers = 0
		swappers = 0
	}

	if viewSize < 0 || healers < 0 || swappers < 0 || viewSize < healers+swappers {
		log.Fatalf("Invalid membership configuration values")
	}

	DescList := make(DescriptorList, 0, viewSize*2)
	for _, node := range nodeList {
		membershipNodeInterface, mIp, mPort, errM := getMembershipInterface(cm.ProtoNodeMembershipAddress(node))
		vivaldiNodeInterface, vIp, vPort, errV := getVivaldiInterface(cm.ProtoNodeVivaldiAddress(node))
		vivaldiGossipNodeInterface, gIp, gPort, errG := getGossipInterface(cm.ProtoNodeGossipAddress(node))
		if errM != nil || errV != nil || errG != nil {
			continue
		}

		currentNode := &pb.Node{
			Id:             currentServerNode.Id,
			MembershipIp:   mIp,
			MembershipPort: mPort,
			VivaldiIp:      vIp,
			VivaldiPort:    vPort,
			GossipIp:       gIp,
			GossipPort:     gPort,
		}

		DescList = append(DescList, &Descriptor{
			receiverServerNode:         node,
			currentClientNode:          currentNode,
			age:                        0,
			membershipNodeInterface:    membershipNodeInterface,
			vivaldiNodeInterface:       vivaldiNodeInterface,
			vivaldiGossipNodeInterface: vivaldiGossipNodeInterface,
		})
	}
	return &PartialView{
		descList:          DescList,
		currentServerNode: currentServerNode,
		ViewSize:          viewSize,
		healers:           healers,
		swappers:          swappers,
		mu:                &sync.RWMutex{},
		r:                 rand.New(rand.NewSource(42)),
	}
}

func getMembershipInterface(address string) (pb.MembershipClient, string, uint32, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, "", 0, err
	}
	ip, port, err := utils.GetLocalIPPort(conn, "Membership")
	if err != nil {
		return nil, "", 0, err
	}
	membershipNodeInterface := pb.NewMembershipClient(conn)

	return membershipNodeInterface, ip, port, nil
}

func getVivaldiInterface(address string) (pb.VivaldiClient, string, uint32, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, "", 0, err
	}
	ip, port, err := utils.GetLocalIPPort(conn, "Vivaldi")
	if err != nil {
		return nil, "", 0, err
	}
	vivaldiNodeInterface := pb.NewVivaldiClient(conn)

	return vivaldiNodeInterface, ip, port, nil
}

func getGossipInterface(address string) (pb.VivaldiGossipClient, string, uint32, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, "", 0, err
	}
	ip, port, err := utils.GetLocalIPPort(conn, "Gossip")
	if err != nil {
		return nil, "", 0, err
	}
	vivaldiGossipNodeInterface := pb.NewVivaldiGossipClient(conn)

	return vivaldiGossipNodeInterface, ip, port, nil
}

func (pv *PartialView) GetCurrentServerNode() *pb.Node {
	return pv.currentServerNode
}

func (pv *PartialView) GetRandomDescriptor() (*Descriptor, bool) {
	pv.mu.RLock()
	defer pv.mu.RUnlock()
	if len(pv.descList) == 0 {
		return &Descriptor{}, false
	}
	return pv.descList[rand.Intn(len(pv.descList))], true
}

func (pv *PartialView) GetSendingNodes() []*pb.Node {
	pv.mu.RLock()
	defer pv.mu.RUnlock()

	if len(pv.descList) == 0 {
		return []*pb.Node{pv.GetCurrentServerNode()}
	}

	sendingNodesSize := min(len(pv.descList), pv.ViewSize/2)
	sendingNodes := make([]*pb.Node, 0, sendingNodesSize)

	// Step 1: add the current node to the list of sending nodes as the first node
	sendingNodes = append(sendingNodes, pv.currentServerNode)

	// Step 2: sample the sending nodes from the youngest nodes
	lastConsideredNode := min(len(pv.descList), pv.ViewSize-pv.healers)
	consideringYoungestView := pv.descList[:lastConsideredNode]
	consideringYoungestView.RemoveDescriptorFromReceiverNodeId(pv.currentServerNode.Id)

	if sendingNodesSize <= len(consideringYoungestView) {
		// Sample the sending nodes ignoring the oldest healers
		pv.r.Shuffle(len(consideringYoungestView), func(i, j int) {
			consideringYoungestView[i], consideringYoungestView[j] = consideringYoungestView[j], consideringYoungestView[i]
		})
		for i := 0; len(sendingNodes) != sendingNodesSize; i++ {
			sendingNodes = append(sendingNodes, consideringYoungestView[i].receiverServerNode)
		}
	} else {
		// Sample the sending nodes getting all the youngest nodes and the remaining from the oldest healers
		consideringOldestView := pv.descList[lastConsideredNode:]
		consideringOldestView.RemoveDescriptorFromReceiverNodeId(pv.currentServerNode.Id)
		pv.r.Shuffle(len(consideringOldestView), func(i, j int) {
			consideringOldestView[i], consideringOldestView[j] = consideringOldestView[j], consideringOldestView[i]
		})
		for _, desc := range consideringYoungestView {
			sendingNodes = append(sendingNodes, desc.currentClientNode)
		}
		for i := 0; len(sendingNodes) != sendingNodesSize; i++ {
			sendingNodes = append(sendingNodes, consideringOldestView[i].receiverServerNode)
		}
	}

	return sendingNodes
}

func (pv *PartialView) RemoveDescriptor(desc *Descriptor) {
	pv.mu.Lock()
	defer pv.mu.Unlock()
	pv.descList.RemoveDescriptor(desc)
}

func (pv *PartialView) containsNode(node *pb.Node) bool {
	for _, desc := range pv.descList {
		if desc.receiverServerNode.Id == node.Id {
			return true
		}
	}
	return false
}

func (pv *PartialView) MergeViews(nodes []*pb.Node) {
	pv.mu.Lock()
	defer pv.mu.Unlock()

	for _, node := range nodes {
		// Skip the current node and the nodes that are already in the partial view
		if pv.containsNode(node) || node.GetId() == pv.currentServerNode.GetId() {
			continue
		}

		membershipNodeInterface, mIp, mPort, errM := getMembershipInterface(cm.ProtoNodeMembershipAddress(node))
		vivaldiNodeInterface, vIp, vPort, errV := getVivaldiInterface(cm.ProtoNodeVivaldiAddress(node))
		vivaldiGossipNodeInterface, gIp, gPort, errG := getGossipInterface(cm.ProtoNodeGossipAddress(node))
		if errM != nil || errV != nil || errG != nil {
			continue
		}

		currentNode := &pb.Node{
			Id:             pv.currentServerNode.Id,
			MembershipIp:   mIp,
			MembershipPort: mPort,
			VivaldiIp:      vIp,
			VivaldiPort:    vPort,
			GossipIp:       gIp,
			GossipPort:     gPort,
		}

		pv.descList = append(pv.descList, &Descriptor{
			receiverServerNode:         node,
			currentClientNode:          currentNode,
			age:                        0,
			membershipNodeInterface:    membershipNodeInterface,
			vivaldiNodeInterface:       vivaldiNodeInterface,
			vivaldiGossipNodeInterface: vivaldiGossipNodeInterface,
		})
	}

	// Remove the FIRST (not newer) swappers items
	if len(pv.descList) > pv.ViewSize {
		removingSwappers := min(pv.swappers, len(pv.descList)-pv.ViewSize)
		pv.descList = pv.descList[removingSwappers:]
	}

	sort.Sort(pv.descList)

	// Remove the OLDEST (not last) healers items
	if len(pv.descList) > pv.ViewSize {
		removingHealers := len(pv.descList) - min(pv.healers, len(pv.descList)-pv.ViewSize)
		pv.descList = pv.descList[:removingHealers]
	}

	// Remove random items
	for len(pv.descList) > pv.ViewSize {
		i := rand.Intn(len(pv.descList))
		pv.descList = append(pv.descList[:i], pv.descList[i+1:]...)
	}

	// Set to 0 the age of the youngest node
	if len(pv.descList) != 0 {
		youngestDesc := pv.descList[len(pv.descList)-1]
		pv.increaseAge(1 - youngestDesc.age)
	}

	//fmt.Println("Merged view")
	//for _, desc := range pv.descList {
	//	fmt.Println(desc.receiverServerNode.Id, ": ", cm.ProtoNodeMembershipAddress(desc.receiverServerNode), " age: ", desc.age)
	//}
	//fmt.Println()
}

func (pv *PartialView) increaseAge(n int) {
	for _, desc := range pv.descList {
		desc.age += n
	}
}
