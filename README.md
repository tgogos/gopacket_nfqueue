# gopacket_nfqueue
Packet inspection with gopacket and nfqueue...

```bash
# how to run...
cd /[golang path]/src/github.com/tgogos/gopacket_nfqueue
go install .
sudo $GOPATH/bin/gopacket_nfqueue
```

##Test environment
Virtualbox with three Ubuntu VMs set up like this:
```
 +-----------+       +-------------------+       +-----------+
 | client VM |-------| pkt inspection VM |-------| server VM |
 +-----------+       +-------------------+       +-----------+
           eth1    eth1                 eth2    eth1

client VM eth1 (host-only): 192.168.4.2
server VM eth1 (host-only): 192.168.5.2

pkt inspection VM eth1 (host-only): 192.168.4.3
pkt inspection VM eth2 (host-only): 192.168.5.3
```
###Client VM configuration:
The client must forward traffic to the `pkt inspection VM eth1` so a route must be added:
```
route add -net 192.168.5.0 netmask 255.255.255.0 gw 192.168.4.3 dev eth1
```

###Server VM configuration:
The server must forward traffic to the `pkt inspection VM eth2` so a route must be added:
```
route add -net 192.168.4.0 netmask 255.255.255.0 gw 192.168.5.3 dev eth1
```

###Packet inspection VM configuration:
The following commands set up the packet forwarding and the routing rules in order to use NFQUEUE:
```
sysctl -w net.ipv4.ip_forward=1
iptables -t raw -A PREROUTING -i eth1 -j NFQUEUE --queue-num 0
iptables -t raw -A PREROUTING -i eth2 -j NFQUEUE --queue-num 0
```
