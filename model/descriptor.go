package model

import (
	"context"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
)

// Descriptor struct contains the information of each node in the partial view and the (ip, port) used by the
// current node to communicate with it: every receiver communicates with the current node using a different port of its.
type Descriptor struct {
	receiverServerNode         *pb.Node // The node the current node wants to communicate with
	currentClientNode          *pb.Node // The current node infos for each receiver node (ip and/or port change for each receiver)
	age                        int
	membershipNodeInterface    pb.MembershipClient
	vivaldiNodeInterface       pb.VivaldiClient
	vivaldiGossipNodeInterface pb.VivaldiGossipClient
}

// DescriptorList Implement sort.Interface for order them increasingly by age
type DescriptorList []*Descriptor

func (dl DescriptorList) Len() int           { return len(dl) }
func (dl DescriptorList) Swap(i, j int)      { dl[i], dl[j] = dl[j], dl[i] }
func (dl DescriptorList) Less(i, j int) bool { return dl[i].age < dl[j].age } // Desc

func (dl *DescriptorList) GetDescriptorFromReceiverNode(node *pb.Node) *Descriptor {
	for _, d := range *dl {
		if d.receiverServerNode.Id == node.Id {
			return d
		}
	}
	return nil
}

func (dl *DescriptorList) RemoveDescriptor(desc *Descriptor) bool {
	for i, d := range *dl {
		if d == desc {
			*dl = append((*dl)[:i], (*dl)[i+1:]...)
			return true
		}
	}
	return false
}

func (dl *DescriptorList) RemoveDescriptorFromReceiverNodeId(id string) bool {
	for i, d := range *dl {
		if d.receiverServerNode.Id == id {
			_ = append((*dl)[:i], (*dl)[i+1:]...)
			return true
		}
	}
	return false
}

func (dl *Descriptor) ShufflePeers(request *pb.MembershipRequestMessage) (*pb.MembershipReplyMessage, error) {
	return dl.membershipNodeInterface.ShufflePeers(context.Background(), request)
}

func (dl *Descriptor) PullCoordinates() (*pb.VivaldiCoordinate, error) {
	return dl.vivaldiNodeInterface.PullCoordinates(context.Background(), &pb.Empty{})
}

func (dl *Descriptor) GossipCoordinates(coords *pb.GossipCoordinateList) (*pb.GossipCoordinateList, error) {
	return dl.vivaldiGossipNodeInterface.Gossip(context.Background(), coords)
}

func (dl *Descriptor) GetReceiverNode() *pb.Node {
	return dl.receiverServerNode
}
