package packet

import "fmt"

/*
================================================================================
MQTT 错误码和原因码定义
================================================================================

参考文档:
- MQTT v3.1.1: 章节 3.2.2.3 CONNACK Return code
- MQTT v5.0: 章节 3.2.2.3 CONNACK Reason Code, 章节 4.13 Handling errors

================================================================================
协议概述
================================================================================

本文件定义了MQTT协议中使用的所有错误码和原因码。这些码用于：
1. 连接确认时的状态指示
2. 发布/订阅操作的响应
3. 断开连接的原因说明
4. 协议错误的分类和处理

================================================================================
错误码分类
================================================================================

1. 成功码 (0x00)
   - 表示操作成功完成
   - 在不同上下文中可能有不同的含义

2. 正常断开码 (0x00-0x04)
   - 表示正常的断开连接情况
   - 包含遗嘱消息发送等

3. 协议错误码 (0x80-0x82)
   - 0x80: 未指定错误
   - 0x81: 格式错误的包
   - 0x82: 协议错误

4. 实现特定错误码 (0x83)
   - 服务器实现相关的错误

5. 连接拒绝码 (0x84-0x8F)
   - 各种连接被拒绝的原因

6. 运行时错误码 (0x90-0xA2)
   - 连接建立后的运行时错误

================================================================================
版本差异
================================================================================

MQTT v3.1.1:
- 只支持基本的连接返回码 (0x00-0x05)
- 错误处理相对简单

MQTT v5.0:
- 支持完整的原因码系统
- 提供详细的错误分类和处理机制
- 支持属性系统和扩展错误信息

================================================================================
*/

// 注释掉的MQTT v3.1.1连接返回码定义
// 这些码在MQTT v3.1.1的CONNACK包中使用
//const (
//	SUCCESS = 0x00 // 0x00 连接已接受
//	//1 0x01 连接已拒绝，不支持的协议版本 服务端不支持客户端请求的 MQTT 协议级别
//	//2 0x02 连接已拒绝，不合格的客户端标识符 客户端标识符是正确的 UTF-8 编码，但服务 端不允许使用
//	//3 0x03 连接已拒绝，服务端不可用 网络连接已建立，但 MQTT 服务不可用
//	//4 0x04 连接已拒绝，无效的用户名或密码 用户名或密码的数据格式无效 MQTT-3.1.1-CN 30
//	// 5 0x05 连接已拒绝，未授权 客户端未被授权连接到此服务器
//)

// ReasonCode MQTT原因码结构
// 参考: MQTT v5.0 章节 4.13 Handling errors
// 包含错误码、英文原因描述和中文描述
type ReasonCode struct {
	Code   uint8  // 错误码值
	Reason string // 英文原因描述
	zh     string // 中文原因描述
}

// Error 实现error接口，返回格式化的错误信息
func (rc ReasonCode) Error() string {
	return fmt.Sprintf("%d:%s", rc.Code, rc.Reason)
}

