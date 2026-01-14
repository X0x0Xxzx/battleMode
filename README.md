# ⚔️ BattleMode 分布式游戏后端框架

---

## 🚀 基础设施搭建

### 1. NATS 消息总线 (消息中枢)
NATS 负责各分布式服务（网关、登录、地图）之间的异步通信与 RPC 调用。

* **下载与安装 (Mac Intel)**
    ```bash
    # 下载二进制压缩包
    curl -L [https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-darwin-amd64.zip](https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-darwin-amd64.zip) -o nats.zip

    # 解压文件
    unzip nats.zip
    ```
* **启动服务**
    ```bash
    # 监听 4222 端口启动
    ./nats-server-v2.10.7-darwin-amd64/nats-server -p 4222
    ```
* **引入 Go 驱动**
    ```bash
    go get [github.com/nats-io/nats.go](https://github.com/nats-io/nats.go)
    ```

---

### 2. Redis 状态存储 (在线管理)
用于存储玩家的在线状态、Session 绑定信息以及全局热数据。

* **安装 (Homebrew)**
    ```bash
    brew install redis
    ```
* **管理命令**

| 运行模式 | 命令 |
| :--- | :--- |
| **后台运行** | `brew services start redis` |
| **前台调试** | `redis-server /usr/local/etc/redis.conf` |
| **停止服务** | `brew services stop redis` |
| **Test** | `./src/redis-server` |
| **Cli** | `redis-cli` |
---

## 🛠 开发工具配置

### Protobuf 协议编译
所有的跨服务通信协议都定义在 `.proto` 文件中。通过 Protobuf 编译器生成高效的二进制序列化 Go 代码。



* **编译命令**
    ```bash
    # 确保已安装 protoc 及 protoc-gen-go 插件
    protoc --go_out=. proto/message.proto
    ```
    > **⚠️ 注意:** 生成的代码将存放在 `pb/` 目录下，请勿手动修改生成的 `.pb.go` 文件。

---

## 📂 项目模块说明

| 模块名称 | 职责描述 |
| :--- | :--- |
| **`gateway-server`** | 维护 TCP 长连接、封包/拆包、消息路由转发 |
| **`login-service`** | 处理鉴权逻辑、数据库校验、分配玩家 ID |
| **`common`** | 公共组件库，包含 Redis 初始化、配置管理等 |
| **`pb`** | 自动生成的协议代码，包含消息结构体与序列化逻辑 |

---

## ⚙️ 系统运行流程

1.  **启动基础组件**: 确保 `nats-server` 和 `redis-server` 已正常运行。
2.  **启动逻辑服务**: 运行 `login-service` (监听 NATS 主题并处理登录逻辑)。
3.  **启动网关**: 运行 `gateway-server` (开启 TCP 端口并连接 NATS)。
4.  **客户端连接**: 客户端通过 8888 端口接入，发送序列化后的消息包。



---
## 2026.1.10
🛠 今日重构核心内容
1. 架构分层设计 (Architecture Decoupling)
为了解决代码耦合和无法独立打包的问题，项目采用了工业级标准的目录结构：

cmd/: 存放各服务的入口文件（gateway, login, client），支持独立编译打包。

internal/network/: 通用网络基础库，实现 TCP 拆解包、Session 管理、路由分发。不包含任何业务逻辑，可复用性强。

internal/gateway/: 网关业务逻辑层，处理玩家登录、下线广播、状态同步等。

common/: 全局公共组件，如 Redis 配置、基础工具函数等。

2. 双容器玩家管理 (Dual-Container System)
网关采用了双 sync.Map 容器管理连接，实现了物理连接与逻辑玩家的隔离：

WaitSessions: 存放已建立 TCP 连接但尚未验证身份的会话。使用 RemoteAddr 作为 Key。

OnlinePlayers: 存放已成功登录的玩家会话。使用 PlayerID 作为 Key，方便业务逻辑快速索引。

3. 会话状态机与并发安全
为每个 Session 引入了状态枚举（SessionState），并配合 sync/atomic 原子操作防止并发漏洞：

状态流转: Connected -> Logining -> InGame -> Closed。

防竞态: 有效拦截了玩家在网络波动时连续点击登录导致的重复协议处理。

4. 依赖注入与事件回调 (Dependency Injection)
通过在底层 Server 暴露 OnSessionStart 钩子，在 main 函数中动态注入业务逻辑：

解耦: network 包不再直接调用 gateway 业务。

灵活: 可以在不修改底层网络代码的情况下，通过 OnClose 回调实现自定义的下线清理逻辑（如 Redis 状态清除、全服公告）。

🚀 关键功能实现
[x] 登录安全校验: 通过 NATS 请求/响应模式与登录服务器交互，验证玩家身份。

[x] 下线全自动清理: 无论玩家是主动退出还是非正常断开，系统均能通过 sync.Once 保证业务清理逻辑（广播、Redis 移除）仅执行一次。

[x] 广播过滤: 实现了基于状态的广播系统，确保只有处于 InGame 状态的玩家能收到系统公告，避免对正在登录页面的用户造成干扰。

[x] 资源防泄露: 针对登录失败的连接实现了即时关闭策略，并预留了超时强制踢出机制。

📦 构建与运行
编译
Bash

# 构建网关服务
go build -o bin/gateway ./cmd/gateway

# 构建登录服务
go build -o bin/login ./cmd/login

# 构建测试客户端
go build -o bin/client ./cmd/client
运行环境
NATS Server: nats://127.0.0.1:4222

Redis: 127.0.0.1:6379

---