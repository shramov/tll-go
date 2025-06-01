package tll

import "strings"

type ChannelPrefixImpl interface {
	OnState(State) int
	OnData(*Message) int
	OnOther(*Message) int
}

type Prefix struct {
	Base
	child Channel
}

func (self *Prefix) InitPrefix(impl ChannelPrefixImpl, url *ConstConfig, ctx *Context) int {
	curl := url.Copy()
	proto := curl.Get("tll.proto")
	if proto == nil {
		println("No proto in url")
		return -1
	}
	if idx := strings.IndexByte(*proto, '+'); idx >= 0 {
		curl.Set("tll.proto", (*proto)[idx+1:])
	} else {
		println("No + separator in url")
		return -1
	}
	curl.Set("name", "go-prefix/"+*url.Get("name"))
	child := ctx.ChannelCfg(&curl.ConstConfig)
	if child == nil {
		return -1
	}

	self.InitInternal()
	self.child = *child
	self.child.CallbackAdd(func(c *Channel, m *Message) int { return prefixCallback(impl, m) }, 0xff)
	self.ChildAdd(child, "child")
	return 0
}

func (self *Prefix) OnState(s State) int {
	switch s {
	case StateActive:
		self.SetState(s)
		break
	case StateClosed:
		self.SetState(s)
		break
	default:
		break
	}
	return 0
}

func (self *Prefix) OnData(m *Message) int {
	self.CallbackData(m)
	return 0
}

func (self *Prefix) OnOther(m *Message) int {
	self.Callback(m)
	return 0
}

func prefixCallback(self ChannelPrefixImpl, m *Message) int {
	switch m.GetType() {
	case MessageData:
		return self.OnData(m)
	case MessageState:
		return self.OnState(State(m.GetMsgId()))
	default:
		return self.OnOther(m)
	}
	return 0
}

func (self *Prefix) Open(cfg *ConstConfig) int {
	return self.child.OpenCfg(cfg)
}

func (self *Prefix) Close(force bool) int {
	return self.child.CloseForce(force)
}

func (self *Prefix) Post(m *Message) int {
	self.child.Post(m)
	return 0
}
