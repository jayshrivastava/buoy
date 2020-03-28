package node

import (
	"context"
	"fmt"
	api "github.com/jayshrivastava/buoy/api"
	"google.golang.org/grpc"
	"net"
)

type apiServer struct {
	api.UnimplementedApiServer
	node *raftNode
}

func RunApiServer(port string, node *raftNode) {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		fmt.Printf("Could not start api server on port %s\n", port)
		return
	}

	server := grpc.NewServer()
	api.RegisterApiServer(server, &apiServer{node: node})
	server.Serve(lis)
}

func (s *apiServer) AddEntry(context context.Context, req *api.AddEntryRequest) (*api.AddEntryResponse, error) {
	s.node.mu.Lock()
	defer s.node.mu.Unlock()
	res := api.AddEntryResponse{}

	if s.node.state != LEADER {
		res.Success = false
		return &res, nil
	}

	if req.Key == 0 || req.Value == "" {
		s.node.l.Log(s.node.id, "Recieved ping from client")
	} else {
		s.node.l.Log(s.node.id, fmt.Sprintf("Got request %d=%s from client", req.Key, req.Value))
	}
	res.Success = true
	return &res, nil
}
