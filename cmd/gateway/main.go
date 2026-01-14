package main

import (
	"battleMode/common"
	"battleMode/internal/gateway"
	"battleMode/internal/network"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
)

var natsConn *nats.Conn

func main() {
	// 连接到NATS Server
	var err error
	natsConn, err = nats.Connect("nats://127.0.0.1:4222")
	if err != nil {
		fmt.Printf("NATS 连接失败: %v\n", err)
		panic(err)
	}
	defer natsConn.Close()

	gwLogic := gateway.NewGatewayLogic(natsConn)
	// 初始化 Redis
	common.InitRedis()

	// 初始化路由
	r := network.NewRouter()

	//  注册业务逻辑
	r.Register(1001, gwLogic.HandleLogin)

	//  启动 Server
	server := &network.Server{
		Addr:   ":8888",
		Router: r,
	}

	// 在这里注入逻辑！
	server.OnSessionStart = func(s *network.Session) {
		// 绑定下线处理
		s.OnClose = func(sess *network.Session) {
			gwLogic.HandleLogout(sess)
		}

		go func() {
			// 给 30 秒宽限期
			time.Sleep(60 * time.Second)

			// 30 秒后检查，如果还没进入游戏状态，直接踢掉
			if atomic.LoadInt32((*int32)(&s.State)) != int32(network.StateInGame) {
				fmt.Printf("[安全] 连接 %s 登录超时，强制断开\n", s.Conn.RemoteAddr().String())
				s.Close()
			}
		}()
	}
	server.Run()
}
