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

// import "bytes"
// import "encoding/gob"
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

type ServerState string

const(
	Leader ServerState = "Leader"
	Follower ServerState = "Follower"
	Candidate ServerState = "Candidate"
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
	
	state string
	currentTerm int
	votedFor int
	log[] LogEntry

	commitIndex int
	lastApplied int

	nextIndex[] int
	matchIndex[] int

	heartbeat bool
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here.
	term = rf.currentTerm
	if rf.state == "Leader" {
		isleader = true
	} else {
		isleader = false
	}
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here.
	// w := new(bytes.Buffer)
	// e := gob.NewEncoder(w)
	// e.Encode(rf.currentTerm)
	// e.Encode(rf.votedFor)
	// e.Encode(rf.log)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	// Your code here.
	// r := bytes.NewBuffer(data)
	// d := gob.NewDecoder(r)
	// d.Decode(&rf.currentTerm)
	// d.Decode(&rf.votedFor)
	// d.Decode(&rf.log)
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
		rf.state = "Follower"
		rf.mu.Unlock()
		go rf.ServerLoop()
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
	ch := make(chan bool)
	go func(){
		ch <- rf.peers[server].Call("Raft.RequestVote", args, reply)
	}()
	timer := time.NewTimer(time.Duration(5) * time.Millisecond)
	return rf.TimerExpired(timer, ch)
}

func (rf *Raft) TimerExpired(timer *time.Timer, ch chan bool) bool {
	select{
	case <-timer.C:
		return false
	case r := <-ch:
		return r
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

	rf.nextIndex = nil
	rf.matchIndex = nil
	rf.heartbeat = false

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	rf.RunServerLoop()

	return rf
}

func (rf *Raft) RunServerLoop() {
	// servers are initialized as followers
	rf.state = "Follower"
	go rf.ServerLoop()
}

func (rf *Raft) randomNumberGenerator() int {
	// generate random number for timeout feature
	randomNumber := rand.Intn(150) + 150
	return randomNumber
}

func (rf *Raft) ServerLoop() {
	for {
		BreakLocation:
		// Server loop contains switch statement, checking whether server state is 'follower', 'candidate' or 'leader'
			switch rf.state {
					case "Follower":
						rf.heartbeat = true
						for {
							// Follower sets a timer. If it hasn't received a heartbeat from the leader before the timeour, it will
							// become a candidate. Timeout is set randomly between 150 and 300 ms by the randomNumberGenerator() function.
							timer := time.NewTimer(time.Duration(rf.randomNumberGenerator()) * time.Millisecond)
							// wait for timer to timeout
							<-timer.C
							// if no heartbeat received from leader within timeout period, server switches state to candidate.
							// it will then exit the switch statement and will re-enter with the state of 'Candidate'
							if !rf.heartbeat {
								rf.state = "Candidate"
								break BreakLocation
							}
							rf.heartbeat = false
						}
					case "Candidate":
						for {
							// Candidate starts new election term and votes for itself
							rf.currentTerm++
							rf.votedFor = rf.me
							votes := 1
							// Election timer is set
							timer := time.NewTimer(time.Duration(rf.randomNumberGenerator()) * time.Millisecond)
							ch := make(chan *RequestVoteReply)
							// Candidate sends vote requests to other peers
							for i := 0; i < len(rf.peers); i++ {
								if i != rf.me {
									go rf.handleSendRequestVote(i, ch)
								}
							}
							// Iterate through replies to vote requests from peers
							for i := 0; i < len(rf.peers); i++ {
								if i != rf.me {
									// retrieve the reply from the channel
									reply := <-ch
									if reply != nil {
										if reply.Term > rf.currentTerm {
											// If the election term of the candidate server is lower than that of a peer, the candidate will
											// update its term value and will also change its state to follower
											rf.currentTerm = reply.Term
											rf.state = "Follower"
											break BreakLocation
										} else if reply.VoteGranted {
											// At this point, we know the candidate's election term is at least as high as that of the 
											// responding peer. Therefore, this peer's vote will be counted.
											votes++
										}
									}
								}
							}
							// If candidate receives majority of votes from peers it can transfer state to 'Leader'
							if votes >= len(rf.peers)/2+1 {
								rf.state = "Leader"
								break BreakLocation
							}
							// Wait for timer to timeout before running new election
							<-timer.C
						}
					case "Leader":
						for {
							// Leader will send regular heartbeats to follower peers.
							ch := make(chan *AppendEntriesReply)
							for i := 0; i < len(rf.peers); i++ {
								if i != rf.me {
									go rf.SendHeartBeat(i, ch)
								}
							}
							// TODO: handle commands received from client and implement logic for logging

							// loop response to heartbeats from peers
							for i := 0; i < len(rf.peers); i++ {
								if i != rf.me {
									reply := <-ch
									if reply != nil {
										// If responding peer is at a later election term, the leader will 
										// update its term and will change state to follower
										if reply.Term > rf.currentTerm {
											rf.currentTerm = reply.Term
											rf.state = "Follower"
											break BreakLocation
										}
									}
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

// Reply false if term < currentTerm
func (rf *Raft) AppendEntries (args AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.heartbeat = true
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
	}
}

// This function includes the RPC for sending the heartbeat along with the payload to the peer servers
func (rf *Raft) sendAppendEntries(server int, args AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ch := make(chan bool)
	go func () {
		ch <- rf.peers[server].Call("Raft.AppendEntries", args, reply)
	}()
	timer := time.NewTimer(time.Duration(5) * time.Millisecond)
	return rf.TimerExpired(timer, ch)
}

// In this function, heartbeats are sent out to peers. The AppendEntriesArgs are 
// conveniently sent with the hearbeat as a payload.
func (rf *Raft) SendHeartBeat(i int,ch chan *AppendEntriesReply) {
	payload := AppendEntriesArgs{}
	payload.Term = rf.currentTerm
	payload.LeaderId = rf.me
	reply := &AppendEntriesReply{}
	if rf.sendAppendEntries(i, payload, reply) {
		ch <- reply
	} else {
		ch <- nil
	}
}