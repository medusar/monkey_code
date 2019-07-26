package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"
)

func main() {
	var shutdown int32
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&shutdown) == 0 {
			w.Write([]byte("ok"))
		} else {
			w.Write([]byte("fail"))
		}
	})

	s := &http.Server{
		Addr: "127.0.0.1:30001",
	}
	go func(){
		if err := s.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				panic(err)
			}
		}
	}()

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     true,
		},

	}
	go func(){
		ch := make(chan struct{}, 32)
		for i := 0; i < 256; i++ {
			go func() {
				for {
					<-ch
					res, err := http.NewRequest("GET", "http://127.0.0.1:30001/hello", nil)
					if err == nil {
						resp, err := client.Do(res)
						if err == nil {
							body, err := ioutil.ReadAll(resp.Body)
							resp.Body.Close()
							if err == nil {
								if string(body) == "fail" {
									panic("Invalid request")
								}
							}
						}
					}
				}
			}()
		}
		for {
			ch <- struct{}{}
		}
	}()
	time.Sleep(1 * time.Second)
	ctx, _ := context.WithTimeout(context.Background(), 30 * time.Second)

	if err := s.Shutdown(ctx); err != nil {
		panic(err)
	}
	atomic.StoreInt32(&shutdown, 1)
	fmt.Println("shutdown..", s.Addr)

	time.Sleep(1 * time.Second)
}
