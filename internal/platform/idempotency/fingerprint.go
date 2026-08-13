package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidJSON        = errors.New("invalid JSON")
	ErrDuplicateObjectKey = errors.New("duplicate object key")
	ErrInvalidUTF8        = errors.New("invalid UTF-8")
	ErrUnsupportedNumber  = errors.New("unsupported number")
	ErrInvalidRoot        = errors.New("invalid root value")
	ErrMaxDepth           = errors.New("maximum nesting depth exceeded")
)

type Fingerprint string

const maxDepth = 256
const fingerprintPrefix = "tally-request-fingerprint:v1"

type node struct {
	kind     byte
	s        string
	b        bool
	children []member
	array    []*node
}
type member struct {
	key   string
	value *node
}

func Canonicalize(input []byte) ([]byte, error) {
	if !utf8.Valid(input) {
		return nil, ErrInvalidUTF8
	}
	p := parser{data: input}
	p.skipSpace()
	n, err := p.value(0)
	if err != nil {
		return nil, err
	}
	if n.kind != '{' {
		return nil, ErrInvalidRoot
	}
	p.skipSpace()
	if p.pos != len(p.data) {
		return nil, fmt.Errorf("%w: trailing data", ErrInvalidJSON)
	}
	var out strings.Builder
	out.Grow(len(input))
	writeNode(&out, n)
	return []byte(out.String()), nil
}

func ComputeFingerprint(input []byte) (Fingerprint, error) {
	canonical, err := Canonicalize(input)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(fingerprintPrefix))
	h.Write([]byte{0})
	h.Write(canonical)
	return Fingerprint("sha256:" + hex.EncodeToString(h.Sum(nil))), nil
}

type parser struct {
	data []byte
	pos  int
}

