package tll

import "strings"
import "errors"

type ChannelPrefixImpl interface {
	OnState(State) int
	OnData(Message) int
	OnOther(Message) int
}

type Prefix struct {
	Base
	child Channel
}

func (self *Prefix) InitPrefix(impl ChannelPrefixImpl, url ConstConfig, ctx Context) error {
	curl := url.Copy()
	defer curl.Unref()
	proto := curl.Get("tll.proto")
	if proto == nil {
		return errors.New("No proto in url")
	}
	if idx := strings.IndexByte(*proto, '+'); idx >= 0 {
		curl.Set("tll.proto", (*proto)[idx+1:])
	} else {
		return errors.New("No + separator in protocol")
	}
	self.ChildUrlFill(*curl, "go-prefix")
	child := ctx.ChannelCfg(curl.ConstConfig)
	if child == nil {
		return errors.New("Failed to create child channel")
	}

	if err := self.InitBase(url, ctx); err != nil {
		return err
	}
	self.child = *child
	self.child.CallbackAdd(func(c Channel, m Message) int { return prefixCallback(impl, m) }, 0xff)
	self.ChildAdd(self.child, "child")
	return nil
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

func (self *Prefix) OnData(m Message) int {
	self.CallbackData(m)
	return 0
}

func (self *Prefix) OnOther(m Message) int {
	self.Callback(m)
	return 0
}

func prefixCallback(self ChannelPrefixImpl, m Message) int {
	switch m.Type() {
	case MessageData:
		return self.OnData(m)
	case MessageState:
		return self.OnState(State(m.MsgId()))
	default:
		return self.OnOther(m)
	}
	return 0
}

func (self *Prefix) Open(cfg ConstConfig) int {
	return self.child.OpenCfg(&cfg)
}

func (self *Prefix) Close(force bool) int {
	return self.child.CloseForce(force)
}

func (self *Prefix) Post(m Message) error {
	if r := self.child.Post(m); r != 0 {
		return errors.New("Child post failed")
	}
	return nil
}
