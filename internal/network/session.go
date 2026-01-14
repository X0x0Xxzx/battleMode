package network

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

type SessionState int32

const (
	StateConnected SessionState = iota // 刚连接
	StateLogining                      // 登录中（防止连点）
	StateInGame                        // 已登录
	StateClosed                        // 已关闭
)

type Session struct {
	Conn          net.Conn
	SendChan      chan []byte // 异步发送信息通道
	Router        *Router
	State         SessionState
	PlayerID      string    // 玩家ID，登录成功后绑定
	Server        *Server   // 新增：持有 Server 引用
	once          sync.Once // 确保关闭逻辑只执行一次
	IsLogin       bool      // 新增：标记是否已登录
	LoginAttempts int32     // 登录尝试次数
	OnClose       func(*Session)
}

func NewSession(conn net.Conn, router *Router, s *Server) *Session {
	return &Session{
		Conn:          conn,
		SendChan:      make(chan []byte, 100),
		Router:        router,
		Server:        s,     // 将当前 server 传给 session
		IsLogin:       false, // 加入登录态
		LoginAttempts: 0,     // 登录尝试次数
	}
}

func (s *Session) Start() {
	fmt.Printf("[会话] 开始处理连接: %s\n", s.Conn.RemoteAddr().String())
	go s.readLoop()
	go s.writeLoop()
}

// 读循环
func (s *Session) readLoop() {
	defer s.Close() // 退出循环时关闭连接

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[会话] 运行时崩溃，已拦截: %v\n", r)
		}
	}()

	for {
		// 1. 读取包头 (4字节长度)
		head := make([]byte, 4)
		if _, err := io.ReadFull(s.Conn, head); err != nil {
			// 这里通常是玩家下线，打印 EOF 或 connection reset
			fmt.Printf("[会话] 连接断开: %v\n", err)
			break
		}
		length := binary.LittleEndian.Uint32(head)

		// 2. 读取包体 (MsgID + Data)
		body := make([]byte, length)
		if _, err := io.ReadFull(s.Conn, body); err != nil {
			fmt.Println("[会话] 读取包体失败:", err)
			break
		}

		// 3. 解析 ID 和数据
		msgID := binary.LittleEndian.Uint32(body[:4])
		data := body[4:]

		// 4. 路由分发
		if s.Router != nil {
			s.Router.Route(msgID, s, data)
		}
	}
}

// 写循环
func (s *Session) writeLoop() {
	// 当 SendChan 被 close 且数据取完后，循环会自动退出
	for data := range s.SendChan {
		if _, err := s.Conn.Write(data); err != nil {
			fmt.Println("[会话] 发送数据失败:", err)
			break
		}
	}
}

func (s *Session) Close() {
	s.once.Do(func() {
		fmt.Printf("[会话] 正在关闭连接 (PlayerID: %s, Addr: %s)...\n", s.PlayerID, s.Conn.RemoteAddr().String())

		// 交给业务层处理下线
		if s.OnClose != nil {
			s.OnClose(s)
		}
		// // 从server容器中移除
		// if s.Server != nil {
		// 	s.Server.WaitSessions.Delete(s.Conn.RemoteAddr().String())
		// 	if s.PlayerID != "" {
		// 		s.Server.OnlinePlayers.Delete(s.PlayerID)
		// 	}
		// }

		// 这会让 readLoop 阻塞的 Read 立即返回 err，从而结束 readLoop
		if s.Conn != nil {
			s.Conn.Close()
		}

		close(s.SendChan)

		// // 处理业务清理
		// if s.PlayerID != "" {
		// 	// 清理 Redis
		// 	err := common.ClearPlayerOnline(s.PlayerID)
		// 	if err != nil {
		// 		fmt.Printf("[警告] 清理 Redis 玩家 %s 状态失败: %v\n", s.PlayerID, err)
		// 	} else {
		// 		fmt.Printf("[网关] 玩家 %s 状态已从 Redis 移除\n", s.PlayerID)
		// 	}
		// }
	})
}

func (s *Session) GetState() SessionState {
	return s.State
}
