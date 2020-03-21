package node

import (
	rpc "github.com/jayshrivastava/buoy/rpc"
)

type client struct {
	rpcClients []rpc.RaftClient
}

type raftClient interface {
	requestVotes(term int32, raftNodeId int32, lastLogIndex int32, lastLogTerm int32) (REQUEST_VOTES_RETURN_TYPE, int32, int32, int32)
	rppendEntries()
}

func (client *client) requestVotes(term int32, raftNodeId int32, lastLogIndex int32, lastLogTerm int32) (REQUEST_VOTES_RETURN_TYPE, int32, int32, int32) {
	// votesReceived := 1 // server votes for itself
	// for _, rpc := range(client.rpcClients) {

	// }
	return MAJORITY, term, lastLogIndex, lastLogTerm
}
func (client *client) rppendEntries() {}

/* CONSTANTS */
type REQUEST_VOTES_RETURN_TYPE int

const (
	TERM_OUT_OF_DATE REQUEST_VOTES_RETURN_TYPE = iota
	MAJORITY
	SPLIT
	LOST
)
