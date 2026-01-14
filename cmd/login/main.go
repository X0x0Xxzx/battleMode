package main

import (
	"battleMode/pb"
	"fmt"
	"runtime"

	"github.com/containerd/containerd/protobuf/proto"
	"github.com/nats-io/nats.go"
)

func main() {
	// 连接NATS
	natsconn, err := nats.Connect("nats://127.0.0.1:4222")
	if err != nil {
		panic(err)
	}

	fmt.Println("登录服务已启动(LOGIN-SERVER), 等待消息...")

	// 订阅 "service.login" 主题
	// 使用 QueueSubscribe 可以实现负载均衡（如果有多个登录服，只有一个会收到消息）
	natsconn.QueueSubscribe("service.login", "login_group", func(m *nats.Msg) {
		fmt.Printf("[LOGIN-SERVER] 接收请求: %s\n", string(m.Data))
		// 解析
		req := &pb.LoginReq{}
		proto.Unmarshal(m.Data, req)

		// 相应结构体
		resp := &pb.LoginResp{
			Code:     0,
			Message:  "Login Success",
			Token:    "jwt_token_example",
			PlayerId: req.Username,
		}

		// 序列化
		out, _ := proto.Marshal(resp)
		m.Respond(out)
	})

	// 保持运行
	runtime.Goexit()
}
