package wksdk

import (
	"time"

	wkproto "github.com/WuKongIM/WuKongIMGoProto"
)

type Options struct {
	UID            string
	Token          string
	ConnectTimeout time.Duration // 连接超时
	proto          wkproto.Protocol
	ProtoVersion   int
	PingInterval   time.Duration // 心跳间隔
	Reconnect      bool          // 是否自动重连
	AutoAck        bool          // 是否自动ack
}

// NewOptions creates a new options.
func NewOptions() *Options {
	return &Options{
		ConnectTimeout: time.Second * 5,
		ProtoVersion:   wkproto.LatestVersion,
		proto:          wkproto.New(),
		PingInterval:   time.Second * 30,
		Reconnect:      true,
		AutoAck:        true,
	}
}

type Option func(opt *Options)

func WithUID(uid string) Option {
	return func(opt *Options) {
		opt.UID = uid
	}
}

func WithToken(token string) Option {
	return func(opt *Options) {
		opt.Token = token
	}
}

func WithConnectTimeout(timeout time.Duration) Option {
	return func(opt *Options) {
		opt.ConnectTimeout = timeout
	}
}

func WithProtoVersion(version int) Option {
	return func(opt *Options) {
		opt.ProtoVersion = version
	}
}

func WithPingInterval(interval time.Duration) Option {
	return func(opt *Options) {
		opt.PingInterval = interval
	}
}

func WithReconnect(reconnect bool) Option {
	return func(opt *Options) {
		opt.Reconnect = reconnect
	}
}

func WithAutoAck(autoAck bool) Option {
	return func(opt *Options) {
		opt.AutoAck = autoAck
	}
}
