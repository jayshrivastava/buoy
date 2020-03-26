package node

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

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
	// Nex† Index (for each server, the index of the next entry to send to that server)
	nextIndex map[int32]int32
	// Match Index (for each server, the index of hightest entry known to be replicated on that server)
	matchIndex map[int32]int32
	// Key value store
	kv map[int32]string
	// Log
	log []logEntry
	// Event to reset the timer or indivate timer expiry
	te TIMEREVENT
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

func RunRaftNode(id int32, port string, iport string, iports map[int32]string, wg *sync.WaitGroup, state STATE) {
	defer wg.Done()

	externalNodeIds := []int32{}
	for externalNodeId, _ := range iports {
		externalNodeIds = append(externalNodeIds, externalNodeId)
	}
	node := raftNode{
		id:              id,
		externalNodeIds: externalNodeIds,
		state:           state,
		term:            0,
		votedFor:        -1,
		commitIndex:     0,
		lastApplied:     0,
		nextIndex:       map[int32]int32{},
		matchIndex:      map[int32]int32{},
		kv:              map[int32]string{},
		log:             []logEntry{logEntry{key: 0, value: "", term: 0}},
		te:              NONE,
		mu:              sync.Mutex{},
		// Sender not implemented until receivers are running
		l: CreateLogger(id),
	}
	for _, nodeId := range node.externalNodeIds {
		node.nextIndex[nodeId] = 0
		node.matchIndex[nodeId] = 0
	}

	go RunApiServer(port, &node)
	go RunRaftReceiver(iport, &node)

	node.sender = CreateRaftSender(iports, &node)

	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	go node.Run()

	node.l.Log("Launched")
	node.becomeFollower(0)
	wg2.Wait()
}

func (node *raftNode) Run() {
	for {
		time.Sleep(500 * time.Millisecond)

		node.l.Log(node.getState().String())
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
	node.l.Log("starting election timer")
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
			node.l.Log("Timer Expired")
			node.beginElection()
			return
		}

		node.l.Log("Timer Reset")
	}
}

// Lock must be aquired before calling this
func (node *raftNode) beginElection() {
	node.l.Log("Starting election")
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
	node.l.Log(fmt.Sprintf("Got vote result %s with term %d", result, node.term))

	// If we become a follower or leader during this time
	if node.getState() != CANDIDATE {
		node.l.Log("Not candidate anymore")
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
	node.l.Log("Became follower")
	node.mu.Lock()

	node.state = FOLLOWER
	node.term = newTerm
	node.votedFor = -1

	node.mu.Unlock()

	go node.runElectionTimer()
}

// LOCK MUST BE ACQUIRED
func (node *raftNode) becomeLeader() {
	node.l.Log("Became LEADER")
	node.mu.Lock()
	node.state = LEADER
	node.votedFor = -1
	term := node.term
	leaderId := node.id
	node.mu.Unlock()

	go func() {
		for {
			if node.getState() != LEADER {
				return
			}

			node.l.Log("Sending append entries")
			result, term := node.sender.appendEntries(term, leaderId, 0, 0, 0, make(map[int32]string))

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
)

func (t STATE) String() string {
	return [...]string{"LEADER", "FOLLOWER", "CANDIDATE"}[t]
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