var (
	/*
		================================================================================
		MQTT v3.1.1 特定错误码
		================================================================================

		参考: MQTT v3.1.1 章节 3.2.2.3 CONNACK Return code
		这些错误码在MQTT v3.1.1的CONNACK包中使用，表示连接被拒绝的原因。

		================================================================================
	*/

	// Err3UnsupportedProtocolVersion 不支持的协议版本
	// 码值: 0x01
	// 含义: 服务端不支持客户端请求的MQTT协议级别
	// 使用场景: CONNACK包
	Err3UnsupportedProtocolVersion = ReasonCode{Code: 0x01, Reason: "unsupported protocol version"}

	// Err3ClientIdentifierNotValid 客户端标识符无效
	// 码值: 0x02
	// 含义: 客户端标识符是正确的UTF-8编码，但服务端不允许使用
	// 使用场景: CONNACK包
	Err3ClientIdentifierNotValid = ReasonCode{Code: 0x02, Reason: "client identifier not valid"}

	// Err3ServerUnavailable 服务端不可用
	// 码值: 0x03
	// 含义: 网络连接已建立，但MQTT服务不可用
	// 使用场景: CONNACK包
	Err3ServerUnavailable = ReasonCode{Code: 0x03, Reason: "server unavailable"}

	// ErrMalformedUsernameOrPassword 用户名或密码格式错误
	// 码值: 0x04
	// 含义: 用户名或密码的数据格式无效
	// 使用场景: CONNACK包
	// 参考: MQTT-3.1.1-CN 30
	ErrMalformedUsernameOrPassword = ReasonCode{Code: 0x04, Reason: "malformed username or password"}

	// Err3NotAuthorized 未授权
	// 码值: 0x05
	// 含义: 客户端未被授权连接到此服务器
	// 使用场景: CONNACK包
	Err3NotAuthorized = ReasonCode{Code: 0x05, Reason: "not authorized"}

	/*
		================================================================================
		MQTT v5.0 原因码系统
		================================================================================

		参考: MQTT v5.0 章节 4.13 Handling errors
		MQTT v5.0提供了完整的原因码系统，用于详细描述各种操作的结果和错误原因。

		================================================================================
	*/

	// 成功码 (0x00) - 在不同上下文中表示不同的成功状态
	// CodeSuccessIgnore 忽略包 (成功)
	// 码值: 0x00
	// 含义: 包被接收但被忽略
	// 使用场景: 各种响应包
	CodeSuccessIgnore = ReasonCode{Code: 0x00, Reason: "ignore packet"}

	// CodeSuccess 成功
	// 码值: 0x00
	// 含义: 操作成功完成
	// 使用场景: CONNACK, PUBACK, PUBREC, PUBREL, PUBCOMP, UNSUBACK, AUTH
	CodeSuccess = ReasonCode{Code: 0x00, Reason: "success", zh: "成功"}

	// CodeDisconnect 正常断开
	// 码值: 0x00
	// 含义: 正常断开连接
	// 使用场景: DISCONNECT包
	CodeDisconnect = ReasonCode{Code: 0x00, Reason: "disconnected", zh: "正常断开"}

	// QoS授权码 - 用于SUBACK包
	// CodeGrantedQos0 授权QoS 0
	// 码值: 0x00
	// 含义: 订阅被授权，使用QoS 0
	// 使用场景: SUBACK包
	CodeGrantedQos0 = ReasonCode{Code: 0x00, Reason: "granted qos 0", zh: "授权的QoS 0"}

	// CodeGrantedQos1 授权QoS 1
	// 码值: 0x01
	// 含义: 订阅被授权，使用QoS 1
	// 使用场景: SUBACK包
	CodeGrantedQos1 = ReasonCode{Code: 0x01, Reason: "granted qos 1"}

	// CodeGrantedQos2 授权QoS 2
	// 码值: 0x02
	// 含义: 订阅被授权，使用QoS 2
	// 使用场景: SUBACK包
	CodeGrantedQos2 = ReasonCode{Code: 0x02, Reason: "granted qos 2"}

	// 断开连接相关码
	// CodeDisconnectWillMessage 断开连接并发送遗嘱消息
	// 码值: 0x04
	// 含义: 服务器断开连接并发送遗嘱消息
	// 使用场景: DISCONNECT包
	CodeDisconnectWillMessage = ReasonCode{Code: 0x04, Reason: "disconnect with will message"}

	// 发布相关码
	// CodeNoMatchingSubscribers 没有匹配的订阅者
	// 码值: 0x10
	// 含义: 没有订阅者匹配发布消息的主题
	// 使用场景: PUBACK, PUBREC包
	CodeNoMatchingSubscribers = ReasonCode{Code: 0x10, Reason: "no matching subscribers"}

	// CodeNoSubscriptionExisted 订阅不存在
	// 码值: 0x11
	// 含义: 尝试取消不存在的订阅
	// 使用场景: UNSUBACK包
	CodeNoSubscriptionExisted = ReasonCode{Code: 0x11, Reason: "no subscription existed"}

	// 认证相关码
	// CodeContinueAuthentication 继续认证
	// 码值: 0x18
	// 含义: 认证过程需要继续
	// 使用场景: AUTH包
	CodeContinueAuthentication = ReasonCode{Code: 0x18, Reason: "continue authentication"}

	// CodeReAuthenticate 重新认证
	// 码值: 0x19
	// 含义: 客户端需要重新认证
	// 使用场景: AUTH包
	CodeReAuthenticate = ReasonCode{Code: 0x19, Reason: "re-authenticate"}

	/*
		================================================================================
		协议错误码 (0x80-0x82)
		================================================================================

		这些错误码表示协议层面的问题，包括格式错误、协议违规等。

		================================================================================
	*/

	// ErrUnspecifiedError 未指定错误
	// 码值: 0x80
	// 含义: 发生了未指定的错误
	// 使用场景: 各种响应包
	ErrUnspecifiedError = ReasonCode{Code: 0x80, Reason: "unspecified error"}

	// 格式错误码 (0x81) - 表示包格式有问题
	// ErrMalformedPacket 格式错误的包
	// 码值: 0x81
	// 含义: 包格式错误
	// 使用场景: 各种响应包
	ErrMalformedPacket = ReasonCode{Code: 0x81, Reason: "malformed packet"}

	// 各种格式错误的具体类型
	ErrMalformedProtocolName          = ReasonCode{Code: 0x81, Reason: "malformed packet: protocol name"}
	ErrMalformedProtocolVersion       = ReasonCode{Code: 0x81, Reason: "malformed packet: protocol version"}
	ErrMalformedFlags                 = ReasonCode{Code: 0x81, Reason: "malformed packet: flags"}
	ErrMalformedKeepalive             = ReasonCode{Code: 0x81, Reason: "malformed packet: keepalive"}
	ErrMalformedPacketID              = ReasonCode{Code: 0x81, Reason: "malformed packet: packet identifier"}
	ErrMalformedTopic                 = ReasonCode{Code: 0x81, Reason: "malformed packet: topic"}
	ErrMalformedWillTopic             = ReasonCode{Code: 0x81, Reason: "malformed packet: will topic"}
	ErrMalformedWillPayload           = ReasonCode{Code: 0x81, Reason: "malformed packet: will message"}
	ErrMalformedUsername              = ReasonCode{Code: 0x81, Reason: "malformed packet: username"}
	ErrMalformedPassword              = ReasonCode{Code: 0x81, Reason: "malformed packet: password"}
	ErrMalformedQos                   = ReasonCode{Code: 0x81, Reason: "malformed packet: qos"}
	ErrMalformedOffsetUintOutOfRange  = ReasonCode{Code: 0x81, Reason: "malformed packet: offset uint out of range"}
	ErrMalformedOffsetBytesOutOfRange = ReasonCode{Code: 0x81, Reason: "malformed packet: offset bytes out of range"}
	ErrMalformedOffsetByteOutOfRange  = ReasonCode{Code: 0x81, Reason: "malformed packet: offset byte out of range"}
	ErrMalformedOffsetBoolOutOfRange  = ReasonCode{Code: 0x81, Reason: "malformed packet: offset boolean out of range"}
	ErrMalformedInvalidUTF8           = ReasonCode{Code: 0x81, Reason: "malformed packet: invalid utf-8 string"}
	ErrMalformedVariableByteInteger   = ReasonCode{Code: 0x81, Reason: "malformed packet: variable byte integer out of range"}
	ErrMalformedBadProperty           = ReasonCode{Code: 0x81, Reason: "malformed packet: unknown property"}
	ErrMalformedProperties            = ReasonCode{Code: 0x81, Reason: "malformed packet: properties"}
	ErrMalformedWillProperties        = ReasonCode{Code: 0x81, Reason: "malformed packet: will properties"}
	ErrMalformedSessionPresent        = ReasonCode{Code: 0x81, Reason: "malformed packet: session present"}
	ErrMalformedReasonCode            = ReasonCode{Code: 0x81, Reason: "malformed packet: reason code"}

	// 协议错误码 (0x82) - 表示协议违规
	// ErrProtocolErr 协议错误
	// 码值: 0x82
	// 含义: 协议错误
	// 使用场景: 各种响应包
	ErrProtocolErr = ReasonCode{Code: 0x82, Reason: "protocol error"}

	// ErrProtocolViolation 协议违规
	// 码值: 0x82
	// 含义: 违反了协议规则
	// 使用场景: 各种响应包
	ErrProtocolViolation = ReasonCode{Code: 0x82, Reason: "protocol violation"}

	// 各种协议违规的具体类型
	ErrProtocolViolationProtocolName          = ReasonCode{Code: 0x82, Reason: "protocol violation: protocol name"}
	ErrProtocolViolationProtocolVersion       = ReasonCode{Code: 0x82, Reason: "protocol violation: protocol version"}
	ErrProtocolViolationReservedBit           = ReasonCode{Code: 0x82, Reason: "protocol violation: reserved bit not 0"}
	ErrProtocolViolationFlagNoUsername        = ReasonCode{Code: 0x82, Reason: "protocol violation: username flag set but no value"}
	ErrProtocolViolationFlagNoPassword        = ReasonCode{Code: 0x82, Reason: "protocol violation: password flag set but no value"}
	ErrProtocolViolationUsernameNoFlag        = ReasonCode{Code: 0x82, Reason: "protocol violation: username set but no flag"}
	ErrProtocolViolationPasswordNoFlag        = ReasonCode{Code: 0x82, Reason: "protocol violation: username set but no flag"}
	ErrProtocolViolationPasswordTooLong       = ReasonCode{Code: 0x82, Reason: "protocol violation: password too long"}
	ErrProtocolViolationUsernameTooLong       = ReasonCode{Code: 0x82, Reason: "protocol violation: username too long"}
	ErrProtocolViolationNoPacketID            = ReasonCode{Code: 0x82, Reason: "protocol violation: missing packet id"}
	ErrProtocolViolationSurplusPacketID       = ReasonCode{Code: 0x82, Reason: "protocol violation: surplus packet id"}
	ErrProtocolViolationQosOutOfRange         = ReasonCode{Code: 0x82, Reason: "protocol violation: qos out of range"}
	ErrProtocolViolationSecondConnect         = ReasonCode{Code: 0x82, Reason: "protocol violation: second connect packet"}
	ErrProtocolViolationZeroNonZeroExpiry     = ReasonCode{Code: 0x82, Reason: "protocol violation: non-zero expiry"}
	ErrProtocolViolationRequireFirstConnect   = ReasonCode{Code: 0x82, Reason: "protocol violation: first packet must be connect"}
	ErrProtocolViolationWillFlagNoPayload     = ReasonCode{Code: 0x82, Reason: "protocol violation: will flag no payload"}
	ErrProtocolViolationWillFlagSurplusRetain = ReasonCode{Code: 0x82, Reason: "protocol violation: will flag surplus retain"}
	ErrProtocolViolationSurplusWildcard       = ReasonCode{Code: 0x82, Reason: "protocol violation: topic contains wildcards"}
	ErrProtocolViolationSurplusSubID          = ReasonCode{Code: 0x82, Reason: "protocol violation: contained subscription identifier"}
	ErrProtocolViolationInvalidTopic          = ReasonCode{Code: 0x82, Reason: "protocol violation: invalid topic"}
	ErrProtocolViolationInvalidSharedNoLocal  = ReasonCode{Code: 0x82, Reason: "protocol violation: invalid shared no local"}
	ErrProtocolViolationNoFilters             = ReasonCode{Code: 0x82, Reason: "protocol violation: must contain at least one filter"}
	ErrProtocolViolationInvalidReason         = ReasonCode{Code: 0x82, Reason: "protocol violation: invalid reason"}
	ErrProtocolViolationOversizeSubID         = ReasonCode{Code: 0x82, Reason: "protocol violation: oversize subscription id"}
	ErrProtocolViolationDupNoQos              = ReasonCode{Code: 0x82, Reason: "protocol violation: dup true with no qos"}
	ErrProtocolViolationUnsupportedProperty   = ReasonCode{Code: 0x82, Reason: "protocol violation: unsupported property"}
	ErrProtocolViolationNoTopic               = ReasonCode{Code: 0x82, Reason: "protocol violation: no topic or alias"}

	/*
		================================================================================
		实现特定错误码 (0x83)
		================================================================================

		这些错误码表示服务器实现相关的错误。

		================================================================================
	*/

	// ErrImplementationSpecificError 实现特定错误
	// 码值: 0x83
	// 含义: 发生了实现特定的错误
	// 使用场景: 各种响应包
	ErrImplementationSpecificError = ReasonCode{Code: 0x83, Reason: "implementation specific error"}

	// ErrRejectPacket 包被拒绝
	// 码值: 0x83
	// 含义: 包被服务器拒绝
	// 使用场景: 各种响应包
	ErrRejectPacket = ReasonCode{Code: 0x83, Reason: "packet rejected"}

	/*
		================================================================================
		连接拒绝码 (0x84-0x8F)
		================================================================================

		这些错误码表示连接被拒绝的各种原因。

		================================================================================
	*/

	// ErrUnsupportedProtocolVersion 不支持的协议版本
	// 码值: 0x84
	// 含义: 服务器不支持客户端请求的MQTT协议版本
	// 使用场景: CONNACK包
	ErrUnsupportedProtocolVersion = ReasonCode{Code: 0x84, Reason: "unsupported protocol version"}

	// ErrClientIdentifierNotValid 客户端标识符无效
	// 码值: 0x85
	// 含义: 客户端标识符无效
	// 使用场景: CONNACK包
	ErrClientIdentifierNotValid = ReasonCode{Code: 0x85, Reason: "client identifier not valid"}

	// ErrClientIdentifierTooLong 客户端标识符过长
	// 码值: 0x85
	// 含义: 客户端标识符长度超出限制
	// 使用场景: CONNACK包
	ErrClientIdentifierTooLong = ReasonCode{Code: 0x85, Reason: "client identifier too long"}

	// ErrBadUsernameOrPassword 用户名或密码错误
	// 码值: 0x86
	// 含义: 用户名或密码错误
	// 使用场景: CONNACK包
	ErrBadUsernameOrPassword = ReasonCode{Code: 0x86, Reason: "bad username or password"}

	// ErrNotAuthorized 未授权
	// 码值: 0x87
	// 含义: 客户端未被授权连接到此服务器
	// 使用场景: CONNACK包
	ErrNotAuthorized = ReasonCode{Code: 0x87, Reason: "not authorized"}

	// ErrServerUnavailable 服务器不可用
	// 码值: 0x88
	// 含义: 服务器不可用
	// 使用场景: CONNACK包
	ErrServerUnavailable = ReasonCode{Code: 0x88, Reason: "server unavailable"}

	// ErrServerBusy 服务器繁忙
	// 码值: 0x89
	// 含义: 服务器繁忙，无法处理请求
	// 使用场景: CONNACK包
	ErrServerBusy = ReasonCode{Code: 0x89, Reason: "server busy"}

	// ErrBanned 被禁止
	// 码值: 0x8A
	// 含义: 客户端被禁止连接
	// 使用场景: CONNACK包
	ErrBanned = ReasonCode{Code: 0x8A, Reason: "banned"}

	// ErrServerShuttingDown 服务器正在关闭
	// 码值: 0x8B
	// 含义: 服务器正在关闭
	// 使用场景: CONNACK包
	ErrServerShuttingDown = ReasonCode{Code: 0x8B, Reason: "server shutting down"}

	// ErrBadAuthenticationMethod 认证方法错误
	// 码值: 0x8C
	// 含义: 认证方法不被支持
	// 使用场景: CONNACK包
	ErrBadAuthenticationMethod = ReasonCode{Code: 0x8C, Reason: "bad authentication method"}

	// ErrKeepAliveTimeout 保活超时
	// 码值: 0x8D
	// 含义: 保活超时
	// 使用场景: CONNACK包
	ErrKeepAliveTimeout = ReasonCode{Code: 0x8D, Reason: "keep alive timeout"}

	// ErrSessionTakenOver 会话被接管
	// 码值: 0x8E
	// 含义: 会话被另一个连接接管
	// 使用场景: CONNACK包
	ErrSessionTakenOver = ReasonCode{Code: 0x8E, Reason: "session takeover"}

	// ErrTopicFilterInvalid 主题过滤器无效
	// 码值: 0x8F
	// 含义: 主题过滤器无效
	// 使用场景: CONNACK包
	ErrTopicFilterInvalid = ReasonCode{Code: 0x8F, Reason: "topic filter invalid"}

	/*
		================================================================================
		运行时错误码 (0x90-0xA2)
		================================================================================

		这些错误码表示连接建立后的运行时错误。

		================================================================================
	*/

	// ErrTopicNameInvalid 主题名无效
	// 码值: 0x90
	// 含义: 主题名无效
	// 使用场景: PUBACK, PUBREC包
	ErrTopicNameInvalid = ReasonCode{Code: 0x90, Reason: "topic name invalid"}

	// ErrPacketIdentifierInUse 报文标识符正在使用
	// 码值: 0x91
	// 含义: 报文标识符正在被使用
	// 使用场景: PUBACK, PUBREC包
	ErrPacketIdentifierInUse = ReasonCode{Code: 0x91, Reason: "packet identifier in use"}

	// ErrPacketIdentifierNotFound 报文标识符未找到
	// 码值: 0x92
	// 含义: 报文标识符未找到
	// 使用场景: PUBREL, PUBCOMP包
	ErrPacketIdentifierNotFound = ReasonCode{Code: 0x92, Reason: "packet identifier not found"}

	// ErrReceiveMaximum 接收最大值超出
	// 码值: 0x93
	// 含义: 接收最大值超出
	// 使用场景: PUBACK, PUBREC包
	ErrReceiveMaximum = ReasonCode{Code: 0x93, Reason: "receive maximum exceeded"}

	// ErrTopicAliasInvalid 主题别名无效
	// 码值: 0x94
	// 含义: 主题别名无效
	// 使用场景: PUBLISH包
	ErrTopicAliasInvalid = ReasonCode{Code: 0x94, Reason: "topic alias invalid"}

	// ErrPacketTooLarge 包过大
	// 码值: 0x95
	// 含义: 包过大
	// 使用场景: 各种响应包
	ErrPacketTooLarge = ReasonCode{Code: 0x95, Reason: "packet too large"}

	// ErrMessageRateTooHigh 消息速率过高
	// 码值: 0x96
	// 含义: 消息速率过高
	// 使用场景: 各种响应包
	ErrMessageRateTooHigh = ReasonCode{Code: 0x96, Reason: "message rate too high"}

	// ErrQuotaExceeded 配额超出
	// 码值: 0x97
	// 含义: 配额超出
	// 使用场景: 各种响应包
	ErrQuotaExceeded = ReasonCode{Code: 0x97, Reason: "quota exceeded"}

	// ErrPendingClientWritesExceeded 待处理客户端写入超出
	// 码值: 0x97
	// 含义: 待处理的客户端写入操作过多
	// 使用场景: 各种响应包
	ErrPendingClientWritesExceeded = ReasonCode{Code: 0x97, Reason: "too many pending writes"}

	// ErrAdministrativeAction 管理操作
	// 码值: 0x98
	// 含义: 由于管理操作而断开连接
	// 使用场景: DISCONNECT包
	ErrAdministrativeAction = ReasonCode{Code: 0x98, Reason: "administrative action"}

	// ErrPayloadFormatInvalid 载荷格式无效
	// 码值: 0x99
	// 含义: 载荷格式无效
	// 使用场景: PUBACK, PUBREC包
	ErrPayloadFormatInvalid = ReasonCode{Code: 0x99, Reason: "payload format invalid"}

	// ErrRetainNotSupported 保留消息不支持
	// 码值: 0x9A
	// 含义: 服务器不支持保留消息
	// 使用场景: PUBACK, PUBREC包
	ErrRetainNotSupported = ReasonCode{Code: 0x9A, Reason: "retain not supported"}

	// ErrQosNotSupported QoS不支持
	// 码值: 0x9B
	// 含义: 服务器不支持请求的QoS级别
	// 使用场景: PUBACK, PUBREC包
	ErrQosNotSupported = ReasonCode{Code: 0x9B, Reason: "qos not supported"}

	// ErrUseAnotherServer 使用另一个服务器
	// 码值: 0x9C
	// 含义: 客户端应该使用另一个服务器
	// 使用场景: CONNACK, DISCONNECT包
	ErrUseAnotherServer = ReasonCode{Code: 0x9C, Reason: "use another server"}

	// ErrServerMoved 服务器已移动
	// 码值: 0x9D
	// 含义: 服务器已移动到另一个地址
	// 使用场景: CONNACK, DISCONNECT包
	ErrServerMoved = ReasonCode{Code: 0x9D, Reason: "server moved"}

	// ErrSharedSubscriptionsNotSupported 共享订阅不支持
	// 码值: 0x9E
	// 含义: 服务器不支持共享订阅
	// 使用场景: SUBACK包
	ErrSharedSubscriptionsNotSupported = ReasonCode{Code: 0x9E, Reason: "shared subscriptions not supported"}

	// ErrConnectionRateExceeded 连接速率超出
	// 码值: 0x9F
	// 含义: 连接速率超出限制
	// 使用场景: CONNACK, DISCONNECT包
	ErrConnectionRateExceeded = ReasonCode{Code: 0x9F, Reason: "connection rate exceeded"}

	// ErrMaxConnectTime 最大连接时间
	// 码值: 0xA0
	// 含义: 达到最大连接时间
	// 使用场景: DISCONNECT包
	ErrMaxConnectTime = ReasonCode{Code: 0xA0, Reason: "maximum connect time"}

	// ErrSubscriptionIdentifiersNotSupported 订阅标识符不支持
	// 码值: 0xA1
	// 含义: 服务器不支持订阅标识符
	// 使用场景: SUBACK, DISCONNECT包
	ErrSubscriptionIdentifiersNotSupported = ReasonCode{Code: 0xA1, Reason: "subscription identifiers not supported"}

	// ErrWildcardSubscriptionsNotSupported 通配符订阅不支持
	// 码值: 0xA2
	// 含义: 服务器不支持通配符订阅
	// 使用场景: SUBACK, DISCONNECT包
	ErrWildcardSubscriptionsNotSupported = ReasonCode{Code: 0xA2, Reason: "wildcard subscriptions not supported"}
)

/*
================================================================================
自定义错误码
================================================================================

这些错误码是为了保持向后兼容性而添加的别名。
================================================================================
*/

// ErrProtocolError 协议错误 (别名)
// 码值: 0x82
// 含义: 协议错误
// 使用场景: 各种响应包
// 注意: 这是ErrProtocolErr的别名，为了保持向后兼容性
var ErrProtocolError = ReasonCode{Code: 0x82, Reason: "protocol error"}
