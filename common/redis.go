package common

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RDB *redis.Client
	Ctx = context.Background()
)

var (
	reids_online_key = "player:online"
)

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	// 测试连接是否连通
	_, err := RDB.Ping(Ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("无法连接 Redis: %v", err))
	}
	fmt.Println("redis服务连通")
}

// SetPlayerOnline 设置玩家在线状态（绑定网关ID）
func SetPlayerOnline(playerID string, gatewayID string) error {
	key := fmt.Sprintf(reids_online_key, ":%s", playerID)
	// 设置为 24 小时过期，防止意外死号永久留存，正常逻辑应由 Close 时删除
	return RDB.Set(Ctx, key, gatewayID, 24*time.Hour).Err()
}

// ClearPlayerOnline 清理玩家在线状态
func ClearPlayerOnline(playerID string) error {
	key := fmt.Sprintf(reids_online_key, ":%s", playerID)
	return RDB.Del(Ctx, key).Err()
}

// GetPlayerGateway 获取玩家所在的网关ID
func GetPlayerGateway(playerID string) (string, error) {
	key := fmt.Sprintf(reids_online_key, ":%s", playerID)
	return RDB.Get(Ctx, key).Result()
}
