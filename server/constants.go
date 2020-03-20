package node

// Raft Node State
type STATE int

const (
	LEADER STATE = iota
	FOLLOWER
	CANDIDATE
)

// Time
const MIN_ELECTION_TIMEOUT = 150
const MAX_ELECTION_TIMEOUT = 300

type TIMEREVENT int

const (
	RESET TIMEREVENT = iota
	EXPIRED
)
