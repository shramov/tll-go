package tll

// #cgo pkg-config: tll
/*
#include <tll/config.h>
*/
import "C"
import "errors"

type ConstConfig struct{ ptr *C.tll_config_t }
type Config struct{ ConstConfig }

func NewConfig() *Config {
	ptr := C.tll_config_new()
	if ptr == nil {
		return nil
	}
	return &Config{ConstConfig{ptr}}
}

func LoadConfig(url string) *Config {
	ptr := C.tll_config_load(C._GoStringPtr(url), C.int(len(url)))
	if ptr == nil {
		return nil
	}
	return &Config{ConstConfig{ptr}}
}

func LoadConfigData(proto string, body string) *Config {
	ptr := C.tll_config_load_data(C._GoStringPtr(proto), C.int(len(proto)), C._GoStringPtr(body), C.int(len(body)))
	if ptr == nil {
		return nil
	}
	return &Config{ConstConfig{ptr}}
}

func ConfigFromMap(settings map[string]string) *Config {
	c := NewConfig()
	for k, v := range(settings) {
		c.Set(k, v)
	}
	return c
}

func (self ConstConfig) Ref() ConstConfig {
	return ConstConfig{C.tll_config_ref(self.ptr)}
}

func (self Config) Ref() Config {
	return Config{self.ConstConfig.Ref()}
}

func (self *ConstConfig) Free() {
	C.tll_config_unref(self.ptr)
	self.ptr = nil
}

func (self *ConstConfig) Copy() *Config {
	ptr := C.tll_config_copy(self.ptr)
	if ptr == nil {
		return nil
	}
	return &Config{ConstConfig{ptr}}
}

func (self ConstConfig) Sub(key string) *ConstConfig {
	ptr := C.tll_config_sub(self.ptr, C._GoStringPtr(key), C.int(len(key)), 0)
	if ptr == nil {
		return nil
	}
	return &ConstConfig{ptr}
}

func (self Config) Sub(key string) *Config {
	r := self.ConstConfig.Sub(key)
	if r == nil {
		return nil
	}
	return &Config{*r}
}

func (self Config) SubCreate(key string) *Config {
	ptr := C.tll_config_sub(self.ptr, C._GoStringPtr(key), C.int(len(key)), 1)
	if ptr == nil {
		return nil
	}
	return &Config{ConstConfig{ptr}}
}

func (self ConstConfig) Value() *string {
	var size C.int
	r := C.tll_config_get_copy(self.ptr, nil, 0, &size)
	if r == nil {
		return nil
	}
	defer C.tll_config_value_free(r)
	s := C.GoStringN(r, size)
	return &s
}

func (self ConstConfig) Get(key string) *string {
	var size C.int
	r := C.tll_config_get_copy(self.ptr, C._GoStringPtr(key), C.int(len(key)), &size)
	if r == nil {
		return nil
	}
	defer C.tll_config_value_free(r)
	s := C.GoStringN(r, size)
	return &s
}

func (self ConstConfig) GetUrl(key string) (*Config, error) {
	r := C.tll_config_get_url(self.ptr, C._GoStringPtr(key), C.int(len(key)))
	if r == nil {
		return nil, errors.New("Invalid Config pointer")
	}
	rc := Config{ConstConfig{r}}
	if v := rc.Value(); v != nil {
		rc.Free()
		return nil, errors.New(*v)
	}
	return &rc, nil
}

func (self Config) Set(key string, value string) {
	C.tll_config_set(self.ptr, C._GoStringPtr(key), C.int(len(key)), C._GoStringPtr(value), C.int(len(value)))
}
