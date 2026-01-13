package network

import (
	"battleMode/common"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

type Session struct {
	Conn     net.Conn
	SendChan chan []byte // 异步发送信息通道
	Router   *Router
	PlayerID string    // 玩家ID，登录成功后绑定
	once     sync.Once // 确保关闭逻辑只执行一次
}

func NewSession(conn net.Conn, router *Router) *Session {
	return &Session{
		Conn:     conn,
		SendChan: make(chan []byte, 100),
		Router:   router,
	}
}

func (s *Session) Start() {
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

// Close
func (s *Session) Close() {
	s.once.Do(func() {
		fmt.Println("[会话] 正在关闭连接并清理资源...")
		// 1. 关闭网络连接，这会触发 readLoop 里的 io.ReadFull 报错并退出循环
		if s.Conn != nil {
			s.Conn.Close()
		}
		// 2. 关闭管道，这会触发 writeLoop 里的 range 退出循环
		close(s.SendChan)

		if s.PlayerID != "" {
			fmt.Printf("[网关] 玩家 %s 下线，正在清理 Redis 状态\n", s.PlayerID)
			// 在这里调用 Redis 移除该玩家的在线记录
			// common.RDB.Del(ctx, "online:"+s.PlayerID)
			// 这里直接调用 common 包的方法
			err := common.ClearPlayerOnline(s.PlayerID)
			if err != nil {
				fmt.Printf("清理 Redis 玩家 %s 状态失败: %v\n", s.PlayerID, err)
			} else {
				fmt.Printf("玩家 %s 状态已从 Redis 移除\n", s.PlayerID)
			}
		}
	})
}
