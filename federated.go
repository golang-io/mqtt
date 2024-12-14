package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-io/requests"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Endpoint 不提供事务能力，事务需要其他的上层协议来处理
type Endpoint struct {
	running bool
	id      string
	url     *url.URL
	active  map[string]string //id: url
	mu      *sync.RWMutex
	sess    *requests.Session
	content chan []byte
}

func (e *Endpoint) Send(content []byte) error {
	for id, endpoint := range e.List() {
		if id == "" || id == e.id {
			continue
		}
		resp, err := e.sess.DoRequest(context.Background(),
			requests.URL(endpoint),
			requests.Path("/send"),
			requests.Header("content-type", "application/json"),
			requests.Body(content),
			requests.Logf(func(ctx context.Context, stat *requests.Stat) {
				//log.Printf("%#v", stat)
			}),
		)
		if err != nil {
			log.Printf("%v", err)
			continue
		}
		if resp.StatusCode != 200 {
			log.Printf("status code=%v", resp.StatusCode)
			continue
		}
	}
	return nil
}

// Ping 这里有个问题，如果一个节点退出后，立即加入集群，会存在之前的节点没有清理的问题
// 这里的目前解决办法就是，节点sleep5秒之后再重新进入集群
func (e *Endpoint) Ping() {
	body := map[string]string{e.id: fmt.Sprintf("http://127.0.0.1:%s", e.url.Port())}
	for id, endpoint := range e.List() {
		resp, err := e.sess.DoRequest(context.Background(),
			requests.URL(endpoint),
			requests.Path("/ping"),
			requests.Header("content-type", "application/json"),
			requests.Body(body),
			requests.Logf(func(ctx context.Context, stat *requests.Stat) {
				//log.Printf("%s", stat)
				//log.Printf("[federated] %s => %v\n", stat.Request.URL, stat.Response.Body)
			}),
		)
		if err != nil {
			log.Printf("%v", err)
			e.mu.Lock()
			delete(e.active, id)
			e.mu.Unlock()
			continue
		}
		if resp.StatusCode != 200 {
			log.Printf("status code=%v", resp.StatusCode)
			e.mu.Lock()
			delete(e.active, id)
			e.mu.Unlock()
			continue
		}
		var remotes map[string]string
		if err := json.Unmarshal(resp.Content.Bytes(), &remotes); err != nil {
			log.Printf("%v", err)
			continue
		}
		e.mu.Lock()
		// 删除不存在的
		for id := range e.active {
			if _, ok := remotes[id]; !ok {
				delete(e.active, id)
			}
		}
		for id, endpoint := range remotes {
			if id == e.id {
				continue
			}
			e.active[id] = endpoint
		}
		e.mu.Unlock()
	}
	//log.Printf("federated %#v", e.active)
}

func (e *Endpoint) List() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var ret = map[string]string{}
	for id, endpoint := range e.active {
		ret[id] = endpoint
	}
	return ret
}

var endpoint = &Endpoint{
	id:      requests.GenId(),
	active:  make(map[string]string),
	mu:      new(sync.RWMutex),
	sess:    requests.New(requests.Timeout(1 * time.Second)),
	content: make(chan []byte, 100),
}

func Fedstart(ctx context.Context, listen string, join string) error {
	u, err := url.Parse(listen)
	if err != nil {
		return err
	}
	endpoint.id = requests.GenId()
	endpoint.url = u
	endpoint.active[endpoint.id] = listen
	if join != "" {
		endpoint.active[""] = join
	}
	mux := requests.NewServeMux(requests.URL(u.Host))
	mux.Route("/list", func(w http.ResponseWriter, r *http.Request) {
		b, err := json.Marshal(endpoint.List())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	})
	mux.Route("/send", func(w http.ResponseWriter, r *http.Request) {
		buf, err := requests.ParseBody(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		endpoint.content <- buf.Bytes()
	})
	mux.Route("/ping", func(w http.ResponseWriter, r *http.Request) {
		buf, err := requests.ParseBody(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body := make(map[string]string)
		if err := json.Unmarshal(buf.Bytes(), &body); err != nil {
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		endpoint.mu.Lock()
		for k, v := range body {
			endpoint.active[k] = v
		}
		endpoint.mu.Unlock()

		b, err := json.Marshal(endpoint.List())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	})
	s := requests.NewServer(ctx, mux, requests.OnStart(func(s *http.Server) {
		log.Printf("federate serve: %s\n", s.Addr)
	}))
	mux.Pprof()
	go func() {
		timer := time.NewTimer(0 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				endpoint.Ping()
				timer.Reset(3 * time.Second)
			}
		}
	}()
	endpoint.running = true
	return s.ListenAndServe()
}
