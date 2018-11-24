package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import "sync"
import "NJU-DisSys-2017/src/labrpc"

import "bytes"
import "encoding/gob"
import "time"
import "math/rand"


//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make().
//
type ApplyMsg struct {
	Index       int
	Command     interface{}
	UseSnapshot bool   // ignore for lab2; only used in lab3
	Snapshot    []byte // ignore for lab2; only used in lab3
}

type LogEntry struct {
	Index int
	Term int
	Command interface {}
}

const(
	Leader = iota
	Follower
	Candidate
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex
	peers     []*labrpc.ClientEnd
	persister *Persister
	me        int // index into peers[]

	// Your data here.
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	
	role int
	currentTerm int
	votedFor int
	log[] LogEntry

	commitIndex int
	lastApplied int

	nextIndex int
	matchIndex int

	heartbeat bool
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here.
	term = rf.currentTerm
	isleader = rf.role == Leader
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here.
	w := new(bytes.Buffer)
	e := gob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	// Your code here.
	r := bytes.NewBuffer(data)
	d := gob.NewDecoder(r)
	d.Decode(&rf.currentTerm)
	d.Decode(&rf.votedFor)
	d.Decode(&rf.log)
}




//
// example RequestVote RPC arguments structure.
//
type RequestVoteArgs struct {
	// Your data here.
	Term int
	CandidateId int
	LastLogIndex int
	LastLogTerm int
}

//
// example RequestVote RPC reply structure.
//
type RequestVoteReply struct {
	// Your data here.
	Term int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here.
	if args.Term < rf.currentTerm {
		rf.mu.Lock()
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		rf.mu.Unlock()
		return
	}
	if args.Term > rf.currentTerm {
		rf.mu.Lock()
		rf.currentTerm = args.Term
		reply.Term = args.Term
		reply.VoteGranted = true
		rf.votedFor = args.CandidateId
		rf.mu.Unlock()
		go rf.follower()
	}
	if (rf.votedFor != -1 || rf.votedFor == args.CandidateId) {
		rf.mu.Lock()
		reply.VoteGranted = true
		rf.votedFor = args.CandidateId
		rf.mu.Unlock()
	}
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// returns true if labrpc says the RPC was delivered.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) bool {
	channel := make(chan bool)
	
	go func() {
		channel <- rf.peers[server].Call("Raft.RequestVote", args, reply)
	}()
	timer := time.NewTimer(time.Duration(5) * time.Millisecond)
	select{
	case <- timer.C:
		return false
	case response := <- channel:
		return response
	}
}

func (rf *Raft) handleSendRequestVote(server int, channel chan *RequestVoteReply) {
	args := RequestVoteArgs{}
	args.Term = rf.currentTerm
	args.CandidateId = rf.me
	reply := &RequestVoteReply{}
	if rf.sendRequestVote(server, args, reply) {
		channel <- reply
	} else {
		channel <- nil
	}
}


//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := rf.lastApplied +1
	term, isLeader := rf.GetState()
	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here.
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]LogEntry, 1, 100)

	rf.commitIndex = 0
	rf.lastApplied = 0

	rf.nextIndex = 0
	rf.matchIndex = 0

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// servers are initialized as followers
	go rf.follower()

	return rf
}

func (rf *Raft) randomNumberGenerator() int {
	randomNumber := rand.Intn(150) + 150
	return randomNumber
}

func (rf *Raft) follower() {
	rf.role = Follower
	rf.heartbeat = true
	for {
		timer := time.NewTimer(time.Duration(rf.randomNumberGenerator()) * time.Millisecond)
		<-timer.C
		// Respond to RPC from candidates and leaders
		if rf.role != Follower {
			return
		}

		// Become candidate if heartbeat not received
		if !rf.heartbeat {
			go rf.candidate()
		}
		rf.heartbeat = false
	}
}

func (rf *Raft) candidate() {
	rf.role = Candidate
	
	// Increment current term
	rf.currentTerm += 1
	// Vote for self
	rf.votedFor = rf.me
	votes := 1
	channel := make(chan *RequestVoteReply)
	timer := time.NewTimer(time.Duration(rf.randomNumberGenerator()) * time.Millisecond)
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			// Send Request Vote to all other servers
			go rf.handleSendRequestVote(i, channel)
		}
	}
	for i := 0; i < len(rf.peers); i++ {
		reply := <-channel
		if reply != nil {
			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				go rf.follower()
				return
			} else if reply.VoteGranted {
				votes += 1
			}
		}
	}
	if votes >= len(rf.peers)-1/2 {
		go rf.leader()
		return
	}
	<-timer.C
	if rf.role != Candidate {
		return
	}
}

func (rf *Raft) leader() {
	rf.role = Leader
	for {
		channel := make(chan *AppendEntriesReply)
		for i := 0; i < len(rf.peers); i++ {
			if i != rf.me {
				reply := <- channel
				if reply != nil {
					if reply.Term > rf.currentTerm {
						rf.currentTerm = reply.Term
						go rf.follower()
						return
					}
				}
			}
		}
	}
}

type AppendEntriesArgs struct {
	Term int
	LeaderId int
	PrevLogIndex int
	PrevLogTerm int
	entries[] LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term int
	Success bool
}

func (rf *Raft) AppendEntries (args AppendEntriesArgs, reply *AppendEntriesReply) {
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	rf.heartbeat = true
	if args.Term > rf.currentTerm {
		rf.votedFor = -1
		go rf.follower()
	}
	reply.Term = args.Term
	reply.Success = true
}

func (rf *Raft) sendAppendEntries(server int, args AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) handleAppendEntries(reply AppendEntriesReply) {
	if reply.Success {
		return
	}
	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.votedFor = -1
		go rf.follower()
		return
	}
}