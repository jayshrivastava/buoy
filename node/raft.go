package node

import (
	"math/rand"
	"sync"
	"time"
)

type Node struct {
	// Identifier of this Server
	nodeId int32
	// Identifiers all other Servers in the cluster
	peerNodeIds int32
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
	// RPC receiver
	receiver receiver
	// RPC Client
	client client
}

type RaftNode interface {
	runElectionTimer()
	beginElection()
	generateElectionTimeout()
	becomeFollower(newTerm int32)
	becomeLeader()
}

func (node *Node) generateElectionTimeout() time.Duration {
	return time.Duration(MIN_ELECTION_TIMEOUT+rand.Intn(MAX_ELECTION_TIMEOUT-MIN_ELECTION_TIMEOUT)) * time.Millisecond
}

func (node *Node) runElectionTimer() {
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
func (node *Node) beginElection() {
	node.state = CANDIDATE
	node.term += 1

	currentTerm := node.term
	currentLastLogIndex := node.lastLogIndex
	currentLastLogTerm := node.lastLogTerm

	result, newTerm, _, _ := node.client.requestVotes(currentTerm, node.nodeId, currentLastLogIndex, currentLastLogTerm)
	node.mu.Lock()
	// Another node was elected leader
	if node.state != CANDIDATE {
		return
	}
	node.mu.Unlock()

	switch result {
	case RV_TERM_OUT_OF_DATE:
		node.becomeFollower(newTerm)
		return
	case MAJORITY:
		node.becomeLeader()
		return
	case SPLIT:
	case LOST:
	}
	return
}

func (node *Node) becomeFollower(newTerm int32) {
	node.mu.Lock()
	node.state = FOLLOWER
	node.term = newTerm
	node.mu.Unlock()

	go node.runElectionTimer()
}

func (node *Node) becomeLeader() {
	node.mu.Lock()
	nodeId := node.nodeId
	node.state = LEADER
	savedCurrentTerm := node.term
	node.mu.Unlock()

	go func() {
		timer := time.NewTimer(HEARTBEAT_INTERVAL * time.Millisecond)
		defer timer.Stop()
		for {
			<- timer.C
			result, term := node.client.appendEntries(savedCurrentTerm, nodeId, 0,0,0, make(map[int32]string))
			switch result {
			case AE_TERM_OUT_OF_DATE:
				node.becomeFollower(term)
				return
			case SUCCESS:
			case FAILIURE:
			}
		}
	}()
}

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
