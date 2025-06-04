package main

import "github.com/shramov/tll-go/tll"

type Null struct{ tll.Base }

func (self *Null) Protocol() string { return "go-null" }

func (self *Null) Init(tll.ConstConfig, tll.Context) (tll.ChannelImpl, error) { return &Null{}, nil }

type Echo struct{ tll.Base }

func (self *Echo) Protocol() string { return "go-echo" }

func (self *Echo) Init(tll.ConstConfig, tll.Context) (tll.ChannelImpl, error) { return &Echo{}, nil }

func (self *Echo) Post(m tll.Message) error {
	self.Callback(m)
	return nil
}

type Prefix struct {
	tll.Prefix
}

func (self *Prefix) Protocol() string { return "go-prefix+" }

func (self *Prefix) Init(url tll.ConstConfig, ctx tll.Context) (tll.ChannelImpl, error) {
	r := Prefix{}
	if err := r.InitPrefix(&r, url, ctx); err != nil {
		return nil, err
	}
	return &r, nil
}
