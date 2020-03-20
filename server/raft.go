package node

import (
	"math/rand"
	"sync"
	"time"
)

type Node struct {
	// Identifier of this Server
	serverId int32
	// Identifiers all other Servers in the cluster
	peerIds int32
	// Node state
	state STATE
	// Current Term
	term int32

	//
	timerEvent chan TIMEREVENT
	// Lock
	mu sync.Mutex

	// RPC Reciever
	// RPC Client
}

type RaftNode interface {
	runElectionTimer()
	beginElection()
	generateElectionTimeout()
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
