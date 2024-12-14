package packet

import "fmt"

//const (
//	SUCCESS = 0x00 // 0x00 连接已接受
//	//1 0x01 连接已拒绝，不支持的协议版本 服务端不支持客户端请求的 MQTT 协议级别
//	//2 0x02 连接已拒绝，不合格的客户端标识符 客户端标识符是正确的 UTF-8 编码，但服务 端不允许使用
//	//3 0x03 连接已拒绝，服务端不可用 网络连接已建立，但 MQTT 服务不可用
//	//4 0x04 连接已拒绝，无效的用户名或密码 用户名或密码的数据格式无效 MQTT-3.1.1-CN 30
//	// 5 0x05 连接已拒绝，未授权 客户端未被授权连接到此服务器
//)

type ReasonCode struct {
	Code   uint8
	Reason string
	zh     string
}

func (rc ReasonCode) Error() string {
	return fmt.Sprintf("%d:%s", rc.Code, rc.Reason)
}

var (
	// MQTTv3.1.1 specific bytes.
	Err3UnsupportedProtocolVersion = ReasonCode{Code: 0x01, Reason: "unsupported protocol version"}
	Err3ClientIdentifierNotValid   = ReasonCode{Code: 0x02, Reason: "client identifier not valid"}
	Err3ServerUnavailable          = ReasonCode{Code: 0x03, Reason: "server unavailable"}
	ErrMalformedUsernameOrPassword = ReasonCode{Code: 0x04, Reason: "malformed username or password"}
	Err3NotAuthorized              = ReasonCode{Code: 0x05, Reason: "not authorized"}

	// MQTT5
	CodeSuccessIgnore                         = ReasonCode{Code: 0x00, Reason: "ignore packet"}
	CodeSuccess                               = ReasonCode{Code: 0x00, Reason: "success", zh: "成功"}             // CONNACK, PUBACK, PUBREC, PUBREL, PUBCOMP, UNSUBACK, AUTH
	CodeDisconnect                            = ReasonCode{Code: 0x00, Reason: "disconnected", zh: "正常断开"}      // DISCONNECT
	CodeGrantedQos0                           = ReasonCode{Code: 0x00, Reason: "granted qos 0", zh: "授权的QoS 0"} // SUBACK
	CodeGrantedQos1                           = ReasonCode{Code: 0x01, Reason: "granted qos 1"}                 // SUBACK
	CodeGrantedQos2                           = ReasonCode{Code: 0x02, Reason: "granted qos 2"}                 // SUBACK
	CodeDisconnectWillMessage                 = ReasonCode{Code: 0x04, Reason: "disconnect with will message"}  // DISCONNECT
	CodeNoMatchingSubscribers                 = ReasonCode{Code: 0x10, Reason: "no matching subscribers"}
	CodeNoSubscriptionExisted                 = ReasonCode{Code: 0x11, Reason: "no subscription existed"}
	CodeContinueAuthentication                = ReasonCode{Code: 0x18, Reason: "continue authentication"}
	CodeReAuthenticate                        = ReasonCode{Code: 0x19, Reason: "re-authenticate"}
	ErrUnspecifiedError                       = ReasonCode{Code: 0x80, Reason: "unspecified error"}
	ErrMalformedPacket                        = ReasonCode{Code: 0x81, Reason: "malformed packet"}
	ErrMalformedProtocolName                  = ReasonCode{Code: 0x81, Reason: "malformed packet: protocol name"}
	ErrMalformedProtocolVersion               = ReasonCode{Code: 0x81, Reason: "malformed packet: protocol version"}
	ErrMalformedFlags                         = ReasonCode{Code: 0x81, Reason: "malformed packet: flags"}
	ErrMalformedKeepalive                     = ReasonCode{Code: 0x81, Reason: "malformed packet: keepalive"}
	ErrMalformedPacketID                      = ReasonCode{Code: 0x81, Reason: "malformed packet: packet identifier"}
	ErrMalformedTopic                         = ReasonCode{Code: 0x81, Reason: "malformed packet: topic"}
	ErrMalformedWillTopic                     = ReasonCode{Code: 0x81, Reason: "malformed packet: will topic"}
	ErrMalformedWillPayload                   = ReasonCode{Code: 0x81, Reason: "malformed packet: will message"}
	ErrMalformedUsername                      = ReasonCode{Code: 0x81, Reason: "malformed packet: username"}
	ErrMalformedPassword                      = ReasonCode{Code: 0x81, Reason: "malformed packet: password"}
	ErrMalformedQos                           = ReasonCode{Code: 0x81, Reason: "malformed packet: qos"}
	ErrMalformedOffsetUintOutOfRange          = ReasonCode{Code: 0x81, Reason: "malformed packet: offset uint out of range"}
	ErrMalformedOffsetBytesOutOfRange         = ReasonCode{Code: 0x81, Reason: "malformed packet: offset bytes out of range"}
	ErrMalformedOffsetByteOutOfRange          = ReasonCode{Code: 0x81, Reason: "malformed packet: offset byte out of range"}
	ErrMalformedOffsetBoolOutOfRange          = ReasonCode{Code: 0x81, Reason: "malformed packet: offset boolean out of range"}
	ErrMalformedInvalidUTF8                   = ReasonCode{Code: 0x81, Reason: "malformed packet: invalid utf-8 string"}
	ErrMalformedVariableByteInteger           = ReasonCode{Code: 0x81, Reason: "malformed packet: variable byte integer out of range"}
	ErrMalformedBadProperty                   = ReasonCode{Code: 0x81, Reason: "malformed packet: unknown property"}
	ErrMalformedProperties                    = ReasonCode{Code: 0x81, Reason: "malformed packet: properties"}
	ErrMalformedWillProperties                = ReasonCode{Code: 0x81, Reason: "malformed packet: will properties"}
	ErrMalformedSessionPresent                = ReasonCode{Code: 0x81, Reason: "malformed packet: session present"}
	ErrMalformedReasonCode                    = ReasonCode{Code: 0x81, Reason: "malformed packet: reason code"}
	ErrProtocolErr                            = ReasonCode{Code: 0x82, Reason: "protocol error"}
	ErrProtocolViolation                      = ReasonCode{Code: 0x82, Reason: "protocol violation"}
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
	ErrImplementationSpecificError            = ReasonCode{Code: 0x83, Reason: "implementation specific error"}
	ErrRejectPacket                           = ReasonCode{Code: 0x83, Reason: "packet rejected"}
	ErrUnsupportedProtocolVersion             = ReasonCode{Code: 0x84, Reason: "unsupported protocol version"}
	ErrClientIdentifierNotValid               = ReasonCode{Code: 0x85, Reason: "client identifier not valid"}
	ErrClientIdentifierTooLong                = ReasonCode{Code: 0x85, Reason: "client identifier too long"}
	ErrBadUsernameOrPassword                  = ReasonCode{Code: 0x86, Reason: "bad username or password"}
	ErrNotAuthorized                          = ReasonCode{Code: 0x87, Reason: "not authorized"}
	ErrServerUnavailable                      = ReasonCode{Code: 0x88, Reason: "server unavailable"}
	ErrServerBusy                             = ReasonCode{Code: 0x89, Reason: "server busy"}
	ErrBanned                                 = ReasonCode{Code: 0x8A, Reason: "banned"}
	ErrServerShuttingDown                     = ReasonCode{Code: 0x8B, Reason: "server shutting down"}
	ErrBadAuthenticationMethod                = ReasonCode{Code: 0x8C, Reason: "bad authentication method"}
	ErrKeepAliveTimeout                       = ReasonCode{Code: 0x8D, Reason: "keep alive timeout"}
	ErrSessionTakenOver                       = ReasonCode{Code: 0x8E, Reason: "session takeover"}
	ErrTopicFilterInvalid                     = ReasonCode{Code: 0x8F, Reason: "topic filter invalid"}
	ErrTopicNameInvalid                       = ReasonCode{Code: 0x90, Reason: "topic name invalid"}
	ErrPacketIdentifierInUse                  = ReasonCode{Code: 0x91, Reason: "packet identifier in use"}
	ErrPacketIdentifierNotFound               = ReasonCode{Code: 0x92, Reason: "packet identifier not found"}
	ErrReceiveMaximum                         = ReasonCode{Code: 0x93, Reason: "receive maximum exceeded"}
	ErrTopicAliasInvalid                      = ReasonCode{Code: 0x94, Reason: "topic alias invalid"}
	ErrPacketTooLarge                         = ReasonCode{Code: 0x95, Reason: "packet too large"}
	ErrMessageRateTooHigh                     = ReasonCode{Code: 0x96, Reason: "message rate too high"}
	ErrQuotaExceeded                          = ReasonCode{Code: 0x97, Reason: "quota exceeded"}
	ErrPendingClientWritesExceeded            = ReasonCode{Code: 0x97, Reason: "too many pending writes"}
	ErrAdministrativeAction                   = ReasonCode{Code: 0x98, Reason: "administrative action"}
	ErrPayloadFormatInvalid                   = ReasonCode{Code: 0x99, Reason: "payload format invalid"}
	ErrRetainNotSupported                     = ReasonCode{Code: 0x9A, Reason: "retain not supported"}
	ErrQosNotSupported                        = ReasonCode{Code: 0x9B, Reason: "qos not supported"}
	ErrUseAnotherServer                       = ReasonCode{Code: 0x9C, Reason: "use another server"}
	ErrServerMoved                            = ReasonCode{Code: 0x9D, Reason: "server moved"}
	ErrSharedSubscriptionsNotSupported        = ReasonCode{Code: 0x9E, Reason: "shared subscriptions not supported"}
	ErrConnectionRateExceeded                 = ReasonCode{Code: 0x9F, Reason: "connection rate exceeded"}               // CONNACK, DISCONNECT
	ErrMaxConnectTime                         = ReasonCode{Code: 0xA0, Reason: "maximum connect time"}                   // DISCONNECT
	ErrSubscriptionIdentifiersNotSupported    = ReasonCode{Code: 0xA1, Reason: "subscription identifiers not supported"} // SUBACK, DISCONNECT
	ErrWildcardSubscriptionsNotSupported      = ReasonCode{Code: 0xA2, Reason: "wildcard subscriptions not supported"}   // SUBACK, DISCONNECT
)

// Created by me.
var (
	ErrProtocolError = ReasonCode{Code: 0x82, Reason: "protocol error"}
)
