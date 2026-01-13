package main

import (
	"battleMode/common"
	"battleMode/gateway-server/network"
	"fmt"
	"time" // 引入官方库

	"battleMode/pb"

	"github.com/containerd/containerd/protobuf/proto"
	"github.com/nats-io/nats.go"
)

var natsConn *nats.Conn

// 处理登录
func HandleLogin(s *network.Session, data []byte) {
	req := &pb.LoginReq{}
	err := proto.Unmarshal(data, req)
	if err != nil {
		fmt.Println("protobuf解析失败:", err)
		return
	}

	fmt.Printf("[网关] 收到登录请求: 用户名=%s\n", req.Username)

	if natsConn == nil || !natsConn.IsConnected() {
		fmt.Println("[网关] 错误: NATS 未连接，请检查 nats-server 是否启动")
		return
	}
	// 使用 NATS 的 Request 模式（请求并等待响应）
	// 参数1: 主题 (Subject) - 登录服务会监听这个
	// 参数2: 数据
	// 参数3: 超时时间
	msg, err := natsConn.Request("service.login", data, 2*time.Second)
	if err != nil {
		fmt.Println("[网关]消息转发失败:", err)
	}

	resp := &pb.LoginResp{}
	err = proto.Unmarshal(msg.Data, resp)
	if err != nil {
		fmt.Println("[gate-way] 解析登录响应失败:", err)
		return
	}

	// 判断登录状态
	if resp.Code == 0 {
		s.PlayerID = resp.PlayerId
		fmt.Printf("[网关] 玩家 %s 验证通过，已绑定到 Session\n", s.PlayerID)
		err := common.SetPlayerOnline(s.PlayerID, "gateway_01")
		if err != nil {
			fmt.Println("Redis 状态写入失败:", err)
		}
		fmt.Printf("[网关] 玩家 %s 已上线并登记到 Redis\n", s.PlayerID)
	}

}

func main() {
	// 连接到NATS Server
	var err error
	natsConn, err = nats.Connect("nats://127.0.0.1:4222")
	if err != nil {
		fmt.Printf("NATS 连接失败: %v\n", err)
		panic(err)
	}
	defer natsConn.Close()

	// 初始化 Redis
	common.InitRedis()

	// 初始化路由
	r := network.NewRouter()

	//  注册业务逻辑
	r.Register(1001, HandleLogin)

	//  启动 Server
	server := &network.Server{
		Addr:   ":8888",
		Router: r,
	}
	server.Run()
}
