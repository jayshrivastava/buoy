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
	node := s.node
	node.mu.Lock()
	defer node.mu.Unlock()
	res := api.AddEntryResponse{}

	if s.node.state != LEADER {
		res.Success = false
		return &res, nil
	}

	if req.Key == 0 || req.Value == "" {
		node.l.Log(node.id, "Recieved ping from client")
	} else {
		node.l.Log(node.id, fmt.Sprintf("Got request %d=%s from client", req.Key, req.Value))

		node.dataMu.Lock()

		node.AddLogEntry(req.Key, req.Value, node.term)

		for _, peerId := range node.externalNodeIds {
			returnMsg, term := node.sender.appendEntries(peerId, node.term, node.id, int32(len(node.log)-2), node.log[len(node.log)-2].term, node.lastApplied, req.Key, req.Value)
			if returnMsg == AE_TERM_OUT_OF_DATE {
				node.dataMu.Unlock()
				node.becomeFollower(term)
				res.Success = false
				return &res, nil
			} else {
				node.nextIndex[peerId] = int32(len(node.log) - 1)
				node.matchIndex[peerId] = int32(len(node.log) - 1)
			}
		}
		node.dataMu.Unlock()

	}
	res.Success = true
	return &res, nil
}
