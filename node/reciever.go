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

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", iport))
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
	if node.state == DEAD {
		return nil, fmt.Errorf("AppendEntries Error. Node is DEAD")
	}

	node.l.Log(node.id, fmt.Sprintf("Rec. RequestVotes from %d term %d", req.RaftNodeId, req.Term))

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
	if node.state == DEAD {
		node.mu.Unlock()
		return nil, fmt.Errorf("AppendEntries Error. Node is DEAD")
	}
	term := node.term
	node.mu.Unlock()

	node.l.Log(node.id, fmt.Sprintf("Rec. appendEntries from %d term %d key %d value %s", req.LeaderId, req.Term, req.Key, req.Value))

	res := AppendEntriesResponse{}

	if req.Term < node.term {
		res.Term = node.term
		res.Success = false
		return &res, nil
	}

	if req.Term >= term {
		if node.getState() != FOLLOWER {
			node.becomeFollower(req.Term)
		} else {
			node.l.Log(node.id, "setting timer to be reset")
			node.resetTimerEvent(req.Term)
		}
	}

	node.dataMu.Lock()
	if node.commitIndex < req.LeaderCommitIndex && int32(len(node.log) - 1) >= req.LeaderCommitIndex {
		node.commitIndex = req.LeaderCommitIndex
		if node.commitIndex > node.lastApplied {
			node.commitIndex = req.LeaderCommitIndex
			node.CatchupCommits()
		}
	}
	node.dataMu.Unlock()

	// Just return if heartbeat
	if req.Key == 0 {
		res.Term = term
		res.Success = true
		return &res, nil
	}

	node.dataMu.Lock()
	// 	Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)
	if req.PrevLogIndex >= int32(len(node.log)) || node.log[req.PrevLogIndex].term != req.PrevLogTerm {
		res.Term = term
		res.Success = false
		node.dataMu.Unlock()
		return &res, nil
	}

	// If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it (§5.3)
	// Here we can delete all after prevlogindex
	node.log = node.log[:req.PrevLogIndex+1]

	// Append any new entries not already in the log
	node.AddLogEntry(req.Key, req.Value, req.Term)

	// If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if node.commitIndex < req.LeaderCommitIndex {
		if req.LeaderCommitIndex > int32(len(node.log)) - 1 {
			node.commitIndex = int32(len(node.log)) - 1 
		} else {
			node.commitIndex = req.LeaderCommitIndex
		}
	} 

	node.dataMu.Unlock()

	res.Term = term
	res.Success = true
	return &res, nil
}
