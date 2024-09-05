package model

import (
	"flag"
	"time"
)

var (
	SpaceType            = 1
	InstanceSpace  Space = EuclideanSpace{}
	Location, _          = time.LoadLocation("Europe/Rome")
	MembershipPort       = flag.Uint("membership_port", 50152, "Membership server port")
	VivaldiPort          = flag.Uint("vivaldi_port", 50153, "Vivaldi server port")
	GossipPort           = flag.Uint("gossip_port", 50154, "Gossip server port")
)
