# IM Demo - 即时通讯系统演示

一个基于 Go + Socket.IO + Redis 的现代化即时通讯系统演示，支持 Web 浏览器、iOS/Android App 多平台接入。

## 功能特性

- ✅ **实时消息传输**：基于 Socket.IO 实现低延迟实时通信
- ✅ **多消息类型**：支持文字、文件、图片等多种消息类型
- ✅ **房间系统**：支持多聊天室，用户可自由切换
- ✅ **文件传输**：支持文件上传下载，最大 10MB
- ✅ **在线状态**：实时显示用户在线状态
- ✅ **输入指示**：显示用户正在输入状态
- ✅ **分布式架构**：使用 Redis 支持多服务器实例部署
- ✅ **跨平台支持**：Web 浏览器、移动 App 均可接入
- ✅ **优雅关闭**：支持优雅关闭和重启
- ✅ **现代化 UI**：响应式设计，支持桌面和移动端

## 技术栈

### 后端
- **Go 1.21+**：高性能服务器语言
- **Gin**：轻量级 Web 框架
- **Socket.IO**：实时双向通信
- **Redis**：分布式缓存和消息队列
- **Logrus**：结构化日志记录

### 前端
- **HTML5/CSS3**：现代化响应式设计
- **JavaScript ES6+**：客户端交互逻辑
- **Socket.IO Client**：WebSocket 客户端

## 项目结构

```
im-demo/
├── cmd/
│   └── server/
│       └── main.go           # 服务器入口点
├── internal/
│   ├── config/
│   │   └── config.go         # 配置管理
│   ├── handlers/
│   │   └── socketio.go       # Socket.IO 处理器
│   ├── models/
│   │   └── message.go        # 数据模型
│   └── services/
│       └── redis.go          # Redis 服务
├── web/
│   ├── index.html            # Web 客户端页面
│   └── client.js             # 客户端 JavaScript
├── config.yaml               # 配置文件
├── go.mod                    # Go 模块文件
└── README.md                 # 说明文档
```

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- Redis 6.0 或更高版本
- 现代化浏览器（Chrome、Firefox、Safari、Edge）

### 安装步骤

1. **克隆项目**
   ```bash
   git clone <your-repo-url>
   cd im-demo
   ```

2. **安装依赖**
   ```bash
   go mod download
   ```

3. **启动 Redis 服务器**
   ```bash
   # macOS (使用 Homebrew)
   brew install redis
   brew services start redis
   
   # Ubuntu/Debian
   sudo apt-get install redis-server
   sudo systemctl start redis
   
   # Docker
   docker run -d -p 6379:6379 redis:7-alpine
   ```

4. **配置环境变量**（可选）
   ```bash
   export PORT=8080
   export REDIS_ADDR=localhost:6379
   export LOG_LEVEL=info
   ```

5. **运行服务器**
   ```bash
   go run cmd/server/main.go
   ```

6. **访问 Web 客户端**
   
   打开浏览器访问：`http://localhost`

## 配置说明

### 配置文件 (config.yaml)

```yaml
# 服务器配置
server:
  port: 8080
  host: localhost
  env: development

# Redis 配置
redis:
  addr: localhost:6379
  password: ""
  db: 0

# Socket.IO 配置
socketio:
  cors_origins: "*"
  ping_timeout: 60
  ping_interval: 25

# 文件上传配置
upload:
  max_file_size: 10485760  # 10MB
  upload_dir: uploads/

# 日志配置
logging:
  level: info
  format: json
```

### 环境变量

配置文件中的任何值都可以通过环境变量覆盖：

- `PORT`: 服务器端口
- `HOST`: 服务器主机名
- `ENV`: 运行环境 (development/production)
- `REDIS_ADDR`: Redis 服务器地址
- `REDIS_PASSWORD`: Redis 密码
- `REDIS_DB`: Redis 数据库编号
- `MAX_FILE_SIZE`: 最大文件大小
- `UPLOAD_DIR`: 文件上传目录
- `LOG_LEVEL`: 日志级别

## API 文档

### Socket.IO 事件

#### 客户端发送事件

