package network

import (
	"fmt"
	"net"
)

type Server struct {
	Addr   string
	Router *Router
}

func (s *Server) Run() {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("网关服务已启动, 监听:%s\n", s.Addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// //为每个连接创建一个session
		// session := NewSession(conn)
		// session.Start()
		session := &Session{
			Conn:     conn,
			SendChan: make(chan []byte, 100),
			Router:   s.Router,
		}
		session.Start()
	}
}
