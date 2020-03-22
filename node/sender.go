package node

import (
	"context"
	"sync"
)

// Current assumption is that the number of nodes int he cluster will not change
type sender struct {
	rpcClients []RaftClient
}

func CreateRaftSender(rpcClients []RaftClient) RaftSender {
	return &sender{rpcClients: rpcClients}
}

type RaftSender interface {
	requestVotes(term int32, raftNodeId int32, lastLogIndex int32, lastLogTerm int32) (REQUEST_VOTES_RETURN_TYPE, int32)
	appendEntries(term int32, leaderId int32, prevLogIndex int32, prevLogTerm int32, leaderCommitIndex int32, entries map[int32]string) (APPEND_ENTRIES_RETURN_TYPE, int32)
}

func (client *sender) requestVotes(term int32, raftNodeId int32, lastLogIndex int32, lastLogTerm int32) (REQUEST_VOTES_RETURN_TYPE, int32) {
	votesReceived := 1
	maxTermSeen := term
	// use go routines here to send the requests in parallel
	responses := []*RequestVotesResponse{}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, nodeClient := range client.rpcClients {
		req := RequestVotesRequest{
			Term:         term,
			RaftServerId: raftNodeId,
			LastLogIndex: lastLogIndex,
			LastLogTerm:  lastLogTerm,
		}
		wg.Add(1)

		go func(nodeClient RaftClient) {
			defer wg.Done()
			if response, err := nodeClient.RequestVotes(context.Background(), &req); err == nil {
				mu.Lock()
				responses = append(responses, response)
				mu.Unlock()
			}
		}(nodeClient)
	}
	wg.Wait()
	for _, response := range responses {
		if response.Term > maxTermSeen {
			maxTermSeen = response.Term
		}
		if response.VoteGranted {
			votesReceived += 1
		}
	}
	if maxTermSeen > term {
		return RV_TERM_OUT_OF_DATE, maxTermSeen
	}

	if votesReceived > len(client.rpcClients)/2 {
		return MAJORITY, term
	}

	if len(client.rpcClients)%2 == 0 && votesReceived == len(client.rpcClients)/2 {
		return SPLIT, term
	}

	return LOST, term
}

func (client *sender) appendEntries(term int32, leaderId int32, prevLogIndex int32, prevLogTerm int32, leaderCommitIndex int32, entries map[int32]string) (APPEND_ENTRIES_RETURN_TYPE, int32) {
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
