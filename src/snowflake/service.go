package main

import (
	"errors"
	"etcdclient"
	"fmt"
	etcd "github.com/coreos/etcd/client"
	log "github.com/MISingularity/deepshare2/pkg/log"
	"golang.org/x/net/context"
	"math/rand"
	"os"
	pb "proto"
	"strconv"
	"time"
)

const (
	SERVICE        = "[SNOWFLAKE]"
	ENV_MACHINE_ID = "MACHINE_ID" // specific machine id
	PATH           = "/seqs/"
	UUID_KEY       = "/seqs/snowflake-uuid"
	BACKOFF        = 100  // max backoff delay millisecond
	CONCURRENT     = 128  // max concurrent connections to etcd
	UUID_QUEUE     = 1024 // uuid process queue
)

const (
	TS_MASK         = 0x1FFFFFFFFFF // 41bit
	SN_MASK         = 0xFFF         // 12bit
	MACHINE_ID_MASK = 0x3FF         // 10bit
)

type server struct {
	machine_id  uint64 // 10-bit machine id
	client_pool chan etcd.KeysAPI
	ch_proc     chan chan uint64
}

func (s *server) init() {
	s.client_pool = make(chan etcd.KeysAPI, CONCURRENT)
	s.ch_proc = make(chan chan uint64, UUID_QUEUE)

	// init client pool
	for i := 0; i < CONCURRENT; i++ {
		s.client_pool <- etcdclient.KeysAPI()
	}

	// check if user specified machine id is set
	if env := os.Getenv(ENV_MACHINE_ID); env != "" {
		if id, err := strconv.Atoi(env); err == nil {
			s.machine_id = (uint64(id) & MACHINE_ID_MASK) << 12
			log.Info("machine id specified:", id)
		} else {
			log.Fatal("checked error: unable to find user specified machine id: ", err)
			os.Exit(-1)
		}
	} else {
		s.init_machine_id()
	}

	go s.uuid_task()
}

func (s *server) init_machine_id() {
	client := <-s.client_pool
	defer func() { s.client_pool <- client }()

	for {
		// get the key
		resp, err := client.Get(context.Background(), UUID_KEY, nil)
		if err != nil {
			log.Fatal("unable to get the key from etcd: ", err)
			os.Exit(-1)
		}

		// get prevValue & prevIndex
		prevValue, err := strconv.Atoi(resp.Node.Value)
		if err != nil {
			log.Fatal("unable to get prevValue & prevIndex: ", err)
			os.Exit(-1)
		}
		prevIndex := resp.Node.ModifiedIndex

		// CompareAndSwap
		resp, err = client.Set(context.Background(), UUID_KEY, fmt.Sprint(prevValue+1), &etcd.SetOptions{PrevIndex: prevIndex})
		if err != nil {
			cas_delay()
			continue
		}

		// record serial number of this service, already shifted
		s.machine_id = (uint64(prevValue+1) & MACHINE_ID_MASK) << 12
		return
	}
}

// get next value of a key, like auto-increment in mysql
func (s *server) Next(ctx context.Context, in *pb.Snowflake_Key) (*pb.Snowflake_Value, error) {
	client := <-s.client_pool
	defer func() { s.client_pool <- client }()
	key := PATH + in.Name
	for {
		// get the key
		resp, err := client.Get(context.Background(), key, nil)
		if err != nil {
			log.Fatal("unable to get next value of a key: ", err)
			return nil, errors.New("Key not exists, need to create first")
		}

		// get prevValue & prevIndex
		prevValue, err := strconv.Atoi(resp.Node.Value)
		if err != nil {
			log.Fatal("unable to get prevValue & prevIndex: ". err)
			return nil, errors.New("marlformed value")
		}
		prevIndex := resp.Node.ModifiedIndex

		// CompareAndSwap
		resp, err = client.Set(context.Background(), key, fmt.Sprint(prevValue+1), &etcd.SetOptions{PrevIndex: prevIndex})
		if err != nil {
			cas_delay()
			continue
		}
		return &pb.Snowflake_Value{int64(prevValue + 1)}, nil
	}
}

// generate an unique uuid
func (s *server) GetUUID(context.Context, *pb.Snowflake_NullRequest) (*pb.Snowflake_UUID, error) {
	req := make(chan uint64, 1)
	s.ch_proc <- req
	return &pb.Snowflake_UUID{<-req}, nil
}

// uuid generator
func (s *server) uuid_task() {
	var sn uint64     // 12-bit serial no
	var last_ts int64 // last timestamp
	for {
		ret := <-s.ch_proc
		// get a correct serial number
		t := ts()
		if t < last_ts { // clock shift backward
			log.Error("clock shift happened, waiting until the clock moving to the next millisecond.")
			t = s.wait_ms(last_ts)
		}

		if last_ts == t { // same millisecond
			sn = (sn + 1) & SN_MASK
			if sn == 0 { // serial number overflows, wait until next ms
				t = s.wait_ms(last_ts)
			}
		} else { // new millsecond, reset serial number to 0
			sn = 0
		}
		// remember last timestamp
		last_ts = t

		// generate uuid, format:
		//
		// 0		0.................0		0..............0	0........0
		// 1-bit	41bit timestamp			10bit machine-id	12bit sn
		var uuid uint64
		uuid |= (uint64(t) & TS_MASK) << 22
		uuid |= s.machine_id
		uuid |= sn
		ret <- uuid
	}
}

// wait_ms will spin wait till next millisecond.
func (s *server) wait_ms(last_ts int64) int64 {
	t := ts()
	for t <= last_ts {
		t = ts()
	}
	return t
}

////////////////////////////////////////////////////////////////////////////////
// random delay
func cas_delay() {
	<-time.After(time.Duration(rand.Int63n(BACKOFF)) * time.Millisecond)
}

// get timestamp
func ts() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
