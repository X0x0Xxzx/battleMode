package main

import (
	"battleMode/pb"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/containerd/containerd/protobuf/proto"
)

// Pack 模拟封包逻辑（必须与服务端协议完全一致）
func Pack(msgID uint32, data []byte) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	// 1. 写入长度 (Data长度 + MsgID长度4字节)
	// 使用 LittleEndian (小端序)
	length := uint32(len(data) + 4)
	if err := binary.Write(buffer, binary.LittleEndian, length); err != nil {
		return nil, err
	}
	// 2. 写入消息ID
	if err := binary.Write(buffer, binary.LittleEndian, msgID); err != nil {
		return nil, err
	}
	// 3. 写入内容
	if err := binary.Write(buffer, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func main() {
	// 1. 连接网关服务器
	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		fmt.Println("连接服务器失败:", err)
		return
	}
	defer conn.Close()
	fmt.Println("已连接到网关...")
	// 客户端模拟发送
	loginData := &pb.LoginReq{
		Username: "admin",
		Password: "password123",
	}
	// 序列化
	binaryData, _ := proto.Marshal(loginData)
	// 封包并发送
	msg, _ := Pack(1001, binaryData)
	conn.Write(msg)

	// 保持连接一会，观察服务端输出
	time.Sleep(5 * time.Second)
}
