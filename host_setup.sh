tc qdisc add dev eth0 root netem delay 9ms
exec ./sdcc_host