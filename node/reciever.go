package node

import(
	"google.golang.org/grpc"
	"net"
	"fmt"
	"context"
)

type raftReceiver struct{
	UnimplementedRaftServer
	node *raftNode
}


func RunRaftReceiver (iport string, node *raftNode) {

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", iport))
	if err != nil {
		fmt.Printf("Could not start api server on port %s\n", iport)
		return
	}

	server := grpc.NewServer()
	RegisterRaftServer(server, &raftReceiver{node: node})
	server.Serve(lis)
}

func (receiver *raftReceiver) RequestVotes(ctx context.Context, req *RequestVotesRequest) (*RequestVotesResponse, error) {
	// receiver.node.mu.Lock()
	// defer receiver.node.mu.Unlock()

	// res := RequestVotesResponse{}
	// if req.Term > receiver.node.term {
	// 	res.Term = receiver.node.term
	// 	res.VoteGranted = true
	// 	return &res, nil
	// }
	// if req.Term == receiver.node.term && req.LastLogIndex <= receiver.node.lastLogIndex && req.LastLogTerm <= receiver.node.lastLogTerm  {
	// 	res.Term = receiver.node.term
	// 	res.VoteGranted = true
	// 	return &res, nil
	// }

	// res.Term = receiver.node.term 
	// res.VoteGranted = false
	// return &res, nil
	return &RequestVotesResponse{}, nil
}
func (receiver *raftReceiver) AppendEntries(ctx context.Context, req *AppendEntriesRequest) (*AppendEntriesResponse, error) {
	return &AppendEntriesResponse{}, nil
}
