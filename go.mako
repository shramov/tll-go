<%def name='setup_options(parser)'><%
    parser.add_argument('--package', dest='package', type=str, default=None,
                        help='Go package name for generated source')
%></%def>\
package ${options.package or 'scheme'}

import "github.com/shramov/tll-go/tll/binder"
import "time"

const SchemeString string = "${scheme.dump('yamls+gz')}"

<%
NUMERIC = {
    S.Type.Int8: 'int8',
    S.Type.Int16: 'int16',
    S.Type.Int32: 'int32',
    S.Type.Int64: 'int64',
    S.Type.UInt8: 'uint8',
    S.Type.UInt16: 'uint16',
    S.Type.UInt32: 'uint32',
    S.Type.UInt64: 'uint64',
    S.Type.Double: 'float64',
}

from tll.chrono import Resolution

def numeric(t):
    return NUMERIC.get(t, None)

def getter(m, f, offset=None):
    offset = f.offset if offset is None else offset
    n = numeric(f.type)
    if n is not None:
	r = f'self.{n.capitalize()}({offset})'
        if f.sub_type in (f.Sub.Duration, f.Sub.TimePoint):
	    res = f.time_resolution.value
	    name = 'Duration' if f.sub_type == f.Sub.Duration else 'Time'
            return f'binder.{name}Cast(int64({r}), {res[0]}, {res[1]})'
	return r
    elif f.type == f.Type.Bytes and f.sub_type == f.Sub.ByteString:
        return f'self.ByteString({offset}, {f.size})'
    elif f.type == f.Type.Array:
        return f'{m.name}_{f.name}{{self.View({offset})}}'
    elif f.type == f.Type.Pointer:
        if f.sub_type == f.Sub.ByteString:
            return f'self.String({offset})'
        return f'{m.name}_{f.name}{{self.View({offset})}}'
    elif f.type == f.Type.Message:
        return f'{f.type_msg.name}{{self.View({offset})}}'
    return 'nil'

KEYWORDS = {'type': 'type_'}
def keyword(n):
    return KEYWORDS.get(n, n)

def field2type(m, f):
    t = numeric(f.type)
    if t is not None:
        if f.sub_type == f.Sub.Duration:
            return 'time.Duration'
        elif f.sub_type == f.Sub.TimePoint:
            return 'time.Time'
        return t
    elif f.type == f.Decimal128:
        return 'scheme.Decimal128'
    elif f.type == f.Bytes:
        if f.sub_type == f.Sub.ByteString:
            return 'string'
        return f"[u8; {f.size}]"
    elif f.type == f.Message:
        return f.type_msg.name
    elif f.type == f.Array:
        return f"{m.name}_{f.name}"
    elif f.type == f.Pointer:
        if f.sub_type == f.Sub.ByteString:
            return f'string'
        return f"{m.name}_{f.name}"
    raise ValueError(f"Unknown type for field {f.name}: {f.type}")
%>\
<%def name='enum2code(name, e)'>\
type ${name} ${numeric(e.type)}
const (
% for n,v in sorted(e.items(), key=lambda t: (t[1], t[0])):
        ${keyword(n)} ${name} = ${v}
% endfor
)
</%def>\
<%def name='field2decl(m, f)'>
% if f.type == f.Array:
${field2decl(m, f.type_array)}type ${m.name}_${f.name} struct{binder.Binder}
func (self ${m.name}_${f.name}) Size() uint { return uint(${getter(m, f.count_ptr)}) }
func (self ${m.name}_${f.name}) Capacity() uint { return ${f.count} }
func (self ${m.name}_${f.name}) ElementSize() uint { return ${f.type_array.size} }
func (self ${m.name}_${f.name}) Get(idx uint) ${field2type(m, f.type_array)} { return ${getter(m, f.type_array, f'{f.type_array.offset} + idx * self.ElementSize()')} }
% elif f.type == f.Pointer and f.sub_type != f.Sub.ByteString:
${field2decl(m, f.type_ptr)}type ${m.name}_${f.name} struct{binder.Binder}
func (self ${m.name}_${f.name}) Size() uint { return self.PointerDefault(0).Size() }
func (self ${m.name}_${f.name}) ElementSize() uint { return self.PointerDefault(0).Entity(${f.size}) }
func (self ${m.name}_${f.name}) Get(idx uint) ${field2type(m, f.type_ptr)} {
        ptr := self.PointerDefault(0)
        offset := ptr.Offset()
        return ${getter(m, f.type_ptr, f'offset + idx * self.ElementSize()')}
}
% elif f.type == f.Bytes:
% endif
</%def>\
<%def name='field2code(msg, f)'>\
${field2decl(msg, f)}func (self ${msg.name}) Get${f.name.capitalize()}() ${field2type(msg, f)} { return ${getter(msg, f)} }
</%def>
% for e in scheme.enums.values():
<%call expr='enum2code(e.name, e)'></%call>
% endfor
% for msg in scheme.messages:
type ${msg.name} struct {binder.Binder}
func Bind${msg.name}(ptr []byte) ${msg.name} { return ${msg.name}{binder.NewBinder(ptr)} }
% if msg.msgid != 0:
func (${msg.name}) MessageId() int { return ${msg.msgid} }
% endif

% for f in msg.fields:
${field2code(msg, f)}\
% endfor
% endfor
