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

func (s *apiServer) Kill(context context.Context, req *api.KillRequest) (*api.KillResponse, error) {
	res := api.KillResponse{}
	node := s.node
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.state == DEAD {
		return nil, fmt.Errorf("Node is already dead")
	}

	node.l.Log(node.id, "This Node is Dead")
	node.state = DEAD

	return &res, nil
}
func (s *apiServer) Revive(context context.Context, req *api.ReviveRequest) (*api.ReviveResponse, error) {
	res := api.ReviveResponse{}
	node := s.node
	node.mu.Lock()

	if node.state == DEAD {
		node.mu.Unlock()
		node.l.Log(node.id, "This Node has been Revived")

		node.becomeFollower(0)
		return &res, nil
	}

	return nil, fmt.Errorf("Node is not dead, could not revive it")
}

func (s *apiServer) AddEntry(context context.Context, req *api.AddEntryRequest) (*api.AddEntryResponse, error) {
	node := s.node
	node.mu.Lock()
	defer node.mu.Unlock()
	res := api.AddEntryResponse{}

	if s.node.state != LEADER {
		res.Success = false
		return &res, fmt.Errorf("Not leader")
	}

	if req.Key == 0 || req.Value == "" {
		node.l.Log(node.id, "Recieved Ping from Client")
	} else {
		node.l.Log(node.id, fmt.Sprintf("Got Request from Client to Add Entry {key: %d, value: %s}", req.Key, req.Value))

		node.dataMu.Lock()

		node.AddLogEntry(req.Key, req.Value, node.term)

		for _, peerId := range node.externalNodeIds {
			returnMsg, term := node.sender.appendEntries(peerId, node.term, node.id, int32(len(node.log)-2), node.log[len(node.log)-2].term, node.lastApplied, req.Key, req.Value)
			if returnMsg == AE_TERM_OUT_OF_DATE {
				node.dataMu.Unlock()
				node.becomeFollower(term)
				res.Success = false
				return &res, nil
			} else if returnMsg == FAILIURE {
				fromLast := 3
				node.l.Log(node.id, fmt.Sprintf("Node %d Reported a Failiure when Appending Entry. Catching Up Entries...", peerId))

				for returnMsg, term = node.sender.appendEntries(peerId, node.term, node.id, int32(len(node.log)-fromLast), node.log[len(node.log)-fromLast].term, node.lastApplied, node.log[len(node.log)-fromLast+1].key, node.log[len(node.log)-fromLast+1].value); returnMsg != SUCCESS; {
					if returnMsg == AE_TERM_OUT_OF_DATE {
						node.dataMu.Unlock()
						node.becomeFollower(term)
						res.Success = false
						return &res, nil
					}
					fromLast += 1
				}
				fromLast -= 1
				node.matchIndex[peerId] = int32(len(node.log) - fromLast)

				for {
					returnMsg, term = node.sender.appendEntries(peerId, node.term, node.id, int32(len(node.log)-fromLast), node.log[len(node.log)-fromLast].term, node.lastApplied, node.log[len(node.log)-fromLast+1].key, node.log[len(node.log)-fromLast+1].value)
					if returnMsg == AE_TERM_OUT_OF_DATE {
						node.dataMu.Unlock()
						node.becomeFollower(term)
						res.Success = false
						return &res, nil
					} else if returnMsg == SUCCESS {
						node.matchIndex[peerId] = int32(len(node.log) - 1)
					}
					fromLast -= 1
					if fromLast <= 2 {
						break
					}
				}
				node.l.Log(node.id, fmt.Sprintf("Node %d Caught Up", peerId))

			} else if returnMsg == SUCCESS {
				node.matchIndex[peerId] = int32(len(node.log) - 1)
			}
		}

		// If there exists an N such that N > commitIndex, a majority
		// of matchIndex[i] ≥ N, and log[N].term == currentTerm:
		// set commitIndex = N (§5.3, §5.4).
		for _, mi := range node.matchIndex {
			if mi > node.commitIndex {
				count := 0
				for _, mi2 := range node.matchIndex {
					if mi <= mi2 {
						count += 1
					}
				}
				if count >= len(node.externalNodeIds)/2 && node.log[mi].term == node.term {
					node.commitIndex = mi
					node.CatchupCommits()
				}
			}
		}

		node.dataMu.Unlock()

	}
	res.Success = true
	return &res, nil
}
