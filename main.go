package main

import (
	"flag"
	"fmt"
	"github.com/jayshrivastava/buoy/client"
	"github.com/jayshrivastava/buoy/node"
	"google.golang.org/grpc"
	"net"
	"strings"
	"sync"
)

func logError(e error) {
	fmt.Printf("%v\n", e)
}

func testPorts(ports []string) bool {
	// Test that ports are unique
	ports_h := map[string]bool{}
	for _, port := range ports {
		ports_h[port] = true
	}
	if len(ports) != len(ports_h) {
		logError(fmt.Errorf("ports are not unique"))
		return false
	}

	// Tests that all ports can be dialed
	for _, port := range ports {
		_, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%s", port), grpc.WithInsecure())
		if err != nil {
			logError(fmt.Errorf("Could not connect to every ports"))
			return false
		}
	}
	return true
}

func testIPorts(iports []string) bool {
	// Test that iports are unique
	iports_h := map[string]bool{}
	for _, port := range iports {
		iports_h[port] = true
	}
	if len(iports) != len(iports_h) {
		logError(fmt.Errorf("Ports are not unique"))
		return false
	}
	// Test that iports are available
	for _, port := range iports {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			logError(err)
			return false
		}
		lis.Close()
	}
	return true
}

func main() {

	iports := flag.String("iports", "", "CSV values of ports for inter node communication")
	ports := flag.String("ports", "", "CSV values of ports for clients to communicate with the cluster ")
	runAsClient := flag.Bool("client", false, "Run as client if flag is present, else run cluster")

	flag.Parse()

	if !*runAsClient {
		if *iports == "" || *ports == "" {
			logError(fmt.Errorf("No ports nor iports provided for cluster"))
			return
		}

		iports := strings.Split(*iports, ",")
		ports := strings.Split(*ports, ",")
		if !testIPorts(iports) {
			return
		} // terminate if iports are invalid
		if !testPorts(ports) {
			return
		} // terminate if iports are invalid
		if len(iports) != len(ports) {
			logError(fmt.Errorf("The number of ports does not match the number of iports"))
		}

		wg := sync.WaitGroup{}

		for i := int32(0); i < int32(len(ports)); i++ {
			otherIPorts := map[int32]string{}
			for j, iport := range iports {
				if iport != iports[i] {
					otherIPorts[int32(j)] = iport
				}
			}
			state := node.FOLLOWER
			// if i == 0 {
			// 	state = node.LEADER
			// }
			wg.Add(1)
			go node.RunRaftNode(i, ports[i], iports[i], otherIPorts, &wg, state)
		}
		wg.Wait()

	} else {
		if *ports == "" {
			logError(fmt.Errorf("No node addresses provided for client"))
			return
		}

		ports := strings.Split(*ports, ",")
		if !testPorts(ports) {
			return
		} // terminate if ports are invalid

		client, err := client.CreateBuoyClient(ports)
		if err != nil {
			logError(err)
			return
		}
		client.Run()
	}
}
