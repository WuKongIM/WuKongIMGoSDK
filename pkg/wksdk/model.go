package wksdk

import wkproto "github.com/WuKongIM/WuKongIMGoProto"

// SendOptions SendOptions
type SendOptions struct {
	NoPersist   bool // 是否不存储 默认 false
	SyncOnce    bool // 是否同步一次（写模式） 默认 false
	RedDot      bool // 是否显示红点 默认true
	NoEncrypt   bool // 是否不需要加密
	ClientMsgNo string
}

// NewSendOptions NewSendOptions
func NewSendOptions() *SendOptions {
	return &SendOptions{
		NoPersist: false,
		SyncOnce:  false,
		RedDot:    true,
	}
}

// SendOption 参数项
type SendOption func(*SendOptions) error

// SendOptionWithNoPersist 是否不存储
func SendOptionWithNoPersist(noPersist bool) SendOption {
	return func(opts *SendOptions) error {
		opts.NoPersist = noPersist
		return nil
	}
}

// SendOptionWithSyncOnce 是否只同步一次（写模式）
func SendOptionWithSyncOnce(syncOnce bool) SendOption {
	return func(opts *SendOptions) error {
		opts.SyncOnce = syncOnce
		return nil
	}
}

// SendOptionWithRedDot 是否显示红点
func SendOptionWithRedDot(redDot bool) SendOption {
	return func(opts *SendOptions) error {
		opts.RedDot = redDot
		return nil
	}
}

// SendOptionWithClientMsgNo 是否显示红点
func SendOptionWithClientMsgNo(clientMsgNo string) SendOption {
	return func(opts *SendOptions) error {
		opts.ClientMsgNo = clientMsgNo
		return nil
	}
}

// SendOptionWithNoEncrypt 是否不需要加密
func SendOptionWithNoEncrypt(noEncrypt bool) SendOption {
	return func(opts *SendOptions) error {
		opts.NoEncrypt = noEncrypt
		return nil
	}
}

type Message struct {
	wkproto.RecvPacket
	Ack func() error
}
