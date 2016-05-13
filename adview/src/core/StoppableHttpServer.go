package core

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func StartHttpServer(mux *HandlerMux, host string, port int, quit chan int, quit_sig chan int) {
	go func() {
		handler := CreateHttpHandler(mux)
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			log.Fatal("Can not start Server", err)
			quit_sig <- 1
			return
		}

		//创建可停止的监听器
		stoppable := NewStoppableListener(listener)
		go func() {
			<-quit
			stoppable.Stop <- true
		}()

		srv := &http.Server{
			Handler:        handler,
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   5 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		log.Printf("Server Started at port:%d", port)
		srv.Serve(stoppable)

		log.Printf("Server at port %d Stopped", port)
		quit_sig <- 1
	}()
}
