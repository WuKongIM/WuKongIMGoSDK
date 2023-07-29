package wksdk

import wkproto "github.com/WuKongIM/WuKongIMGoProto"

// Connect connects to the server.
func (c *Client) Connect() error {

	return c.connect()
}

// Disconnect disconnects from the server.
func (c *Client) Disconnect() error {

	return c.disconnect(true)
}

func (c *Client) OnConnect(listener listenerConnFnc) {
	c.listenerConnFnc = listener
}

func (c *Client) OnMessage(listener listenerMsgFnc) {
	c.listenerMsgFnc = listener
}

// SendMessage sends a message to the server.
func (c *Client) SendMessage(payload []byte, channel wkproto.Channel, opt ...SendOption) (*wkproto.SendackPacket, error) {

	return c.send(payload, channel, opt...)
}
