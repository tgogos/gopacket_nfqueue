# gopacket_nfqueue
Packet inspection with gopacket and nfqueue...

##Test environment
Virtualbox with three Ubuntu VMs set up like this:
```
 +-----------+       +-------------------+       +-----------+
 | client VM |-------| pkt inspection VM |-------| server VM |
 +-----------+       +-------------------+       +-----------+
           eth1    eth1                 eth2    eth1

client eth1: 192.168.2.X
server eth1: 192.168.3.X

pkt inspection eth1: 192.168.2.4
pkt inspection eth2: 192.168.3.4
```
###Client configuration:



 
 