func (p *parser) skipSpace() {
	for p.pos < len(p.data) && strings.ContainsRune(" \t\r\n", rune(p.data[p.pos])) {
		p.pos++
	}
}
func (p *parser) value(depth int) (*node, error) {
	p.skipSpace()
	if p.pos >= len(p.data) {
		return nil, ErrInvalidJSON
	}
	switch p.data[p.pos] {
	case '{', '[':
		if depth >= maxDepth {
			return nil, ErrMaxDepth
		}
		if p.data[p.pos] == '{' {
			return p.object(depth + 1)
		}
		return p.arrayValue(depth + 1)
	case '"':
		s, err := p.string()
		if err != nil {
			return nil, err
		}
		return &node{kind: 's', s: s}, nil
	case 't':
		return p.literal("true", &node{kind: 'b', b: true})
	case 'f':
		return p.literal("false", &node{kind: 'b'})
	case 'n':
		return p.literal("null", &node{kind: 'n'})
	default:
		if p.data[p.pos] == '-' || (p.data[p.pos] >= '0' && p.data[p.pos] <= '9') {
			return p.number()
		}
	}
	return nil, ErrInvalidJSON
}
func (p *parser) literal(want string, n *node) (*node, error) {
	if p.pos+len(want) > len(p.data) || string(p.data[p.pos:p.pos+len(want)]) != want {
		return nil, ErrInvalidJSON
	}
	p.pos += len(want)
	return n, nil
}
func (p *parser) object(depth int) (*node, error) {
	p.pos++
	n := &node{kind: '{'}
	p.skipSpace()
	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++
		return n, nil
	}
	seen := map[string]bool{}
	for {
		p.skipSpace()
		if p.pos >= len(p.data) || p.data[p.pos] != '"' {
			return nil, ErrInvalidJSON
		}
		key, err := p.string()
		if err != nil {
			return nil, err
		}
		if seen[key] {
			return nil, ErrDuplicateObjectKey
		}
		seen[key] = true
		p.skipSpace()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return nil, ErrInvalidJSON
		}
		p.pos++
		v, err := p.value(depth)
		if err != nil {
			return nil, err
		}
		n.children = append(n.children, member{key, v})
		p.skipSpace()
		if p.pos >= len(p.data) {
			return nil, ErrInvalidJSON
		}
		if p.data[p.pos] == '}' {
			p.pos++
			return n, nil
		}
		if p.data[p.pos] != ',' {
			return nil, ErrInvalidJSON
		}
		p.pos++
	}
}
func (p *parser) arrayValue(depth int) (*node, error) {
	p.pos++
	n := &node{kind: '['}
	p.skipSpace()
	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++
		return n, nil
	}
	for {
		v, e := p.value(depth)
		if e != nil {
			return nil, e
		}
		n.array = append(n.array, v)
		p.skipSpace()
		if p.pos >= len(p.data) {
			return nil, ErrInvalidJSON
		}
		if p.data[p.pos] == ']' {
			p.pos++
			return n, nil
		}
		if p.data[p.pos] != ',' {
			return nil, ErrInvalidJSON
		}
		p.pos++
	}
}
func (p *parser) number() (*node, error) {
	start := p.pos
	negative := p.data[p.pos] == '-'
	if negative {
		p.pos++
		if p.pos >= len(p.data) {
			return nil, ErrUnsupportedNumber
		}
	}
	if p.data[p.pos] == '0' {
		if negative {
			return nil, ErrUnsupportedNumber
		}
		p.pos++
		if p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
			return nil, ErrUnsupportedNumber
		}
	} else if p.data[p.pos] >= '1' && p.data[p.pos] <= '9' {
		for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
			p.pos++
		}
	} else {
		return nil, ErrUnsupportedNumber
	}
	if p.pos < len(p.data) && (p.data[p.pos] == '.' || p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		return nil, ErrUnsupportedNumber
	}
	return &node{kind: 'i', s: string(p.data[start:p.pos])}, nil
}
func (p *parser) string() (string, error) {
	p.pos++
	var b strings.Builder
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c == '"' {
			p.pos++
			return b.String(), nil
		}
		if c < 0x20 {
			return "", ErrInvalidJSON
		}
		if c == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				return "", ErrInvalidJSON
			}
			esc := p.data[p.pos]
			p.pos++
			switch esc {
			case '"', '\\', '/':
				b.WriteByte(esc)
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'u':
				r, err := p.unicodeEscape()
				if err != nil {
					return "", err
				}
				b.WriteRune(r)
			default:
				return "", ErrInvalidJSON
			}
		} else {
			r, size := utf8.DecodeRune(p.data[p.pos:])
			if r == utf8.RuneError && size == 1 {
				return "", ErrInvalidUTF8
			}
			b.WriteRune(r)
			p.pos += size
		}
	}
	return "", ErrInvalidJSON
}
func (p *parser) unicodeEscape() (rune, error) {
	if p.pos+4 > len(p.data) {
		return 0, ErrInvalidJSON
	}
	v, e := strconv.ParseUint(string(p.data[p.pos:p.pos+4]), 16, 16)
	if e != nil {
		return 0, ErrInvalidJSON
	}
	p.pos += 4
	r := rune(v)
	if r >= 0xD800 && r <= 0xDBFF {
		if p.pos+6 > len(p.data) || p.data[p.pos] != '\\' || p.data[p.pos+1] != 'u' {
			return 0, ErrInvalidJSON
		}
		p.pos += 2
		v2, e := strconv.ParseUint(string(p.data[p.pos:p.pos+4]), 16, 16)
		if e != nil || v2 < 0xDC00 || v2 > 0xDFFF {
			return 0, ErrInvalidJSON
		}
		p.pos += 4
		r = 0x10000 + (rune(v)-0xD800)*0x400 + (rune(v2) - 0xDC00)
	} else if r >= 0xDC00 {
		return 0, ErrInvalidJSON
	}
	return r, nil
}
func writeNode(b *strings.Builder, n *node) {
	switch n.kind {
	case '{':
		b.WriteByte('{')
		sortMembers(n.children)
		for i, m := range n.children {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, m.key)
			b.WriteByte(':')
			writeNode(b, m.value)
		}
		b.WriteByte('}')
	case '[':
		b.WriteByte('[')
		for i, v := range n.array {
			if i > 0 {
				b.WriteByte(',')
			}
			writeNode(b, v)
		}
		b.WriteByte(']')
	case 's':
		writeString(b, n.s)
	case 'b':
		if n.b {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case 'n':
		b.WriteString("null")
	case 'i':
		b.WriteString(n.s)
	}
}
func sortMembers(ms []member) {
	sort.SliceStable(ms, func(i, j int) bool { return string([]byte(ms[i].key)) < string([]byte(ms[j].key)) })
}
func writeString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\b':
			b.WriteString("\\b")
		case '\f':
			b.WriteString("\\f")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			if r < 0x20 {
				fmt.Fprintf(b, "\\u%04x", r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}
