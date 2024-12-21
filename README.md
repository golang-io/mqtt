# mqtt 

### 巨人的肩膀


implement
### https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/mqtt-v3.1.1.html


mqtt.Client -> http.Transport -> http.Conn -> net.Conn

http.Server -> http.ResponseWriter -> http.Conn -> net.Conn


https://mcxiaoke.gitbooks.io/mqtt-cn/content/mqtt/06-WebSocket.html


p2p模式：https://help.aliyun.com/zh/apsaramq-for-mqtt/cloud-message-queue-mqtt-upgraded/developer-reference/features-p2p-messaging-model?spm=a2c4g.11186623.0.0.44ff2ab0JCP9Sa#concept-96176-zh



### 注入数据包(Packet Injection)

如果你想要更多的服务端控制，或者想要设置特定的MQTT v5属性或其他属性，你可以选择指定的客户端创建自己的发布包(publish packets)。这种方法允许你将MQTT数据包(packets)直接注入到运行中的服务端，相当于服务端直接自己模拟接收到了某个客户端的数据包。

数据包注入(Packet Injection)可用于任何MQTT数据包，包括ping请求、订阅等。你可以获取客户端的详细信息，因此你甚至可以直接在服务端模拟某个在线的客户端，发布一个数据包。

大多数情况下，您可能希望使用上面描述的内联客户端(Inline Client)，因为它具有独特的特权：它可以绕过所有ACL和主题验证检查，这意味着它甚至可以发布到$SYS主题。你也可以自己从头开始制定一个自己的内联客户端，它将与内置的内联客户端行为相同。

```go
cl := server.NewClient(nil, "local", "inline", true)
server.InjectPacket(cl, packets.Packet{
  FixedHeader: packets.FixedHeader{
    Type: packets.Publish,
  },
  TopicName: "direct/publish",
  Payload: []byte("scheduled message"),
})
```

> MQTT数据包仍然需要满足规范的结构，所以请参考[测试用例中数据包的定义](packets/tpackets.go) 和 [MQTTv5规范](https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html) 以获取一些帮助。

具体如何使用请参考 [hooks example](examples/hooks/main.go) 。

## 测试(Testing)
#### 单元测试(Unit Tests)


#### Paho 互操作性测试(Paho Interoperability Test)

您可以使用 `examples/paho/main.go` 启动服务器，然后在 _interoperability_ 文件夹中运行 `python3 client_test5.py` 来检查代理是否符合 [Paho互操作性测试](https://github.com/eclipse/paho.mqtt.testing/tree/master/interoperability) 的要求，包括 MQTT v5 和 v3 的测试。

> 请注意，关于 paho 测试套件存在一些尚未解决的问题，因此在 `paho/main.go` 示例中启用了某些兼容性模式。

## 基准测试(Performance Benchmarks)

Mochi MQTT 的性能与其他的一些主流的mqtt中间件（如 Mosquitto、EMQX 等）不相上下。

基准测试是使用 [MQTT-Stresser](https://github.com/inovex/mqtt-stresser) 在 Apple Macbook Air M2 上进行的，使用 `cmd/main.go` 默认设置。考虑到高低吞吐量的突发情况，中位数分数是最有用的。数值越高越好。


> 基准测试中呈现的数值不代表真实每秒消息吞吐量。它们依赖于 mqtt-stresser 的一种不寻常的计算方法，但它们在所有代理之间是一致的。性能基准测试的结果仅供参考。这些比较都是使用默认配置进行的。

`mqtt-stresser -broker tcp://localhost:1883 -num-clients=2 -num-messages=10000`

| Broker            | publish fastest | median | slowest | receive fastest | median | slowest |
| --                | --             | --   | --   | --             | --   | --   |
| Mochi v2.2.10      | 124,772 | 125,456 | 124,614 | 314,461 | 313,186 | 311,910 |
| [Mosquitto v2.0.15](https://github.com/eclipse/mosquitto) | 155,920 | 155,919 | 155,918 | 185,485 | 185,097 | 184,709 |
| [EMQX v5.0.11](https://github.com/emqx/emqx)      | 156,945 | 156,257 | 155,568 | 17,918 | 17,783 | 17,649 |
| [Rumqtt v0.21.0](https://github.com/bytebeamio/rumqtt) | 112,208 | 108,480 | 104,753 | 135,784 | 126,446 | 117,108 |

`mqtt-stresser -broker tcp://localhost:1883 -num-clients=10 -num-messages=10000`

| Broker                       | publish fastest | median | slowest | receive fastest | median | slowest |
|------------------------------|-----------------| --   | --   | --             | --   | --   |
| Mochi v2.2.10                | 41,825          | 31,663| 23,008 | 144,058 | 65,903 | 37,618 |
| Mosquitto v2.0.15            | 42,729          | 38,633 | 29,879 | 23,241 | 19,714 | 18,806 |
| EMQX v5.0.11                 | 21,553          | 17,418 | 14,356 | 4,257 | 3,980 | 3,756 |
| Rumqtt v0.21.0               | 42,213          | 23,153 | 20,814 | 49,465 | 36,626 | 19,283 |
| mqtt-server[v0.0.1]          | 24,374          | 16,040 | 10,856 | 71,731 | 32,081 | 15,785 |
| mqtt-server[v0.0.2] - buffer | 38,435          | 23407 | 30615 | 40351 | 27848 | 22182 |
| mqtt-server-buffer/sync.Map| 14,191          | 2,912  | 1,180   | 89,528          | 5,605  | 1,468   |

百万消息挑战（立即向服务器发送100万条消息）:

`mqtt-stresser -broker tcp://localhost:1883 -num-clients=100 -num-messages=10000`

| Broker            | publish fastest | median | slowest | receive fastest | median | slowest |
| --                |-----------------|--------|---------|-----------------|--------|---------|
| Mochi v2.2.10     | 13,532          | 4,425  | 2,344   | 52,120          | 7,274  | 2,701   |
| Mosquitto v2.0.15 | 3,826           | 3,395  | 3,032   | 1,200           | 1,150  | 1,118   |
| EMQX v5.0.11      | 4,086           | 2,432  | 2,274   | 434             | 333    | 311     |
| Rumqtt v0.21.0    | 78,972          | 5,047  | 3,804   | 4,286           | 3,249  | 2,027   |
| mqtt-server       | 75,974          | 6,059  | 1,685   | 24,711          | 2,724  | 1,819   |
| mqtt-server-buffer/sync.Map| 14,191          | 2,912  | 1,180   | 89,528          | 5,605  | 1,468   |
> 这里还不确定EMQX是不是哪里出了问题，可能是因为 Docker 的默认配置优化不对，所以要持保留意见，因为我们确实知道它是一款可靠的软件。
