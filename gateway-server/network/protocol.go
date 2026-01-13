package network

import (
	"bytes"
	"encoding/binary"
)

type MsgPacker struct {
	Len   uint32
	MsgID uint32
	Data  []byte
}

// 封包:将数据转为二进制
func Pack(msgID uint32, data []byte) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	// 1. 写入长度 (Data长度 + MsgID长度4字节)
	if err := binary.Write(buffer, binary.LittleEndian, uint32(len(data)+4)); err != nil {
		return nil, err
	}
	// 2. 写入消息ID
	if err := binary.Write(buffer, binary.LittleEndian, msgID); err != nil {
		return nil, err
	}
	// 3. 写入data
	if err := binary.Write(buffer, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
