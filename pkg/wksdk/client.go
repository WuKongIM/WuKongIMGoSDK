package wksdk

import (
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/WuKongIM/WuKongIMGoSDK/pkg/wkutil"
	"go.uber.org/atomic"
)

type listenerConnFnc func(connectStatus ConnectStatus, reasonCode wkproto.ReasonCode)

type Client struct {
	addr string
	opts *Options
	conn net.Conn

	clientPrivKey [32]byte
	aesKey        string // aes密钥
	salt          string // 安全码

	connackChan chan *wkproto.ConnackPacket

	connectStatus ConnectStatus

	listenerConnFnc listenerConnFnc

	stopped         bool // 是否已停止
	forceDisconnect bool // 是否强制关闭

	pongs []chan struct{}

	pingTimer *time.Timer
	exitChan  chan struct{}

	lastActivity time.Time // 最后一次活动时间

	clientSeq atomic.Uint64 // 客户端消息序号

	sendackMapLock sync.RWMutex
	sendackChanMap map[uint64]chan *wkproto.SendackPacket // 发送ack
}

// NewClient creates a new client. addr is the address of the server. example: "tcp://localhost:5100"
func NewClient(addr string, opt ...Option) *Client {
	opts := NewOptions()

	if len(opt) > 0 {
		for _, o := range opt {
			o(opts)
		}
	}

	return &Client{
		addr:           addr,
		opts:           opts,
		exitChan:       make(chan struct{}),
		sendackChanMap: make(map[uint64]chan *wkproto.SendackPacket),
	}
}

func (c *Client) connect() error {
	c.connectStatusChange(CONNECTING)
	err := c.handshake()
	if err != nil {
		c.connectStatusChange(DISCONNECTED)
		return err
	}
	c.connectStatusChange(CONNECTED)
	return nil
}

// 重连
func (c *Client) reconnect() error {
	c.connectStatusChange(RECONNECTING)
	c.disconnect(false)
	return c.connect()
}

func (c *Client) disconnect(forceDisconnect bool) error {
	defer c.connectStatusChange(DISCONNECTED)

	c.sendackMapLock.Lock()
	c.sendackChanMap = make(map[uint64]chan *wkproto.SendackPacket)
	c.sendackMapLock.Unlock()

	c.forceDisconnect = forceDisconnect
	c.stopped = true
	c.stopPing()
	err := c.conn.Close()
	if err != nil {

		return err
	}
	return nil
}

// 握手
func (c *Client) handshake() error {

	c.stopped = false
	// ========= 创建连接 =========
	var err error
	c.conn, err = c.createConn()
	if err != nil {
		return err
	}
	// ========= 循环读客户端数据 =========
	go c.loopRead(c.conn)

	// ========= 发送连接包 =========
	c.connackChan = make(chan *wkproto.ConnackPacket)
	err = c.sendConnect()
	if err != nil {
		return err
	}

	// ========= 处理连接返回 =========
	connack := <-c.connackChan
	if connack.ReasonCode != wkproto.ReasonSuccess {
		return fmt.Errorf("reason is %d", connack.ReasonCode)
	}
	c.salt = connack.Salt
	serverKey, err := base64.StdEncoding.DecodeString(connack.ServerKey)
	if err != nil {
		return err
	}
	var serverPubKey [32]byte
	copy(serverPubKey[:], serverKey[:32])

	shareKey := wkutil.GetCurve25519Key(c.clientPrivKey, serverPubKey) // 共享key
	c.aesKey = wkutil.MD5(base64.StdEncoding.EncodeToString(shareKey[:]))[:16]

	c.pongs = make([]chan struct{}, 0, 8)

	// start ping
	c.startPing()

	return nil
}

func (c *Client) startPing() {
	c.pingTimer = time.AfterFunc(c.opts.PingInterval, c.processPingTimer)

}

func (c *Client) stopPing() {
	c.pingTimer.Stop()
}

