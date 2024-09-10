package model

import (
	"flag"
	"time"
)

var (
	// SpaceType = 1 for EuclideanSpace
	// SpaceType = 2 for HeightVectorEuclideanSpace
	SpaceType            = 1
	InstanceSpace  Space = EuclideanSpace{}
	Location, _          = time.LoadLocation("Europe/Rome")
	MembershipPort       = flag.Uint("membership_port", 50152, "Membership server port")
	VivaldiPort          = flag.Uint("vivaldi_port", 50153, "Vivaldi server port")
	GossipPort           = flag.Uint("gossip_port", 50154, "Gossip server port")

	LoggingEnv           = "LOGGING"
	LoggingResultEnv     = "RESULT_LOGGING"
	LoggingMembershipEnv = "MEMBERSHIP_LOGGING"
	LoggingVivaldiEnv    = "VIVALDI_LOGGING"
	LoggingGossipEnv     = "GOSSIPING_LOGGING"
)
