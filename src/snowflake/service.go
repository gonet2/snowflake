package main

import (
	"errors"
	"fmt"
	log "github.com/GameGophers/nsq-logger"
	"github.com/coreos/go-etcd/etcd"
	"golang.org/x/net/context"
	"os"
	pb "snowflake/proto"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SERVICE      = "[SNOWFLAKE]"
	DEFAULT_ETCD = "127.0.0.1:2379"
	PATH         = "/seqs/"
	UUID_KEY     = "/seqs/snowflake-uuid"
	RETRY_MAX    = 10
	RETRY_DELAY  = 10 * time.Millisecond
)

const (
	TS_MASK         = 0x1FFFFFFFFFF // 41bit
	SN_MASK         = 0xFFF         // 12bit
	MACHINE_ID_MASK = 0x3FF         // 10bit
)

type server struct {
	machines    []string
	sn          uint64 // 12-bit serial no
	machine_id  uint64 // 10-bit machine id
	last_ts     int64  // last timestamp
	client_pool sync.Pool
	sync.Mutex
}

func (s *server) init() {
	// get an unique value for consumer channel of nsq
	s.machines = []string{DEFAULT_ETCD}
	if env := os.Getenv("ETCD_HOST"); env != "" {
		s.machines = strings.Split(env, ";")
	}

	s.client_pool.New = func() interface{} {
		return etcd.NewClient(s.machines)
	}
	s.init_machine_id()
}

func (s *server) Next(ctx context.Context, in *pb.Snowflake_Key) (*pb.Snowflake_Value, error) {
	client := s.client_pool.Get().(*etcd.Client)
	defer func() {
		s.client_pool.Put(client)
	}()
	key := PATH + in.Name

	for i := 0; i < RETRY_MAX; i++ {
		// get the key
		resp, err := client.Get(key, false, false)
		if err != nil {
			log.Critical(SERVICE, err)
			return nil, errors.New("Key not exists, need to create first")
		}

		// get prevValue & prevIndex
		prevValue, err := strconv.Atoi(resp.Node.Value)
		if err != nil {
			log.Critical(SERVICE, err)
			return nil, errors.New("marlformed value")
		}
		prevIndex := resp.Node.ModifiedIndex

		// CAS
		resp, err = client.CompareAndSwap(key, fmt.Sprint(prevValue+1), 0, resp.Node.Value, prevIndex)
		if err != nil {
			log.Error(SERVICE, err)
			<-time.After(RETRY_DELAY)
			continue
		}
		return &pb.Snowflake_Value{int64(prevValue + 1)}, nil
	}
	return nil, errors.New("etcd server busy")
}

func (s *server) init_machine_id() {
	client := s.client_pool.Get().(*etcd.Client)
	defer func() {
		s.client_pool.Put(client)
	}()

	for i := 0; i < RETRY_MAX; i++ {
		// get the key
		resp, err := client.Get(UUID_KEY, false, false)
		if err != nil {
			log.Critical(SERVICE, err)
			os.Exit(-1)
		}

		// get prevValue & prevIndex
		prevValue, err := strconv.Atoi(resp.Node.Value)
		if err != nil {
			log.Critical(SERVICE, err)
			os.Exit(-1)
		}
		prevIndex := resp.Node.ModifiedIndex

		// CAS
		resp, err = client.CompareAndSwap(UUID_KEY, fmt.Sprint(prevValue+1), 0, resp.Node.Value, prevIndex)
		if err != nil {
			log.Error(SERVICE, err)
			<-time.After(RETRY_DELAY)
			continue
		}

		// record serial number of this service, already shifted
		s.machine_id = (uint64(prevValue+1) & MACHINE_ID_MASK) << 12
		return
	}
}

func (s *server) GetUUID(context.Context, *pb.Snowflake_NullRequest) (*pb.Snowflake_UUID, error) {
	s.Lock()
	defer s.Unlock()

	// get a correct serial number
	t := s.ts()
	if t < s.last_ts { // clock shift backward
		log.Error(SERVICE, "clock shift happened, waiting until the clock moving to the next millisecond.")
		t = s.wait_ms(s.last_ts)
	}

	if s.last_ts == t { // same millisecond
		s.sn = (s.sn + 1) & SN_MASK
		if s.sn == 0 { // serial number overflows, wait until next ms
			t = s.wait_ms(s.last_ts)
		}
	} else { // new millsecond, reset serial number to 0
		s.sn = 0
	}
	// remember last timestamp
	s.last_ts = t

	// generate uuid, format:
	//
	// 0		0.................0		0..............0	0........0
	// 1-bit	41bit timestamp			10bit machine-id	12bit sn
	var uuid uint64
	uuid |= (uint64(t) & TS_MASK) << 22
	uuid |= s.machine_id
	uuid |= s.sn

	return &pb.Snowflake_UUID{uuid}, nil
}

// wait_ms will spin wait till next millisecond.
func (s *server) wait_ms(last_ts int64) int64 {
	t := s.ts()
	for t <= s.last_ts {
		t = s.ts()
	}
	return t
}

// get timestamp
func (s *server) ts() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
