package node

import (
	rpc "github.com/jayshrivastava/buoy/rpc"
)

type client struct {
	rpcClients []rpc.RaftClient
}

type raftClient interface {
	requestVotes(term int32, raftNodeId int32, lastLogIndex int32, lastLogTerm int32) (REQUEST_VOTES_RETURN_TYPE, int32, int32, int32)
	appendEntries(term int32, leaderId int32, prevLogIndex int32, prevLogTerm int32, leaderCommitIndex int32, entries map[int32]string) (APPEND_ENTRIES_RETURN_TYPE, int32)
}

func (client *client) requestVotes(term int32, raftNodeId int32, lastLogIndex int32, lastLogTerm int32) (REQUEST_VOTES_RETURN_TYPE, int32, int32, int32) {
	// votesReceived := 1 // server votes for itself
	// use go routines here to send the requests in parallel
	// for _, rpc := range(client.rpcClients) {

	// }
	return MAJORITY, term, lastLogIndex, lastLogTerm
}
func (client *client) rppendEntries() {}

func (client *client) appendEntries(term int32, leaderId int32, prevLogIndex int32, prevLogTerm int32, leaderCommitIndex int32, entries map[int32]string) (APPEND_ENTRIES_RETURN_TYPE, int32) {
	// votesReceived := 1 // server votes for itself
	// use go routines here to send the requests in parallel
	// for _, rpc := range(client.rpcClients) {

	// }
	return AE_TERM_OUT_OF_DATE, 1
}

/* CONSTANTS */
type REQUEST_VOTES_RETURN_TYPE int

const (
	RV_TERM_OUT_OF_DATE REQUEST_VOTES_RETURN_TYPE = iota
	MAJORITY
	SPLIT
	LOST
)
/* CONSTANTS */
type APPEND_ENTRIES_RETURN_TYPE int

const (
	AE_TERM_OUT_OF_DATE APPEND_ENTRIES_RETURN_TYPE = iota
	SUCCESS
	FAILIURE
)
