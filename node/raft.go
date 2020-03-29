package node

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type NodeConfig struct {
	Id               int32
	InternalOutbound string
	ExternalOutbound string
	PeerNodeHosts    map[int32]string
}

type NodeLogger interface {
	Log(int32, string)
}

type logEntry struct {
	key   int32
	value string
	term  int32
}

type raftNode struct {
	// Identifier of this Server
	id int32
	// Identifier of all other servers
	externalNodeIds []int32
	// Node state
	state STATE
	// Current Term
	term int32
	// Id of the server voted for during the current term (above) -1 to uninitialize
	votedFor int32
	// Index of highest log entry known to be committed. Monotonic increments
	commitIndex int32
	// Index of highest log entry known to be applied to the key value store
	lastApplied int32
	// Match Index (for each server, the index of hightest entry known to be replicated on that server)
	matchIndex map[int32]int32
	// Key value store
	kv map[int32]string
	// Log
	log []*logEntry
	// Event to reset the timer or indivate timer expiry
	te TIMEREVENT
	// Lock - applies everything other than locks below this one
	mu sync.Mutex
	// Lock - protects commitIndex, lastApplied, log, and kv, and matchIndex
	dataMu sync.RWMutex
	// RPC Client
	sender RaftSender
	// logger
	l NodeLogger
}

func RunRaftNode(cfg NodeConfig, l NodeLogger) {

	externalNodeIds := []int32{}
	for externalNodeId, _ := range cfg.PeerNodeHosts {
		externalNodeIds = append(externalNodeIds, externalNodeId)
	}
	node := raftNode{
		id:              cfg.Id,
		externalNodeIds: externalNodeIds,
		state:           FOLLOWER,
		term:            0,
		votedFor:        -1,
		commitIndex:     0,
		lastApplied:     0,
		matchIndex:      map[int32]int32{},
		kv:              map[int32]string{},
		log:             []*logEntry{&logEntry{key: 0, value: "", term: 0}},
		te:              NONE,
		mu:              sync.Mutex{},
		dataMu:          sync.RWMutex{},
		// Sender not implemented until receivers are running
		l: l,
	}
	for _, nodeId := range node.externalNodeIds {
		node.matchIndex[nodeId] = 0
	}

	go RunApiServer(cfg.ExternalOutbound, &node)
	go RunRaftReceiver(cfg.InternalOutbound, &node)

	node.sender = CreateRaftSender(cfg.PeerNodeHosts, &node)

	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	go node.Run()

	node.l.Log(node.id, "Launched Node")
	node.becomeFollower(0)
	wg2.Wait()
}

// lock must be acquired here
func (node *raftNode) AddLogEntry(key int32, value string, term int32) {
	node.l.Log(node.id, fmt.Sprintf("Appending log entry k:%d v:%s term:%d", key, value, term))	
	node.log = append(node.log, &logEntry{key: key, value: value, term: term})
}

// lock must be acquired here
func (node *raftNode) CatchupCommits() {

	for i := node.lastApplied + 1; i <= node.commitIndex; i++ {
		node.kv[node.log[i].key] = node.log[i].value
		node.l.Log(node.id, fmt.Sprintf("Committing Log Entry k:%d v:%s", node.log[i].key, node.log[i].value))
	}
	node.lastApplied = node.commitIndex
}

func (node *raftNode) Run() {
	for {
		time.Sleep(500 * time.Millisecond)

		node.l.Log(node.id, node.getState().String())
	}
}

func (node *raftNode) generateElectionTimeout() time.Duration {
	return time.Duration(MIN_ELECTION_TIMEOUT+rand.Intn(MAX_ELECTION_TIMEOUT-MIN_ELECTION_TIMEOUT)) * time.Millisecond
}

func (node *raftNode) resetTimerEvent(term int32) {
	node.mu.Lock()
	defer node.mu.Unlock()
	node.te = RESET
	node.term = term
}

