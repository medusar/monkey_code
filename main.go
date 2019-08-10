package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

// A brief description about the program
// 1. Create a http server with a simpile uri: /hello
// 2. Start some clients and make http requests to the http server concurrently
// 3. Shutdown the http server and the main thread sleep for 1 seconds.

// How to reproduce the problem
// Build the code and run the program 100 times or more
// You will see a panic: panic("Invalid request")

func main() {
	var shutdown int32

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&shutdown) == 0 {
			w.Write([]byte("ok"))
		} else {
			//If server was shutdown, response "fail"
			w.Write([]byte("fail"))
		}
	})

	s := &http.Server{
		Addr: "127.0.0.1:30001",
	}

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	//Create some clients
	ch := make(chan struct{}, 32)
	for i := 0; i < 256; i++ {
		go func() {
			for {
				<-ch
				res, err := http.NewRequest("GET", "http://127.0.0.1:30001/hello", nil)
				if err != nil {
					continue
				}
				resp, err := client.Do(res)
				if err != nil {
					continue
				}

				body, err := ioutil.ReadAll(resp.Body)
				resp.Body.Close()

				if err != nil {
					continue
				}

				if string(body) == "fail" {
					panic("Invalid request")
				}
			}
		}()
	}

	go func() {
		for {
			ch <- struct{}{}
		}
	}()

	//Shutdown http server after 2 seconds
	time.AfterFunc(time.Second*2, func() {
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

		//Shutdown the http server
		if err := s.Shutdown(ctx); err != nil {
			panic(err)
		}

		//Set flag:shutdown to 1
		atomic.StoreInt32(&shutdown, 1)
	})

	if err := s.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			panic(err)
		}
	}

	time.Sleep(1 * time.Second)
	log.Println("end")

}
