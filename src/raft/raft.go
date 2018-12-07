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
import "math"
import "math/rand"
import "fmt"

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
	
	state ServerState
	currentTerm int
	votedFor int
	log[] LogEntry

	commitIndex int
	lastApplied int

	nextIndex[] int
	matchIndex[] int

	heartbeat bool
	applyCh chan ApplyMsg
	votes int
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
	From int
}

func (rf *Raft) isLogMatch(args RequestVoteArgs) bool {
	if rf.log[len(rf.log)-1].Term != args.LastLogTerm {
		return rf.log[len(rf.log)-1].Term < args.LastLogTerm
	} else {
		return len(rf.log) -1 <= args.LastLogIndex
	}
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Lock()
	defer rf.persist()
	reply.VoteGranted = false
	if rf.state == "Follower" {
		if args.Term > rf.currentTerm {
			rf.currentTerm = args.Term
			rf.votedFor = -1
			if rf.isLogMatch(args) {
				rf.votedFor = args.CandidateId
				rf.heartbeat = true
				reply.VoteGranted = true
			}
		} else if args.Term == rf.currentTerm {
			if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && rf.isLogMatch(args) {
				rf.votedFor = args.CandidateId
				rf.heartbeat = true
				reply.VoteGranted = true
			}
		}
	} else if rf.state == "Candidate" {
		if args.Term > rf.currentTerm {
			// If this peer has a lower election term than that of the candidate, update peer's term and grant vote
			rf.currentTerm = args.Term
			rf.votedFor = -1
			rf.RunServerLoopAsFollower()
			if rf.isLogMatch(args) {
				rf.votedFor = args.CandidateId
				reply.VoteGranted = true
			}
		}
	} else if rf.state == "Leader" {
		if args.Term > rf.currentTerm {
			rf.currentTerm = args.Term
			rf.votedFor = -1
			rf.RunServerLoopAsFollower()
			if rf.isLogMatch(args) {
				rf.votedFor = args.CandidateId
				reply.VoteGranted = true
			}
		}
	}
	reply.From = rf.me
	reply.Term = rf.currentTerm
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
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) handleSendRequestVote(server int) {
	args := RequestVoteArgs{}
	args.Term = rf.currentTerm
	args.LastLogIndex = len(rf.log) - 1
	args.LastLogTerm = rf.log[len(rf.log)-1].Term
	args.CandidateId = rf.me
	reply := &RequestVoteReply{}
	if rf.sendRequestVote(server, args, reply) {
		fmt.Println(reply)
		rf.mu.Lock()
		defer rf.mu.Lock()
		if rf.state == "Candidate" {
			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				rf.persist()
				rf.RunServerLoopAsFollower()
				return
			} else if reply.VoteGranted {
				rf.votes ++
			}
			if rf.votes >= len(rf.peers)/2+1 {
				rf.RunServerLoopAsLeader()
				return
			}
		}
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
	index := -1
	term, isLeader := rf.GetState()
	if isLeader {
		rf.mu.Lock()
		entry := LogEntry{Term: term, Command: command}
		rf.log = append(rf.log, entry)
		rf.persist()
		index = len(rf.log) - 1
		rf.mu.Unlock()
	}
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
	rf.state = "Follower"
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]LogEntry, 1)

	rf.commitIndex = 0
	rf.lastApplied = 0

	rf.nextIndex = nil
	rf.matchIndex = nil
	rf.heartbeat = false
	rf.applyCh = applyCh
	rf.votes = 0

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	rf.ServerLoop()

	return rf
}

func (rf *Raft) RunServerLoopAsFollower() {
	// servers are initialized as followers
	rf.state = "Follower"
	go rf.ServerLoop()
}

func (rf *Raft) RunServerLoopAsLeader() {
	rf.state = "Leader"
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	for i:= 0; i<len(rf.peers); i++ {
		if i != rf.me {
			rf.nextIndex[i] = len(rf.log)
			rf.matchIndex[i] = 0
		}
	}
	go rf.ServerLoop()
}

func (rf *Raft) randomNumberGenerator() int {
	// generate random number for timeout feature
	randomNumber := rand.Intn(150) + 150
	return randomNumber
}

