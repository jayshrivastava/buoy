package node

import (
	"google.golang.org/grpc"
	"math/rand"
	"sync"
	"time"
	"fmt"
)

type raftNode struct {
	// Identifier of this Server
	nodeId int32
	// Node state
	state STATE
	// Current Term
	term int32
	// Last Log Index
	lastLogIndex int32
	// Last Log Index
	lastLogTerm int32
	// Event to reset the timer or indivate timer expiry
	timerEvent chan TIMEREVENT
	// Lock - applies to term
	mu sync.Mutex
	// RPC Client
	sender RaftSender
	// logger
	l Logger
}

type RaftNode interface {
	runElectionTimer()
	beginElection()
	generateElectionTimeout()
	becomeFollower(newTerm int32)
	becomeLeader()
	appendToLog()
	Run()
}

func RunRaftNode(nodeId int32, port string, iport string, iports []string, wg *sync.WaitGroup, state STATE) {
	defer wg.Done()

	node := raftNode{
		nodeId: nodeId,
		state: state,
		term: 0,
		lastLogIndex: 0,
		lastLogTerm: 0,
		timerEvent: make(chan TIMEREVENT),
		mu: sync.Mutex{},
		// Sender not implemented until receivers are running
		l: CreateLogger(nodeId),
	}

	go RunApiServer(port, &node)
	go RunRaftReceiver(iport, &node)

	rpcClients := []RaftClient{}
	for _, iport := range(iports) {
		conn, _ := grpc.Dial(fmt.Sprintf("localhost:%s", iport), grpc.WithInsecure())
		rpcClients = append(rpcClients, NewRaftClient(conn))
	}
	node.sender = CreateRaftSender(rpcClients)

	node.Run()
	
}

func (node *raftNode) Run() {
	for {
		time.Sleep(3*time.Second)

		node.l.Log("running")
	}
}

func (node *raftNode) generateElectionTimeout() time.Duration {
	return time.Duration(MIN_ELECTION_TIMEOUT+rand.Intn(MAX_ELECTION_TIMEOUT-MIN_ELECTION_TIMEOUT)) * time.Millisecond
}

func (node *raftNode) runElectionTimer() {
	timeout := node.generateElectionTimeout()
	node.mu.Lock()
	termStarted := node.term
	node.mu.Unlock()

	for {
		timer := time.NewTimer(timeout)
		go func() {
			<-timer.C
			node.timerEvent <- EXPIRED
		}()
		timerEvent := <-node.timerEvent

		node.mu.Lock()
		// Stop election timer since we were elected as leader
		if node.state != CANDIDATE && node.state != FOLLOWER {
			node.mu.Unlock()
			return
		}

		// Stop election since a new term means a new leader was elected
		if termStarted != node.term {
			node.mu.Unlock()
			return
		}

		// If the timer expired, trigger an election
		if timerEvent == EXPIRED {
			// node.StartElection()
			node.mu.Unlock()
			return
		}
		// The timer must have been reset by another thread, so we can reset the timer and loop again
		node.mu.Unlock()
	}
}

// Guarenteed to be called when runElectionTimer is not running
func (node *raftNode) beginElection() {
	node.state = CANDIDATE
	node.term += 1

	currentTerm := node.term
	currentLastLogIndex := node.lastLogIndex
	currentLastLogTerm := node.lastLogTerm

	result, newTerm := node.sender.requestVotes(currentTerm, node.nodeId, currentLastLogIndex, currentLastLogTerm)
	node.mu.Lock()
	// Another node was elected leader
	if node.state != CANDIDATE {
		return
	}
	switch result {
	case RV_TERM_OUT_OF_DATE:
		node.becomeFollower(newTerm)
		node.mu.Unlock()
		return
	case MAJORITY:
		node.becomeLeader()
		node.mu.Unlock()
		return
	case SPLIT:
	case LOST:
	}
	return
}

// LOCK MUST BE ACQUIRED
func (node *raftNode) becomeFollower(newTerm int32) {
	node.state = FOLLOWER
	node.term = newTerm

	go node.runElectionTimer()
}

// LOCK MUST BE ACQUIRED
func (node *raftNode) becomeLeader() {
	nodeId := node.nodeId
	node.state = LEADER
	savedCurrentTerm := node.term

	go func() {
		timer := time.NewTimer(HEARTBEAT_INTERVAL * time.Millisecond)
		defer timer.Stop()
		for {
			<-timer.C
			result, term := node.sender.appendEntries(savedCurrentTerm, nodeId, 0, 0, 0, make(map[int32]string))
			switch result {
			case AE_TERM_OUT_OF_DATE:
				node.mu.Lock()
				node.becomeFollower(term)
				node.mu.Unlock()
				return
			case SUCCESS:
			case FAILIURE:
			}
		}
	}()
}

func (node *raftNode) appendToLog(){}

/* CONSTANTS */

// Raft Node State
type STATE int

const (
	LEADER STATE = iota
	FOLLOWER
	CANDIDATE
)

// Timers
const MIN_ELECTION_TIMEOUT = 150
const MAX_ELECTION_TIMEOUT = 300
const HEARTBEAT_INTERVAL = 50

type TIMEREVENT int

const (
	RESET TIMEREVENT = iota
	EXPIRED
)
