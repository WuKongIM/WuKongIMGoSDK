# 悟空IM GO SDK

引入包

```go
go get github.com/WuKongIM/WuKongIMGoSDK
```

使用
```go
import  "github.com/WuKongIM/WuKongIMGoSDK/pkg/wksdk"
```

初始化

```go
cli := wksdk.NewClient("tcp://127.0.0.1:5100",wksdk.WithUID("xxxx"),wksdk.WithToken("xxxxx"),wksdk.WithReconnect(true))
```

连接

```go

cli.Connect()

// 监听连接状态
cli.OnConnect(func(status Status,reasonCode int) {
    //TODO
})

```

发送消息

```go

result,_ := cli.SendMessage(content,channel)

```

接受消息

```go

cli.OnMessage(func(m *Message) {
    //TODO
})


```