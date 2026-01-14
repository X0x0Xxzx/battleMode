package network

import "fmt"

// HandlerFunc 定义了处理消息的函数原型
type handlerFunc func(s *Session, data []byte)

// Router 消息路由管理器
type Router struct {
	// handlers 存放 MsgID 到 处理函数 的映射
	handlers map[uint32]handlerFunc
}

// 实例
func NewRouter() *Router {
	return &Router{
		handlers: make(map[uint32]handlerFunc),
	}
}

// Register 注册路由
func (r *Router) Register(msgID uint32, handler handlerFunc) {
	r.handlers[msgID] = handler
}

// 路由分发
func (r *Router) Route(msgID uint32, s *Session, data []byte) {
	if handler, ok := r.handlers[msgID]; ok {
		handler(s, data)
	} else {
		fmt.Println("未被定义的MsgID:", msgID)
	}
}
