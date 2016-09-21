#!/bin/sh
sysctl -w net.ipv4.ip_forward=1
iptables -t raw -A PREROUTING -i eth1 -j NFQUEUE --queue-num 0
iptables -t raw -A PREROUTING -i eth2 -j NFQUEUE --queue-num 0