| 事件名 | 数据格式 | 说明 |
|--------|----------|------|
| `join` | `{userId, userName, avatar}` | 用户加入系统 |
| `join_room` | `{roomId, userId}` | 加入聊天室 |
| `leave_room` | `{roomId, userId}` | 离开聊天室 |
| `message` | `{type, content, sender, roomId, receiver}` | 发送消息 |
| `file_upload` | `{fileName, fileData, fileType, sender, roomId}` | 上传文件 |
| `typing` | `{userId, roomId}` | 开始输入 |
| `stop_typing` | `{userId, roomId}` | 停止输入 |

#### 服务器发送事件

| 事件名 | 数据格式 | 说明 |
|--------|----------|------|
| `joined` | `{userId, userName, status}` | 加入确认 |
| `message` | `Message` | 接收消息 |
| `user_status` | `{userId, status}` | 用户状态变更 |
| `room_joined` | `{roomId, userId}` | 房间加入确认 |
| `user_joined_room` | `{userId, roomId}` | 用户加入房间 |
| `user_left_room` | `{userId, roomId}` | 用户离开房间 |
| `typing` | `{userId, roomId}` | 用户正在输入 |
| `stop_typing` | `{userId, roomId}` | 用户停止输入 |
| `error` | `{message}` | 错误消息 |

### HTTP API

#### 文件上传
```
POST /api/upload
Content-Type: multipart/form-data

Form Data:
- file: 文件内容
```

#### 获取房间成员
```
GET /api/rooms/:roomId/members
```

#### 获取消息
```
GET /api/messages/:messageId
```

#### 健康检查
```
GET /health
```

## 部署指南

### 生产环境部署

1. **构建应用**
   ```bash
   go build -o im-server cmd/server/main.go
   ```

2. **配置生产环境**
   ```yaml
   server:
     port: 8080
     host: 0.0.0.0
     env: production
   
   redis:
     addr: your-redis-server:6379
     password: your-redis-password
   
   logging:
     level: warn
     format: json
   ```

3. **使用 systemd 管理服务**
   ```ini
   [Unit]
   Description=IM Demo Server
   After=network.target
   
   [Service]
   Type=simple
   User=www-data
   WorkingDirectory=/opt/im-demo
   ExecStart=/opt/im-demo/im-server
   Restart=always
   RestartSec=10
   
   [Install]
   WantedBy=multi-user.target
   ```

4. **Nginx 反向代理**
   ```nginx
   server {
       listen 80;
       server_name your-domain.com;
       
       location / {
           proxy_pass http://localhost:8080;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_cache_bypass $http_upgrade;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```

### Docker 部署

1. **创建 Dockerfile**
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   
   WORKDIR /app
   COPY go.mod go.sum ./
   RUN go mod download
   
   COPY . .
   RUN go build -o im-server cmd/server/main.go
   
   FROM alpine:latest
   RUN apk --no-cache add ca-certificates
   WORKDIR /root/
   
   COPY --from=builder /app/im-server .
   COPY --from=builder /app/config.yaml .
   COPY --from=builder /app/web ./web
   
   EXPOSE 8080
   CMD ["./im-server"]
   ```

2. **Docker Compose**
   ```yaml
   version: '3.8'
   services:
     im-server:
       build: .
       ports:
         - "8080:8080"
       depends_on:
         - redis
       environment:
         - REDIS_ADDR=redis:6379
         - ENV=production
     
     redis:
       image: redis:7-alpine
       ports:
         - "6379:6379"
       volumes:
         - redis_data:/data
   
   volumes:
     redis_data:
   ```

## 移动端接入

### iOS 接入示例

```swift
import SocketIO

class IMClient {
    private var manager: SocketManager
    private var socket: SocketIOClient
    
    init() {
        manager = SocketManager(socketURL: URL(string: "http://your-server:8080")!)
        socket = manager.defaultSocket
        
        socket.on(clientEvent: .connect) { data, ack in
            print("Connected to server")
            self.joinUser()
        }
        
        socket.on("message") { data, ack in
            // Handle incoming message
            if let messageData = data[0] as? [String: Any] {
                self.handleMessage(messageData)
            }
        }
    }
    
