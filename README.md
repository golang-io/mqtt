# MQTT Server Implementation

## 项目概述

这是一个完整的MQTT服务器实现，支持MQTT v3.1.1和v5.0协议。

## 协议支持

### MQTT v3.1.1 (OASIS Standard, 29 October 2014)
- 完整的14种控制报文支持
- 三种QoS等级：0（最多一次）、1（至少一次）、2（恰好一次）
- 遗嘱消息、用户名密码认证
- 会话持久化

### MQTT v5.0 (OASIS Standard, 7 March 2019)
- 在v3.1.1基础上增加了属性系统
- 新增AUTH报文用于扩展认证
- 支持会话过期间隔、接收最大值、最大报文长度等高级特性
- 主题别名、订阅标识符等优化功能

## 代码注释规范

本项目已按照MQTT协议规范为所有代码添加了详细注释，包含：

### 1. 协议章节索引
- 每个字段都标注了对应的协议章节号
- 便于开发者快速查找协议原文
- 例如：`参考章节: 3.1.2.1 Protocol Name`

### 2. 版本差异说明
- 明确标注v3.1.1和v5.0的区别
- 突出新增功能和行为变化
- 帮助开发者理解协议演进

### 3. 字段详细说明
- 位置：在报文中的具体位置
- 类型：数据类型和编码方式
- 含义：字段的具体作用
- 约束：协议规定的限制条件

### 4. 行为差异说明
- 不同版本协议的行为差异
- 错误处理方式的变化
- 兼容性注意事项

## 已注释的核心文件

### packet/packet.go
- MQTT控制报文通用接口
- 报文类型分发逻辑
- 版本差异说明

### packet/0x0.fixed_header.go
- 固定报头结构
- 标志位验证规则
- 剩余长度编码

### packet/0x1.connect.go
- CONNECT报文完整实现
- 连接标志位解析
- v5.0属性系统支持

## 使用示例

```go
// 创建MQTT v5.0客户端连接
connect := &packet.CONNECT{
    FixedHeader: &packet.FixedHeader{Version: packet.VERSION500},
    ClientID: "test-client",
    KeepAlive: 60,
    Props: &packet.ConnectProperties{
        SessionExpiryInterval: 3600, // 1小时会话过期
        ReceiveMaximum: 100,         // 最大接收100条消息
    },
}

// 序列化报文
var buf bytes.Buffer
err := connect.Pack(&buf)
```

## 开发指南

### 添加新功能
1. 参考MQTT协议文档确定功能规范
2. 在代码中添加详细的协议章节注释
3. 说明v3.1.1和v5.0的差异
4. 添加字段位置、类型、含义等说明

### 协议兼容性
- 优先保证v3.1.1兼容性
- v5.0功能作为可选扩展
- 明确标注版本差异

## 协议文档参考

- [MQTT v3.1.1 Specification](docs/MQTT%20Version%203.1.1.html)
- [MQTT v5.0 Specification](docs/MQTT%20Version%205.0.html)

## 贡献指南

欢迎提交Issue和Pull Request。在贡献代码时，请：

1. 遵循现有的注释规范
2. 添加完整的协议章节索引
3. 说明版本差异
4. 包含字段的详细说明

## 许可证

本项目采用MIT许可证，详见LICENSE文件。
