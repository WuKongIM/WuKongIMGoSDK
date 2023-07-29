package main

import (
	"sync"

	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/WuKongIM/WuKongIMGoSDK/pkg/wksdk"
)

func main() {
	var wg sync.WaitGroup
	// ================== client 1 ==================
	cli1 := wksdk.NewClient("tcp://localhost:5100", wksdk.WithUID("test1"))

	err := cli1.Connect()
	if err != nil {
		panic(err)
	}

	// ================== client 2 ==================
	cli2 := wksdk.NewClient("tcp://localhost:5100", wksdk.WithUID("test2"))

	err = cli2.Connect()
	if err != nil {
		panic(err)
	}
	cli2.OnMessage(func(msg *wksdk.Message) {
		println("client2 receive msg:", string(msg.Payload))
		if string(msg.Payload) == "hello" {
			wg.Done()
		}
	})

	// ================== send msg to client2 ==================
	wg.Add(1)
	_, err = cli1.SendMessage([]byte("hello"), wkproto.Channel{
		ChannelType: wkproto.ChannelTypePerson,
		ChannelID:   "test2",
	})
	if err != nil {
		panic(err)
	}

	wg.Wait()
}
