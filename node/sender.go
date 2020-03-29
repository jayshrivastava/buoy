package node

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"sync"
	"time"
)

// Current assumption is that the number of nodes int he cluster will not change
type sender struct {
	rpcClients map[int32]RaftClient
	node       *raftNode
}

func CreateRaftSender(hosts map[int32]string, node *raftNode) RaftSender {
	rpcClients := map[int32]RaftClient{}
	for externalNodeId, host := range hosts {
		conn, _ := grpc.Dial(host, grpc.WithInsecure())
		rpcClients[externalNodeId] = NewRaftClient(conn)
	}
	return &sender{rpcClients: rpcClients, node: node}
}

type RaftSender interface {
	requestVotes(term int32, raftNodeId int32, lastLogIndex int32, lastLogTerm int32) (REQUEST_VOTES_RETURN_TYPE, int32)
	heartbeat(term int32, leaderId int32) (APPEND_ENTRIES_RETURN_TYPE, int32)
	appendEntries(to int32, term int32, leaderId int32, prevLogIndex int32, prevLogTerm int32, leaderCommitIndex int32, key int32, value string) (APPEND_ENTRIES_RETURN_TYPE, int32)
}

func (client *sender) requestVotes(term int32, raftNodeId int32, lastLogIndex int32, lastLogTerm int32) (REQUEST_VOTES_RETURN_TYPE, int32) {
	node := client.node
	node.l.Log(node.id, fmt.Sprintf("Sending requestVotes"))

	votesReceived := 1
	maxTermSeen := term
	// use go routines here to send the requests in parallel
	responses := []*RequestVotesResponse{}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for sendingTo, nodeClient := range client.rpcClients {
		req := RequestVotesRequest{
			Term:         term,
			RaftNodeId:   raftNodeId,
			LastLogIndex: lastLogIndex,
			LastLogTerm:  lastLogTerm,
		}
		wg.Add(1)

		go func(nodeClient RaftClient, sendingTo int32) {
			defer wg.Done()
			node.l.Log(node.id, fmt.Sprintf("Sent requestVotes to %d", sendingTo))
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			response, err := nodeClient.RequestVotes(ctx, &req)
			if err == nil {
				node.l.Log(node.id, fmt.Sprintf("Got requestVotes response %t from %d", response.VoteGranted, sendingTo))
				mu.Lock()
				responses = append(responses, response)
				mu.Unlock()
			}
			if err != nil {
				node.l.Log(node.id, err.Error())
			}
		}(nodeClient, sendingTo)
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

	node.l.Log(node.id, fmt.Sprintf("Rec. %d votes", votesReceived))

	if votesReceived > len(client.rpcClients)/2 {
		return MAJORITY, term
	}

	if len(client.rpcClients)%2 == 1 && votesReceived == len(client.rpcClients)/2 {
		return SPLIT, term
	}

	return LOST, maxTermSeen
}

func (client *sender) heartbeat(term int32, leaderId int32) (APPEND_ENTRIES_RETURN_TYPE, int32) {

	maxTermSeen := term
	responses := []*AppendEntriesResponse{}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	client.node.dataMu.Lock()
	ci := client.node.commitIndex
	client.node.dataMu.Unlock()

	for _, nodeClient := range client.rpcClients {
		req := AppendEntriesRequest{
			Term:              term,
			LeaderId:          leaderId,
			PrevLogIndex:      0,
			PrevLogTerm:       0,
			LeaderCommitIndex: ci,
			Key:               0,
			Value:             "",
		}
		wg.Add(1)

		go func(nodeClient RaftClient) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if response, err := nodeClient.AppendEntries(ctx, &req); err == nil {
				mu.Lock()
				responses = append(responses, response)
				mu.Unlock()
			} else {
				client.node.l.Log(client.node.id, err.Error())
			}
		}(nodeClient)
	}
	wg.Wait()

	for _, response := range responses {
		if response.Term > maxTermSeen {
			maxTermSeen = response.Term
		}
	}

	if maxTermSeen > term {
		return AE_TERM_OUT_OF_DATE, 1
	}

	return SUCCESS, term
}

func (client *sender) appendEntries(to int32, term int32, leaderId int32, prevLogIndex int32, prevLogTerm int32, leaderCommitIndex int32, key int32, value string) (APPEND_ENTRIES_RETURN_TYPE, int32) {

	nodeClient := client.rpcClients[to]
	req := AppendEntriesRequest{
		Term:              term,
		LeaderId:          leaderId,
		PrevLogIndex:      prevLogIndex,
		PrevLogTerm:       prevLogTerm,
		LeaderCommitIndex: leaderCommitIndex,
		Key:               key,
		Value:             value,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	response, err := nodeClient.AppendEntries(ctx, &req)
	if err != nil {
		client.node.l.Log(leaderId, err.Error())
		return SUCCESS, term
	}

	if response.Term > term {
		return AE_TERM_OUT_OF_DATE, 1
	}

	if response.Success == false {
		return FAILIURE, term
	}

	return SUCCESS, term
}

/* CONSTANTS */
type REQUEST_VOTES_RETURN_TYPE int

const (
	LOST REQUEST_VOTES_RETURN_TYPE = iota
	MAJORITY
	SPLIT
)

func (t REQUEST_VOTES_RETURN_TYPE) String() string {
	return [...]string{"LOST", "Majority", "Split"}[t]
}

/* CONSTANTS */
type APPEND_ENTRIES_RETURN_TYPE int

const (
	AE_TERM_OUT_OF_DATE APPEND_ENTRIES_RETURN_TYPE = iota
	SUCCESS
	FAILIURE
)

func (t APPEND_ENTRIES_RETURN_TYPE) String() string {
	return [...]string{"TERM OUT OF DATE", "SUCCESS", "FAILIURE"}[t]
}
