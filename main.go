// started with code from: https://github.com/chifflier/nfqueue-go/blob/master/nfqueue/test_nfqueue_gopacket/test_nfqueue.go

// how to install : go install . # use it inside mane_go_nfqueue folder
// net config     : sudo ./netconfig.sh
// how to run     : sudo $GOPATH/bin/mane_go_nfqueue

package main

import (
	// "encoding/hex"
	"fmt"
	"mane_go_nfqueue/nfqueue"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
	// "reflect"
	"bufio"
	"bytes"
	"net/http"
	// "strings"
)

func real_callback(payload *nfqueue.Payload) int {

	// global variables
	var srcIP net.IP
	var dstIP net.IP
	// var srcPort layers.TCPPort
	// var dstPort layers.TCPPort

	fmt.Printf("PKT [%03d] ", payload.Id)
	// fmt.Printf("------------------------------------------------------------------------\n id: %d\n", payload.Id)
	// fmt.Println(hex.Dump(payload.Data))
	// Decode a packet
	packet := gopacket.NewPacket(payload.Data, layers.LayerTypeIPv4, gopacket.Default)

	// Get the IP layer from this packet
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		// Get actual IP data from this layer
		ip, _ := ipLayer.(*layers.IPv4)
		srcIP = ip.SrcIP
		dstIP = ip.DstIP
		fmt.Printf("%15s > %-15s ", srcIP, dstIP)
	}

	if icmpLayer := packet.Layer(layers.LayerTypeICMPv4); icmpLayer != nil {
		fmt.Print("\t√\tICMP\n")
		payload.SetVerdict(nfqueue.NF_ACCEPT)
		return 0
	}

	if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
		fmt.Print("\t√\tARP\n")
		payload.SetVerdict(nfqueue.NF_ACCEPT)
		return 0
	}

	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		fmt.Print("\t√\tUDP\n")
		payload.SetVerdict(nfqueue.NF_ACCEPT)
		return 0
	}

	// Get the TCP layer from this packet
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {

		// Get actual TCP data from this layer
		tcp, _ := tcpLayer.(*layers.TCP)
		// srcPort = tcp.SrcPort
		// dstPort = tcp.DstPort
		// fmt.Printf("%s:%d > %s:%d ", srcIP, srcPort, dstIP, dstPort)
		// fmt.Printf("%15s > %-15s ", srcIP, dstIP)

		//
		// if there is payload...
		// try to read it and see if it is an http request
		//
		if len(tcp.Payload) != 0 {

			reader := bufio.NewReader(bytes.NewReader(tcp.Payload))
			httpReq, err := http.ReadRequest(reader)

			if err != nil {
				//
				// if the Payload could not be parsed as an HTTP request
				// forward the packet...
				//

				//fmt.Print(err)
				fmt.Print("\t√\tTCP ")
				payload.SetVerdict(nfqueue.NF_ACCEPT)
				if tcp.SYN {
					fmt.Print("[SYN] ")
				}
				if tcp.ACK {
					fmt.Print("[ACK] ")
				}
				if tcp.FIN {
					fmt.Print("[FIN] ")
				}
			} else {
				//
				// if the payload contained http request
				//

				fmt.Print("\t√\tTCP ")
				payload.SetVerdict(nfqueue.NF_ACCEPT)
				if tcp.SYN {
					fmt.Print("[SYN] ")
				}
				if tcp.ACK {
					fmt.Print("[ACK] ")
				}
				if tcp.FIN {
					fmt.Print("[FIN] ")
				}

				fmt.Printf("[HTTP] REQUEST %s", httpReq.URL.String())

			}
		} else {
			//
			// if there was no payload
			// forward the packet...
			//
			fmt.Print("\t√\tTCP ")
			if tcp.SYN {
				fmt.Print("[SYN] ")
			}
			if tcp.ACK {
				fmt.Print("[ACK] ")
			}
			if tcp.FIN {
				fmt.Print("[FIN] ")
			}
			payload.SetVerdict(nfqueue.NF_ACCEPT)
		}

		fmt.Print("\n")

		return 0

	}

	fmt.Printf("\t√\tOTHER\n")
	payload.SetVerdict(nfqueue.NF_ACCEPT)
	return 0
}

func main() {
	q := new(nfqueue.Queue)

	q.SetCallback(real_callback)

	q.Init()
	defer q.Close()

	q.Unbind(syscall.AF_INET)
	q.Bind(syscall.AF_INET)

	q.CreateQueue(0)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			// sig is a ^C, handle it
			_ = sig
			q.Close()
			os.Exit(0)
			// XXX we should break gracefully from loop
		}
	}()

	// XXX Drop privileges here

	// XXX this should be the loop
	q.TryRun()

	fmt.Printf("exit...\n")
}