func (c *Client) createConn() (net.Conn, error) {
	network, address, _ := parseAddr(c.addr)
	conn, err := net.DialTimeout(network, address, c.opts.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *Client) loopRead(conn net.Conn) {
	var buff = make([]byte, 10240)
	var tmpBuff = make([]byte, 0)
	for !c.stopped {
		n, err := c.conn.Read(buff)
		if err != nil {
			fmt.Println("read error:", err)
			goto exit
		}
		tmpBuff = append(tmpBuff, buff[:n]...)
		frame, size, err := c.opts.proto.DecodeFrame(tmpBuff, wkproto.LatestVersion)
		if err != nil {
			fmt.Println("decode error:", err)
			goto exit
		}
		if size > 0 {
			tmpBuff = tmpBuff[size:]
		}
		if frame != nil {
			c.handleFrame(frame, conn)
		}
	}
exit:
	fmt.Println("loop read exit")
	if c.needReconnect() {
		c.reconnect()
	} else {
		c.connectStatusChange(CLOSED)
	}

}

func (c *Client) processPingTimer() {
	if c.connectStatus != CONNECTED {
		return
	}
	if c.lastActivity.Add(c.opts.PingInterval).Before(time.Now()) {
		if c.forceDisconnect {
			return
		}
		c.reconnect()
		return
	}
	c.sendPing(nil)
	c.pingTimer.Reset(c.opts.PingInterval)
}

func (c *Client) needReconnect() bool {
	if c.opts.Reconnect && !c.forceDisconnect {
		return true
	}
	return false
}

func (c *Client) handleFrame(frame wkproto.Frame, conn net.Conn) {
	c.lastActivity = time.Now()
	switch frame.GetFrameType() {
	case wkproto.CONNACK:
		c.handleConnack(frame, conn)
	case wkproto.PONG:
		c.handlePong(frame, conn)
	case wkproto.SENDACK:
		c.handleSendack(frame, conn)

	}
}

func (c *Client) handleSendack(frame wkproto.Frame, conn net.Conn) {
	sendack := frame.(*wkproto.SendackPacket)
	c.triggerSendack(sendack)
	c.removeSendackChan(sendack.ClientSeq)
}

func (c *Client) handleConnack(frame wkproto.Frame, conn net.Conn) {
	connack := frame.(*wkproto.ConnackPacket)
	c.connackChan <- connack
}

func (c *Client) handlePong(frame wkproto.Frame, conn net.Conn) {
	var ch chan struct{}
	if len(c.pongs) > 0 {
		ch = c.pongs[0]
		c.pongs = append(c.pongs[:0], c.pongs[1:]...)
	}
	if ch != nil {
		ch <- struct{}{}
	}
}

func (c *Client) send(payload []byte, channel wkproto.Channel, opt ...SendOption) (*wkproto.SendackPacket, error) {
	opts := NewSendOptions()
	if len(opt) > 0 {
		for _, op := range opt {
			op(opts)
		}
	}
	// 加密消息内容
	newPayload, err := wkutil.AesEncryptPkcs7Base64(payload, []byte(c.aesKey), []byte(c.salt))
	if err != nil {
		return nil, err
	}

	clientMsgNo := opts.ClientMsgNo
	if clientMsgNo == "" {
		clientMsgNo = wkutil.GenUUID() // TODO: uuid生成非常耗性能
	}

	clientSeq := c.clientSeq.Inc()
	packet := &wkproto.SendPacket{
		Framer: wkproto.Framer{
			NoPersist: opts.NoPersist,
			SyncOnce:  opts.SyncOnce,
			RedDot:    opts.RedDot,
		},
		ClientSeq:   clientSeq,
		ClientMsgNo: clientMsgNo,
		ChannelID:   channel.ChannelID,
		ChannelType: channel.ChannelType,
		Payload:     newPayload,
	}
	signStr := packet.VerityString()
	actMsgKey, err := wkutil.AesEncryptPkcs7Base64([]byte(signStr), []byte(c.aesKey), []byte(c.salt))
	if err != nil {
		return nil, err
	}
	packet.MsgKey = wkutil.MD5(string(actMsgKey))
	return c.sendSendPacket(packet)
}

func (c *Client) sendConnect() error {
	var clientPubKey [32]byte
	c.clientPrivKey, clientPubKey = wkutil.GetCurve25519KeypPair() // 生成服务器的DH密钥对
	packet := &wkproto.ConnectPacket{
		Version:         uint8(c.opts.ProtoVersion),
		DeviceID:        wkutil.GenUUID(),
		DeviceFlag:      wkproto.APP,
		ClientKey:       base64.StdEncoding.EncodeToString(clientPubKey[:]),
		ClientTimestamp: time.Now().Unix(),
		UID:             c.opts.UID,
		Token:           c.opts.Token,
	}
	err := c.sendPacket(packet)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) sendPing(ch chan struct{}) error {

	c.pongs = append(c.pongs, ch)
	data, err := c.opts.proto.EncodeFrame(&wkproto.PingPacket{}, uint8(c.opts.ProtoVersion))
	if err != nil {
		return err
	}
	_, err = c.conn.Write(data)
	return err
}

func (c *Client) connectStatusChange(connectStatus ConnectStatus) {
	c.connectStatus = connectStatus
	if c.listenerConnFnc != nil {
		c.listenerConnFnc(c.connectStatus, wkproto.ReasonSuccess)
	}
}

func (c *Client) sendSendPacket(packet *wkproto.SendPacket) (*wkproto.SendackPacket, error) {

	ch := c.createSendackChan(packet.ClientSeq)

	err := c.sendPacket(packet)
	if err != nil {
		c.sendackMapLock.Lock()
		delete(c.sendackChanMap, packet.ClientSeq)
		c.sendackMapLock.Unlock()
		return nil, err
	}
	sendack := <-ch
	if sendack.ReasonCode != wkproto.ReasonSuccess {
		return sendack, fmt.Errorf("reason is %d", sendack.ReasonCode)
	}
	return sendack, nil
}

func (c *Client) createSendackChan(clientSeq uint64) chan *wkproto.SendackPacket {
	c.sendackMapLock.Lock()
	ch := make(chan *wkproto.SendackPacket)
	c.sendackChanMap[clientSeq] = ch
	c.sendackMapLock.Unlock()

	return ch
}

func (c *Client) triggerSendack(sendack *wkproto.SendackPacket) {
	c.sendackMapLock.RLock()
	ch, ok := c.sendackChanMap[sendack.ClientSeq]
	c.sendackMapLock.RUnlock()
	if ok {
		ch <- sendack
	}
}

func (c *Client) removeSendackChan(clientSeq uint64) {
	c.sendackMapLock.Lock()
	delete(c.sendackChanMap, clientSeq)
	c.sendackMapLock.Unlock()
}

// 发送包
func (c *Client) sendPacket(packet wkproto.Frame) error {
	data, err := c.opts.proto.EncodeFrame(packet, uint8(c.opts.ProtoVersion))
	if err != nil {
		return err
	}
	_, err = c.conn.Write(data)
	return err
}

func parseAddr(addr string) (network, address string, port int) {
	network = "tcp"
	address = strings.ToLower(addr)
	if strings.Contains(address, "://") {
		pair := strings.Split(address, "://")
		network = pair[0]
		address = pair[1]
		pair2 := strings.Split(address, ":")
		portStr := pair2[1]
		portInt64, _ := strconv.ParseInt(portStr, 10, 64)
		port = int(portInt64)
	}
	return
}
