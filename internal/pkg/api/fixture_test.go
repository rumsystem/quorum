package api

import (
	"context"
	"log"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/rumsystem/quorum/testnode"
)

var (
	pidlist                                   []int
	bootstrapapi, peerapi, peerapi2           string
	peerapilist, groupIds                     []string
	timerange, nodes, groups, posts, synctime int
)

func TestMain(m *testing.M) {
	nodes = 2
	pidch := make(chan int)

	go func() {
		for {
			select {
			case pid := <-pidch:
				log.Println("receive pid", pid)
				pidlist = append(pidlist, pid)
				if len(pidlist) == 3 {
					return
				}
			}
		}
	}()

	var tempdatadir string
	ctx := context.Background()
	cliargs := testnode.Nodecliargs{Rextest: false}
	bootstrapapi, peerapilist, tempdatadir, _ = testnode.RunNodesWithBootstrap(ctx, cliargs, pidch, nodes)
	log.Println("peers: ", peerapilist)
	peerapi = peerapilist[0]
	peerapi2 = peerapilist[1]

	exitVal := m.Run()
	log.Println("after tests clean:", tempdatadir)
	testnode.Cleanup(tempdatadir, peerapilist)
	os.Exit(exitVal)
}

func GetMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func StringSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	sort.Strings(a)
	sort.Strings(b)

	b = b[:len(a)]
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func StringSetIn(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func RandString(n int) string {
	rand.Seed(time.Now().UnixNano())

	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
