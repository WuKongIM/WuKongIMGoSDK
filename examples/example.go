package main

import (
	"time"

	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/WuKongIM/WuKongIMGoSDK/pkg/wksdk"
)

func main() {
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

	// ================== send msg to client2 ==================
	_, err = cli1.SendMessage([]byte(`{"type":1,"content":"hello"}`), wkproto.Channel{
		ChannelType: wkproto.ChannelTypePerson,
		ChannelID:   "test2",
	})
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Millisecond * 100)
}
