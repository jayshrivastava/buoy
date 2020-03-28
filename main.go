package main

import (
	"flag"
	"fmt"
	"github.com/jayshrivastava/buoy/client"
	"github.com/jayshrivastava/buoy/node"
	"sync"
)

func main() {

	runAsClient := flag.Bool("client", false, "Run as client if flag is present, else run cluster")
	debug := flag.Bool("debug", false, "Output to log.html if enabled")

	flag.Parse()

	config, err := CreateConfig()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if !*runAsClient {
		logger := CreateLogger(int32(len(config.nodesCfg)), *debug)

		wg := sync.WaitGroup{}
		for _, cfg := range config.nodesCfg {
			wg.Add(1)
			go node.RunRaftNode(cfg, logger)
		}
		wg.Wait()

	} else {

		client, err := client.CreateBuoyClient(config.clientCfg)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		client.Run()
	}
}
