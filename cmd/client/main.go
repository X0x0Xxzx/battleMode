package main

import (
	"battleMode/pb"
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		fmt.Printf("无法连接网关: %v\n", err)
		return
	}
	defer conn.Close()

	// 1. 开启异步接收协程
	go readLoop(conn)

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("======= 欢迎来到 BattleMode =======")

	for {
		fmt.Print("\n请输入指令 [login|exit]: ")
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)

		if cmd == "exit" {
			break
		}

		if cmd == "login" {
			// 2. 自定义输入账号
			fmt.Print("账号: ")
			username, _ := reader.ReadString('\n')
			username = strings.TrimSpace(username)

			fmt.Print("密码: ")
			password, _ := reader.ReadString('\n')
			password = strings.TrimSpace(password)

			// 3. 发送登录请求
			sendLogin(conn, username, password)
		}
	}
}

// 封包并发送登录请求
func sendLogin(conn net.Conn, user, pwd string) {
	req := &pb.LoginReq{
		Username: user,
		Password: pwd,
	}
	data, _ := proto.Marshal(req)

	// 组装包体: MsgID(1001) + Data
	msgID := uint32(1001)
	body := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(body[:4], msgID)
	copy(body[4:], data)

	// 组装全包: Length(4字节) + Body
	length := uint32(len(body))
	packet := make([]byte, 4+len(body))
	binary.LittleEndian.PutUint32(packet[:4], length)
	copy(packet[4:], body)

	conn.Write(packet)
	fmt.Printf("[系统] 已提交登录请求: %s\n", user)
}

func readLoop(conn net.Conn) {
	for {
		head := make([]byte, 4)
		if _, err := io.ReadFull(conn, head); err != nil {
			fmt.Println("\n[系统] 连接断开")
			os.Exit(0)
		}
		length := binary.LittleEndian.Uint32(head)
		body := make([]byte, length)
		io.ReadFull(conn, body)

		msgID := binary.LittleEndian.Uint32(body[:4])
		data := body[4:]

		switch msgID {
		case 1001:
			resp := &pb.LoginResp{}
			proto.Unmarshal(data, resp)
			if resp.Code == 0 {
				fmt.Printf("\n[结果] 登录成功！你的 PlayerID 是: %s\n", resp.PlayerId)
			} else {
				fmt.Printf("\n[结果] 登录失败: %s\n", resp.Message)
			}
		case 2001:
			// 处理登录广播通知
			notice := &pb.LoginNotice{}
			proto.Unmarshal(data, notice)
			fmt.Printf("\n[系统公告] %s\n", notice.Message)
		default:
			fmt.Printf("\n[系统] 收到未知消息 ID: %d\n", msgID)
		}
	}
}
