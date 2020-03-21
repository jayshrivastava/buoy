package node

type receiver struct{}

type raftReciever interface {
	RequestVoteHandler()
	AppendEntriesHandler()
}

func (receiver *receiver) RequestVoteHandler()   {}
func (receiver *receiver) AppendEntriesHandler() {}
