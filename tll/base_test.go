package tll

import "testing"

type Echo struct{ Base }

func (self *Echo) Protocol() string {
	return "echo"
}

func (self *Echo) Init(ConstConfig, Context) (ChannelImpl, error) {
	println("Init")
	return &Echo{}, nil
}

func (self *Echo) Free() {
	println("Free")
}

func (self *Echo) Open(ConstConfig) error {
	println("Open")
	return nil 
}

func (self *Echo) Close(bool) error {
	println("Close")
	return nil
}

func (self *Echo) Process() int {
	println("Process")
	return 0
}

func (self *Echo) Post(m Message) error {
	println("Post")
	return nil
}

func TestEcho(t *testing.T) {
	ctx := NewContext()
	impl := CreateImpl[*Echo]()
	if err := ctx.register(impl); err != nil {
		panic("Fail to register impl")
	}

	c := ctx.Channel("echo://;name=echo;dump=frame")
	if c == nil {
		panic("Fail to init channel")
	}
	println(c.Name())
	h := c.CallbackAdd(func(c Channel, m Message) int {
		println("Tick")
		return 0
	}, MessageMaskAll)
	c.Open()
	c.Process()
	c.Process()
	println("Drop callback")
	h.Free()
	c.Process()
	c.Free()
}
