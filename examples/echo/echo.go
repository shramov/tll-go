package main

import "github.com/shramov/tll-go/tll"

type Null struct{ tll.Base }

func (self *Null) Protocol() string { return "go-null" }

func (self *Null) Init(*tll.ConstConfig, *tll.Context) tll.ChannelImpl { return &Null{} }

type Echo struct{ tll.Base }

func (self *Echo) Protocol() string { return "go-echo" }

func (self *Echo) Init(*tll.ConstConfig, *tll.Context) tll.ChannelImpl { return &Echo{} }

func (self *Echo) Post(m *tll.Message) int {
	self.Callback(m)
	return 0
}
