package network

import (
	"fmt"
	"net"
	"sync"
)

type Server struct {
	Addr   string
	Router *Router
	// 加入session管理
	WaitSessions sync.Map

	// 容器 2：存放所有已验证身份的玩家
	// Key: PlayerID (string), Value: *Session
	OnlinePlayers sync.Map

	OnSessionStart func(s *Session)
}

func (server *Server) Run() {
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("网关服务已启动, 监听:%s\n", server.Addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// session := NewSession(conn)
		// session.Start()
		session := NewSession(conn, server.Router, server)

		remoteAddr := conn.RemoteAddr().String()
		// 加入到等待登录
		server.WaitSessions.Store(remoteAddr, session)

		// 注入业务逻辑
		if server.OnSessionStart != nil {
			server.OnSessionStart(session)
		}
		session.Start()
	}
}
