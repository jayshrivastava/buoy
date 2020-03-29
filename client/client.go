package client

import (
	"bufio"
	"context"
	"fmt"
	api "github.com/jayshrivastava/buoy/api"
	"google.golang.org/grpc"
	"os"
	"strconv"
	"strings"
	"sync"
)

type ClientConfig struct {
	NodeHosts map[int32]string
}

type buoyClient struct {
	hosts map[int32]string
}

type BuoyClient interface {
	findLeader() (string, error)
	Run()
}

func CreateBuoyClient(cfg ClientConfig) (BuoyClient, error) {

	bc := buoyClient{
		hosts: cfg.NodeHosts,
	}

	return &bc, nil
}

func (bc *buoyClient) findLeader() (string, error) {

	fmt.Println("Searching for leader")
	type hcr struct {
		host     string
		client   api.ApiClient
		response *api.AddEntryResponse
	}
	hcrs := []hcr{}

	for _, host := range bc.hosts {
		data := hcr{
			host:     host,
			client:   nil,
			response: nil,
		}

		conn, err := grpc.Dial(host, grpc.WithInsecure())
		defer conn.Close()
		if err == nil {
			data.client = api.NewApiClient(conn)
			hcrs = append(hcrs, data)
		}
	}

	req := api.AddEntryRequest{Key: 0, Value: ""}
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}

	for i, data := range hcrs {
		if data.client != nil {
			wg.Add(1)
			go func(i int, data hcr) {
				defer wg.Done()
				if response, err := data.client.AddEntry(context.Background(), &req); err == nil {
					mu.Lock()
					hcrs[i].response = response
					mu.Unlock()
				}
			}(i, data)
		}
	}
	wg.Wait()

	leader := ""
	for _, data := range hcrs {
		if data.response != nil && data.response.Success {
			leader = data.host
		}
	}

	if leader == "" {
		return leader, fmt.Errorf("Could not find leader")
	}
	fmt.Println("Found leader")
	return leader, nil
}

func (bc *buoyClient) Run() {
	host, err := bc.findLeader()
	for err != nil {
		host, err = bc.findLeader()
	}
	conn, _ := grpc.Dial(host, grpc.WithInsecure())
	defer conn.Close()
	client := api.NewApiClient(conn)

	for {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, _ := reader.ReadString('\n')
			text = strings.TrimSuffix(text, "\n")
			kv := strings.Split(text, "=")
			if i, err := strconv.Atoi(kv[0]); err == nil {
				req := api.AddEntryRequest{
					Key:   int32(i),
					Value: kv[1],
				}
				res, err := client.AddEntry(context.Background(), &req)
				if err != nil {
					fmt.Println(err.Error())
					host, err = bc.findLeader()
					for err != nil {
						host, err = bc.findLeader()
					}
				} else {
					fmt.Println("Response status: ", res.Success)
				}
			}
		}
		
	}
}
