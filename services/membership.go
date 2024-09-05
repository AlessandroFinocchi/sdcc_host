package services

import (
	"context"
	"flag"
	"fmt"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	u "github.com/AlessandroFinocchi/sdcc_common/utils"
	"google.golang.org/grpc"
	"log"
	"net"
	m "sdcc_host/model"
	"sync"
	"time"
)

type MembershipProtocol struct {
	pb.UnimplementedMembershipServer
	pView *m.PartialView
	mu    *sync.RWMutex
}

func NewMembershipProtocol() *MembershipProtocol {
	membershipProtocol := &MembershipProtocol{
		mu: &sync.RWMutex{},
	}

	return membershipProtocol
}

func (mp *MembershipProtocol) ShufflePeers(ctx context.Context, request *pb.MembershipRequestMessage) (*pb.MembershipReplyMessage, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if err := u.ContextError(ctx); err != nil {
		return nil, err
	}

	if len(request.GetNodes()) > mp.pView.ViewSize {
		return nil, fmt.Errorf("invalid message")
	}

	sendingNodes := mp.pView.GetSendingNodes()

	mp.pView.MergeViews(request.GetNodes())

	return &pb.MembershipReplyMessage{Nodes: sendingNodes}, nil
}

func (mp *MembershipProtocol) StartServer() (string, uint32) {
	flag.Parse()
	serverAddress := fmt.Sprintf(":%d", *m.MembershipPort)
	lis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	serverIp, err := u.GetIpFromListener(lis)
	if err != nil {
		log.Fatalf("Failed to get IP from listener: %v", err)
	}
	serverPort := uint32(*m.MembershipPort)

	registry := grpc.NewServer()
	pb.RegisterMembershipServer(registry, mp)

	go func() {
		err = registry.Serve(lis)
		if err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return serverIp, serverPort
}

func (mp *MembershipProtocol) StartClient() {
	if mp.pView == nil {
		log.Fatalf("Partial view is not initialized")
	}

	samplingInterval, err := u.ReadConfigInt("config.ini", "membership", "sampling_interval")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Distribute the coordinates
	ticker := time.NewTicker(time.Duration(samplingInterval) * time.Second)
	for range ticker.C {
		desc, ok := mp.pView.GetRandomDescriptor()
		if ok {
			request := &pb.MembershipRequestMessage{
				Nodes:  mp.pView.GetSendingNodes(),
				Source: mp.pView.GetCurrentServerNode(),
			}

			reply, errM := desc.ShufflePeers(request)
			if errM != nil {
				fmt.Printf("failed to shuffle peers: %v\n", errM)
				mp.pView.RemoveDescriptor(desc)
			} else {
				mp.pView.MergeViews(reply.GetNodes())
			}
		}
	}
}

func (mp *MembershipProtocol) SetPartialView(view *m.PartialView) {
	if mp.pView == nil {
		mp.pView = view
	}
}
