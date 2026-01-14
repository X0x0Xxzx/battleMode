package gateway

import (
	"battleMode/common"
	"battleMode/internal/network"
	"battleMode/pb"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/containerd/containerd/protobuf/proto"
	"github.com/nats-io/nats.go"
)

type GatewayLogic struct {
	natcsConn *nats.Conn
}

func NewGatewayLogic(nc *nats.Conn) *GatewayLogic {
	return &GatewayLogic{natcsConn: nc}
}

// 处理登录
func (gl *GatewayLogic) HandleLogin(session *network.Session, data []byte) {
	req := &pb.LoginReq{}
	err := proto.Unmarshal(data, req)
	if err != nil {
		fmt.Println("protobuf解析失败:", err)
		return
	}

	fmt.Printf("[网关] 收到登录请求: 用户名=%s\n", req.Username)

	// 1. 原子检查状态
	if atomic.LoadInt32((*int32)(&session.State)) != int32(network.StateConnected) {
		return // 正在登录或已登录，拒绝重复请求
	}

	// 2. 切换为“登录中”
	atomic.StoreInt32((*int32)(&session.State), int32(network.StateLogining))

	if gl.natcsConn == nil || !gl.natcsConn.IsConnected() {
		fmt.Println("[网关] 错误: NATS 未连接，请检查 nats-server 是否启动")
		return
	}
	// 使用 NATS 的 Request 模式（请求并等待响应）
	// 参数1: 主题 (Subject) - 登录服务会监听这个
	// 参数2: 数据
	// 参数3: 超时时间
	msg, err := gl.natcsConn.Request("service.login", data, 2*time.Second)
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
		session.PlayerID = resp.PlayerId
		session.IsLogin = true // 标记为已登录
		atomic.StoreInt32((*int32)(&session.State), int32(network.StateInGame))
		// 登录处理
		gl.loginSuccessOp(session)
	} else {
		fmt.Printf("[网关] 玩家登录失败: %s, 错误码: %d\n", req.Username, resp.Code)
		atomic.AddInt32(&session.LoginAttempts, 1)

		if session.LoginAttempts >= 3 {
			fmt.Println("[网关] 登录尝试次数过多，强制断开")
			session.Close()
			return
		}

		// 校验失败，退回初始状态或断开
		atomic.StoreInt32((*int32)(&session.State), int32(network.StateConnected))
	}
}

// 处理下线
func (gl *GatewayLogic) HandleLogout(session *network.Session) {
	// 获取当前状态
	state := session.GetState()

	switch state {
	case network.StateInGame:
		// 踢下线
		if session.PlayerID == "" {
			fmt.Printf("[逻辑] 游客连接 %s 断开，无需广播，清理完毕。\n", session.Conn.RemoteAddr().String())
			return
		}
		// 从登录容器删除
		session.Server.OnlinePlayers.Delete(session.PlayerID)
		gl.broadcastLoginNotice(session, fmt.Sprintf("玩家 [%s] 下线\n", session.PlayerID))
		common.ClearPlayerOnline(session.PlayerID)
	default:
		// 从待验证容器删除
		session.Server.WaitSessions.Delete(session.Conn.RemoteAddr().String())
	}

}

// 登录处理
func (gl *GatewayLogic) loginSuccessOp(session *network.Session) {
	// 保存登录数据到Redis
	gl.loginDataToRedis(session)

	// 广播玩家进入消息
	gl.broadcastLoginNotice(session, fmt.Sprintf("玩家[%s] 登录\n", session.PlayerID))

	// 从等待列表移除
	session.Server.WaitSessions.Delete(session.Conn.RemoteAddr().String())
	// 2. 标记状态
	session.IsLogin = true
	// 添加session数据到map
	session.Server.OnlinePlayers.Store(session.PlayerID, session)

}

// 消息广播
func (gl *GatewayLogic) broadcastLoginNotice(session *network.Session, broadcastMsg string) {
	fmt.Println(broadcastMsg)
	notice := &pb.LoginNotice{
		Message: broadcastMsg,
	}
	// 封装消息
	msg, _ := proto.Marshal(notice)

	// 封包
	broadcastPacket, _ := network.Pack(2001, msg)

	// 广播消息
	session.Server.OnlinePlayers.Range(func(key, value interface{}) bool {
		targetSession := value.(*network.Session)
		targetSession.SendChan <- broadcastPacket
		return true
	})

}

// 保存登录数据到数据库
func (gl *GatewayLogic) loginDataToRedis(session *network.Session) {
	fmt.Printf("[网关] 玩家 %s 验证通过，已绑定到 Session\n", session.PlayerID)
	err := common.SetPlayerOnline(session.PlayerID, "gateway_01")
	if err != nil {
		fmt.Println("[网关] 玩家Redis 状态写入失败:", err)
	}
	fmt.Printf("[网关] 玩家 %s 已上线并登记到 Redis\n", session.PlayerID)
}
