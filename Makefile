.PHONY: host

host:
	go run ./main.go -membership_port 50152 -vivaldi_port 50153 -gossip_port 50154

host1:
	go run ./main.go -membership_port 50155 -vivaldi_port 50156 -gossip_port 50157

host2:
	go run ./main.go -membership_port 50158 -vivaldi_port 50159 -gossip_port 50160

host3:
	go run ./main.go -membership_port 50161 -vivaldi_port 50162 -gossip_port 50163
