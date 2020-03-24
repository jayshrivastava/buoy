package node

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"net"
)

type raftReceiver struct {
	UnimplementedRaftServer
	node *raftNode
}

func RunRaftReceiver(iport string, node *raftNode) {

	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", iport))
	if err != nil {
		fmt.Printf("Could not start api server on port %s\n", iport)
		return
	}

	server := grpc.NewServer()
	RegisterRaftServer(server, &raftReceiver{node: node})
	server.Serve(lis)
}

func (receiver *raftReceiver) RequestVotes(ctx context.Context, req *RequestVotesRequest) (*RequestVotesResponse, error) {
	node := receiver.node
	node.mu.Lock()
	defer node.mu.Unlock()

	node.l.Log(fmt.Sprintf("Was requested vote from %d with term %d", req.RaftNodeId, req.Term))

	res := RequestVotesResponse{}

	if req.Term < node.term {
		res.Term = node.term
		res.VoteGranted = false
		return &res, nil
	}

	if node.votedFor == -1 {
		if req.LastLogTerm > node.log[int32(len(node.log)-1)].term || req.LastLogIndex >= int32(len(receiver.node.log)-1) {
			node.votedFor = req.RaftNodeId
			res.Term = node.term
			res.VoteGranted = true
			return &res, nil
		}
	}

	res.Term = receiver.node.term
	res.VoteGranted = false
	return &res, nil
}
func (receiver *raftReceiver) AppendEntries(ctx context.Context, req *AppendEntriesRequest) (*AppendEntriesResponse, error) {
	node := receiver.node
	node.mu.Lock()
	term := node.term
	node.mu.Unlock()

	node.l.Log(fmt.Sprintf("Recieved appendEntries req from (leader) %d term %d", req.LeaderId, req.Term))

	res := AppendEntriesResponse{}

	// heartbeat
	if len(req.Entries) == 0 {
		if req.Term >= term {
			if node.getState() != FOLLOWER {
				node.becomeFollower(req.Term)
			} else {
				node.l.Log("setting timer to be reset")
				node.resetTimerEvent(req.Term)
			}
			res.Term = term
			res.Success = true
			return &res, nil
		}
	}

	res.Term = term
	res.Success = true
	return &res, nil
}