// blocking
func (node *raftNode) getAndUnsetTimerEvent() TIMEREVENT {
	node.mu.Lock()
	defer node.mu.Unlock()
	te := node.te
	node.te = NONE
	return te
}

func (node *raftNode) getState() STATE {
	node.mu.Lock()
	defer node.mu.Unlock()
	return node.state
}

func (node *raftNode) runElectionTimer() {
	node.l.Log(node.id, "Starting election timer")
	timeout := node.generateElectionTimeout()

	for {
		timer := time.NewTimer(timeout)
		<-timer.C

		timerEvent := node.getAndUnsetTimerEvent()
		state := node.getState()

		if state != CANDIDATE && state != FOLLOWER {
			return
		}
		// If the timer expired, trigger an election
		if timerEvent == NONE {
			node.l.Log(node.id, "Timer Expired")
			node.beginElection()
			return
		}

		node.l.Log(node.id, "Timer Reset")
	}
}

// Lock must be aquired before calling this
func (node *raftNode) beginElection() {
	node.l.Log(node.id, "Starting election")
	node.mu.Lock()

	node.state = CANDIDATE
	node.term += 1
	node.votedFor = node.id

	term := node.term
	raftNodeId := node.id
	lastLogIndex := int32(len(node.log) - 1)
	lastLogTerm := node.log[int32(len(node.log)-1)].term

	node.mu.Unlock()

	result, newTerm := node.sender.requestVotes(term, raftNodeId, lastLogIndex, lastLogTerm)
	node.l.Log(node.id, fmt.Sprintf("requestVotes %s. Term = %d", result, node.term))

	// If we become a follower or leader during this time
	if node.getState() != CANDIDATE {
		node.l.Log(node.id, "Not candidate anymore")
		return
	}
	switch result {
	case MAJORITY:
		node.becomeLeader()
	default:
		node.becomeFollower(newTerm)
	}
}

func (node *raftNode) becomeFollower(newTerm int32) {
	node.l.Log(node.id, "Became follower")
	node.mu.Lock()

	node.state = FOLLOWER
	node.term = newTerm
	node.votedFor = -1

	node.mu.Unlock()

	go node.runElectionTimer()
}

// LOCK MUST BE ACQUIRED
func (node *raftNode) becomeLeader() {
	node.mu.Lock()
	node.l.Log(node.id, fmt.Sprintf("Became LEADER with term %d", node.term))
	node.state = LEADER
	node.votedFor = -1
	term := node.term
	leaderId := node.id
	node.dataMu.RLock()
	for _, nodeId := range node.externalNodeIds {
		node.matchIndex[nodeId] = 0
	}
	node.dataMu.RUnlock()
	node.mu.Unlock()

	go func() {
		for {
			if node.getState() != LEADER {
				return
			}

			node.l.Log(node.id, "Sending heartbeat")
			result, term := node.sender.heartbeat(term, leaderId)

			switch result {
			case AE_TERM_OUT_OF_DATE:
				node.becomeFollower(term)
				return
			case SUCCESS:
			case FAILIURE:
			}
			timer := time.NewTimer(HEARTBEAT_INTERVAL * time.Millisecond)
			defer timer.Stop()
			<-timer.C
			// Stop if we are not the leader anymore
		}
	}()
}

func (node *raftNode) appendToLog() {}

/* CONSTANTS */

// Raft Node State
type STATE int

const (
	LEADER STATE = iota
	FOLLOWER
	CANDIDATE
	DEAD
)

func (t STATE) String() string {
	return [...]string{"LEADER", "FOLLOWER", "CANDIDATE", "DEAD"}[t]
}

// Timers
const MIN_ELECTION_TIMEOUT = 150
const MAX_ELECTION_TIMEOUT = 300
const HEARTBEAT_INTERVAL = 50

type TIMEREVENT int

const (
	RESET TIMEREVENT = iota
	NONE
)

func (t TIMEREVENT) String() string {
	return [...]string{"RESET", "NONE"}[t]
}