    func connect() {
        socket.connect()
    }
    
    func joinUser() {
        socket.emit("join", [
            "userId": "user123",
            "userName": "iOS User",
            "avatar": ""
        ])
    }
    
    func sendMessage(content: String, roomId: String) {
        socket.emit("message", [
            "type": "text",
            "content": content,
            "sender": "user123",
            "roomId": roomId
        ])
    }
}
```

### Android 接入示例

```java
import io.socket.client.IO;
import io.socket.client.Socket;
import org.json.JSONObject;

public class IMClient {
    private Socket socket;
    
    public void connect() {
        try {
            socket = IO.socket("http://your-server:8080");
            
            socket.on(Socket.EVENT_CONNECT, args -> {
                System.out.println("Connected to server");
                joinUser();
            });
            
            socket.on("message", args -> {
                JSONObject message = (JSONObject) args[0];
                handleMessage(message);
            });
            
            socket.connect();
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
    
    private void joinUser() {
        try {
            JSONObject userData = new JSONObject();
            userData.put("userId", "user123");
            userData.put("userName", "Android User");
            userData.put("avatar", "");
            socket.emit("join", userData);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
    
    public void sendMessage(String content, String roomId) {
        try {
            JSONObject messageData = new JSONObject();
            messageData.put("type", "text");
            messageData.put("content", content);
            messageData.put("sender", "user123");
            messageData.put("roomId", roomId);
            socket.emit("message", messageData);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}
```

## 性能优化

### 服务器端

1. **连接池配置**
   ```go
   // Redis 连接池
   client := redis.NewClient(&redis.Options{
       Addr:         "localhost:6379",
       PoolSize:     10,
       MinIdleConns: 5,
       MaxRetries:   3,
   })
   ```

2. **Socket.IO 优化**
   ```go
   server := socketio.NewServer(&engineio.Options{
       PingTimeout:  60 * time.Second,
       PingInterval: 25 * time.Second,
       Transports:   []transport.Transport{websocket.Default},
   })
   ```

3. **消息压缩**
   ```go
   // 在生产环境中启用消息压缩
   server.SetCompressionLevel(6)
   ```

### 客户端

1. **连接优化**
   ```javascript
   const socket = io('/', {
       transports: ['websocket'],
       upgrade: false,
       rememberUpgrade: false,
       timeout: 20000,
       autoConnect: true,
       reconnection: true,
       reconnectionDelay: 1000,
       reconnectionDelayMax: 5000,
       reconnectionAttempts: 5
   });
   ```

2. **消息缓存**
   ```javascript
   // 客户端消息缓存
   class MessageCache {
       constructor(maxSize = 1000) {
           this.messages = new Map();
           this.maxSize = maxSize;
       }
       
       add(message) {
           if (this.messages.size >= this.maxSize) {
               const firstKey = this.messages.keys().next().value;
               this.messages.delete(firstKey);
           }
           this.messages.set(message.id, message);
       }
   }
   ```

## 监控和日志

### 日志配置

```yaml
logging:
  level: info
  format: json
  file: /var/log/im-demo/app.log
  max_size: 100MB
  max_age: 30
  max_backups: 10
```

### 监控指标

- 连接数量
- 消息发送频率
- 错误率
- 响应时间
- 内存使用率
- Redis 连接状态

## 故障排除

### 常见问题

1. **连接失败**
   - 检查 Redis 服务是否运行
   - 确认防火墙设置
   - 验证网络连接

2. **文件上传失败**
   - 检查文件大小限制
   - 确认上传目录权限
   - 验证磁盘空间

3. **消息丢失**
   - 检查 Redis 连接状态
   - 确认网络稳定性
   - 查看服务器日志

### 调试模式

```bash
# 开启调试日志
export LOG_LEVEL=debug
go run cmd/server/main.go
```

## 贡献指南

1. Fork 本项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 联系方式

- 项目主页：https://github.com/your-username/im-demo
- 问题反馈：https://github.com/your-username/im-demo/issues
- 邮箱：your-email@example.com

---

⭐ 如果这个项目对您有帮助，请给一个 Star！ 