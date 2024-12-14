package topic

import (
	"log"
	"strings"
	"testing"
)

func Test_TrieNode(t *testing.T) {

	topic := NewMemoryTrie()
	topic.Subscribe("1/2/3")
	topic.Subscribe("2/4")
	//topic.Subscribe("2/+/+")
	topic.Subscribe("2/+/#")
	topic.Subscribe("#")

	//topic.Subscribe("/2/3/4")
	topic.Print()

	for _, path := range []string{
		"1/2/3",
		"1/2/3/4",
		"2/3/4",
		"2/3/4/5",
	} {
		subs, ok := topic.Find(path)
		log.Printf("path=%s, match=%v, subs=%v", path, ok, strings.Join(subs, "/"))
	}

	topic.Unsubscribe("#")
	topic.Print()

	topic.Unsubscribe("2/4")
	topic.Print()

	topic.Unsubscribe("2")
	topic.Print()

}