func (rf *Raft) ServerLoop() {
		BreakLocation:
		// Server loop contains switch statement, checking whether server state is 'follower', 'candidate' or 'leader'
			switch rf.state {
					case "Follower":
						rf.heartbeat = false
						for {
							// Follower sets a timer. If it hasn't received a heartbeat from the leader before the timeour, it will
							// become a candidate. Timeout is set randomly between 150 and 300 ms by the randomNumberGenerator() function.
							timer := time.NewTimer(time.Duration(rf.randomNumberGenerator()) * time.Millisecond)
							// wait for timer to timeout
							<-timer.C
							// if no heartbeat received from leader within timeout period, server switches state to candidate.
							// it will then exit the switch statement and will re-enter with the state of 'Candidate'
							if rf.state == "Follower" && rf.heartbeat == false {
								rf.state = "Candidate"
								break BreakLocation
							}
							rf.heartbeat = false
						}
					case "Candidate":
						fmt.Println(rf.state)
						fmt.Println(rf.heartbeat)
						for {
							// Candidate starts new election term and votes for itself
							rf.mu.Lock()
							rf.currentTerm++
							rf.votedFor = rf.me
							rf.persist()
							rf.votes = 1
							// Candidate sends vote requests to other peers
							for i := 0; i < len(rf.peers); i++ {
								if i != rf.me {
									go rf.handleSendRequestVote(i)
								}
							}
							rf.mu.Unlock()
							// Election timer is set
							timer := time.NewTimer(time.Duration(rf.randomNumberGenerator()) * time.Millisecond)
							// Wait for timer to timeout before running new election
							<-timer.C
						}
					case "Leader":
						for {
							// Leader will send regular heartbeats to follower peers.
							for i := 0; i < len(rf.peers); i++ {
									go rf.SendHeartBeat(i)
								}
							for i := len(rf.log) - 1; i > rf.commitIndex; i-- {
								if rf.log[i].Term == rf.currentTerm {
									count := 0
									for j := 0; j < len(rf.peers); j++ {
										if j != rf.me && rf.matchIndex[j] >= i {
											count ++
										}
									}
									if count >= len(rf.peers)/2 {
										rf.commitIndex = i
										break
										}
									}
								}
								timer := time.NewTimer(time.Duration(200) * time.Millisecond)
								go func() {
									for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
										rf.applyCh <- ApplyMsg{Index: i, Command: rf.log[i].Command}
										rf.lastApplied = i
									}
								}()
								<-timer.C
							}
						}
		}

type AppendEntriesArgs struct {
	Term int
	LeaderId int
	PrevLogIndex int
	PrevLogTerm int
	Entries[] LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term int
	Success bool
	NextIndex int
	From int
}

// Reply false if term < currentTerm
func (rf *Raft) AppendEntries (args AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	if rf.state == "Follower" {
		rf.heartbeat = true
		if args.Term > rf.currentTerm {
			rf.currentTerm = args.Term
		}
	} else if rf.state == Candidate {
		if args.Term >= rf.currentTerm {
			rf.currentTerm = args.Term
			rf.RunServerLoopAsFollower()
		}
	} else if rf.state == Leader {
		if args.Term > rf.currentTerm {
			rf.currentTerm = args.Term
			rf.RunServerLoopAsFollower()
		}
	}
	reply.From = rf.me
	reply.Term = rf.currentTerm
	// deal log replication
	if len(args.Entries)!=0 {}
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.NextIndex = 0//useless
	} else if args.PrevLogIndex >= len(rf.log) {
		reply.Success = false
		// optimization
		reply.NextIndex = len(rf.log)
	} else if rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Success = false
		// optimization
		for i:=args.PrevLogIndex-1;i>=0;i--{
			if rf.log[i].Term != rf.log[args.PrevLogIndex].Term {
				reply.NextIndex = i + 1
				break
			}
		}
	} else {
		if len(rf.log) > args.PrevLogIndex+1 {
			rf.log = rf.log[:args.PrevLogIndex+1]
		}
		rf.log = append(rf.log, args.Entries...)
		if len(args.Entries) != 0 {}
		reply.Success = true
		reply.NextIndex = len(rf.log)

		if args.LeaderCommit > rf.commitIndex {
			N := int(math.Min(float64(args.LeaderCommit), float64(len(rf.log)-1)))
			rf.commitIndex = N
		}
		go func() {
			for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
				rf.applyCh <- ApplyMsg{Index: i, Command: rf.log[i].Command}
				rf.lastApplied = i
			}
		}()
	}
}

// This function includes the RPC for sending the heartbeat along with the payload to the peer servers
func (rf *Raft) sendAppendEntries(server int, args AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// In this function, heartbeats are sent out to peers. The AppendEntriesArgs are 
// conveniently sent with the hearbeat as a payload.
func (rf *Raft) SendHeartBeat(i int) {
	payload := AppendEntriesArgs{}
	payload.Term = rf.currentTerm
	payload.LeaderId = rf.me
	payload.PrevLogIndex = rf.nextIndex[i] - 1
	payload.PrevLogTerm = rf.log[payload.PrevLogIndex].Term
	payload.Entries = rf.log[rf.nextIndex[i]:]
	payload.LeaderCommit = rf.commitIndex
	reply := &AppendEntriesReply{}
	if rf.sendAppendEntries(i, payload, reply) {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if rf.state == "Leader" {
			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				rf.persist()
				rf.RunServerLoopAsFollower()
				return
			} else if reply.Success {
				rf.matchIndex[i] = reply.NextIndex - 1
				if rf.nextIndex[i] < reply.NextIndex {
					rf.nextIndex[i] = reply.NextIndex
				}
				rf.nextIndex[i] = reply.NextIndex
			} else {
				rf.nextIndex[i] = reply.NextIndex
			}
		}
	}
}