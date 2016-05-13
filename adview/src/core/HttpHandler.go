package core

import (
	"strings"
	//"log"
	_ "net"
	"net/http"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request)
type HandlerMux map[string]HandlerFunc

type HttpHandler struct {
	mux *HandlerMux
}

//构造函数
func CreateHttpHandler(mux *HandlerMux) *HttpHandler {
	return &HttpHandler{mux}
}

//实现http.Handler接口
func (self *HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Path
	if h, o := (*self.mux)[uri]; o {
		h(w, r)
		return
	}

	for k, v := range *self.mux {
		if strings.HasPrefix(uri, k) {
			v(w, r)
			return
		}
	}

	w.WriteHeader(404)
}
