package tll

import "strings"
import "errors"

type ChannelPrefixImpl interface {
	OnState(State) error
	OnData(Message) error
	OnOther(Message) error
}

type Prefix struct {
	Base
	child Channel
}

func (self *Prefix) InitPrefix(impl ChannelPrefixImpl, url ConstConfig, ctx Context) error {
	curl := url.Copy()
	defer curl.Free()
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
	child := ctx.ChannelCfg(curl)
	if child == nil {
		return errors.New("Failed to create child channel")
	}

	if err := self.InitBase(url, ctx); err != nil {
		child.CloseForce(true)
		return err
	}
	self.child = *child
	self.child.CallbackAdd(func(c Channel, m Message) int { return prefixCallback(impl, m) }, MessageMaskAll)
	self.ChildAdd(self.child, "child")
	return nil
}

func (self *Prefix) OnState(s State) error {
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
	return nil
}

func (self *Prefix) OnData(m Message) error {
	self.CallbackData(m)
	return nil
}

func (self *Prefix) OnOther(m Message) error {
	self.Callback(m)
	return nil
}

func prefixCallback(self ChannelPrefixImpl, m Message) int {
	switch m.Type() {
	case MessageData:
		self.OnData(m)
	case MessageState:
		self.OnState(State(m.MsgId()))
	default:
		self.OnOther(m)
	}
	return 0
}

func (self *Prefix) Open(cfg ConstConfig) error {
	return self.child.OpenCfg(&cfg)
}

func (self *Prefix) Close(force bool) error {
	return self.child.CloseForce(force)
}

func (self *Prefix) Post(m Message) error {
	if err := self.child.Post(m); err != nil {
		return errors.New("Child post failed")
	}
	return nil
}
