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
	hosts []string
}

type BuoyClient interface {
	findLeader() (string, error)
	Run()
}

func CreateBuoyClient(ports []string) (BuoyClient, error) {

	for i, port := range ports {
		ports[i] = fmt.Sprintf("127.0.0.1:%s", port)
	}
	bc := buoyClient{
		hosts: ports,
	}

	return &bc, nil
}

func (bc *buoyClient) findLeader() (string, error) {

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

	req := api.AddEntryRequest{}
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
	return leader, nil
}

func (bc *buoyClient) Run() {
	host, err := bc.findLeader()
	for err != nil {
		fmt.Println("ASdfasfijasij")
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
			fmt.Println(strconv.Atoi(kv[0]))
			if i, err := strconv.Atoi(kv[0]); err == nil {
				req := api.AddEntryRequest{
					Key:   int32(i),
					Value: kv[1],
				}
				res, err := client.AddEntry(context.Background(), &req)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(res)
				}
			}

			fmt.Println("text")
		}
	}
}
