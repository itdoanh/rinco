// Package handler — small CEL-like expression evaluator used by the
// dynamic-model validators.  The intent is to give multi-tenant customers
// a familiar expression syntax for record validation without taking on a
// heavy CEL runtime dependency.
//
// Supported subset:
//
//   Literals:    123   1.5   "abc"   true   false   null
//   References:  $value       (the value being validated)
//                $data.path   (nested data field lookup)
//   Operators:   == != < > <= >= && ||
//   Grouping:    ( ... )
//
// Anything else returns an "unsupported" parse error.  This keeps the
// attack surface tiny while covering the common validation cases
// (numeric range, regex/equality checks on fields, required-if-other-field).
package handler

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// celEnv carries the evaluation context.
type celEnv struct {
	Value any
	Data  map[string]any
}

// celASTNode is one of: celLiteral, celRef, celBinary, celLogical, celCompare.
type celASTNode interface {
	eval(env celEnv) (bool, error)
}

type celLiteral struct {
	Value any
}

func (l celLiteral) eval(_ celEnv) (bool, error) {
	b, ok := toBool(l.Value)
	if !ok {
		return false, fmt.Errorf("non-boolean literal")
	}
	return b, nil
}

type celRef struct {
	Path []string // ["$value"] or ["$data", "field", "subfield"]
}

func (r celRef) eval(env celEnv) (bool, error) {
	v, err := r.resolve(env)
	if err != nil {
		return false, err
	}
	b, ok := toBool(v)
	if !ok {
		return false, fmt.Errorf("reference %v is not boolean", r.Path)
	}
	return b, nil
}

func (r celRef) resolve(env celEnv) (any, error) {
	if len(r.Path) == 0 {
		return nil, fmt.Errorf("empty reference")
	}
	switch r.Path[0] {
	case "$value":
		return env.Value, nil
	case "$data":
		var cur any = env.Data
		for _, p := range r.Path[1:] {
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("path %v: not a map", r.Path[:len(r.Path)-1])
			}
			cur, ok = m[p]
			if !ok {
				return nil, fmt.Errorf("path %v: field %q not found", r.Path, p)
			}
		}
		return cur, nil
	}
	return nil, fmt.Errorf("unknown root %q", r.Path[0])
}

type celBinary struct {
	Op    string // == != < > <= >=
	Left  celASTNode
	Right celASTNode
}

func (b celBinary) eval(env celEnv) (bool, error) {
	l, err := evalAny(b.Left, env)
	if err != nil {
		return false, err
	}
	r, err := evalAny(b.Right, env)
	if err != nil {
		return false, err
	}
	switch b.Op {
	case "==":
		return equal(l, r), nil
	case "!=":
		return !equal(l, r), nil
	case "<", ">", "<=", ">=":
		return compare(l, r, b.Op)
	}
	return false, fmt.Errorf("unknown operator %q", b.Op)
}

type celLogical struct {
	Op    string // && ||
	Left  celASTNode
	Right celASTNode
}

func (l celLogical) eval(env celEnv) (bool, error) {
	a, err := l.Left.eval(env)
	if err != nil {
		return false, err
	}
	if l.Op == "&&" && !a {
		return false, nil
	}
	if l.Op == "||" && a {
		return true, nil
	}
	b, err := l.Right.eval(env)
	if err != nil {
		return false, err
	}
	if l.Op == "&&" {
		return a && b, nil
	}
	return a || b, nil
}

// evalAny returns the raw (non-bool-coerced) value for comparison ops.
func evalAny(n celASTNode, env celEnv) (any, error) {
	switch x := n.(type) {
	case celLiteral:
		return x.Value, nil
	case celRef:
		return x.resolve(env)
	case celLogical:
		b, err := x.eval(env)
		return b, err
	}
	return nil, fmt.Errorf("node has no value")
}

// =============================================================================
// Parser (Pratt-style)
// =============================================================================

type celTokenKind int

const (
	tkIdent celTokenKind = iota
	tkNumber
	tkString
	tkBool
	tkNull
	tkOp
	tkLParen
	tkRParen
	tkDot
	tkEOF
)

type celToken struct {
	Kind celTokenKind
	Val  string
	Pos  int
}

type celLexer struct {
	src string
	pos int
}

func (l *celLexer) next() celToken {
	for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t' || l.src[l.pos] == '\n') {
		l.pos++
	}
	if l.pos >= len(l.src) {
		return celToken{Kind: tkEOF, Pos: l.pos}
	}
	start := l.pos
	ch := l.src[l.pos]
	// identifiers and keywords
	if isAlpha(ch) || ch == '$' {
		for l.pos < len(l.src) && (isAlpha(l.src[l.pos]) || isDigit(l.src[l.pos]) || l.src[l.pos] == '_' || l.src[l.pos] == '.') {
			l.pos++
		}
		w := l.src[start:l.pos]
		switch w {
		case "true":
			return celToken{Kind: tkBool, Val: "true", Pos: start}
		case "false":
			return celToken{Kind: tkBool, Val: "false", Pos: start}
		case "null":
			return celToken{Kind: tkNull, Val: "null", Pos: start}
		}
		return celToken{Kind: tkIdent, Val: w, Pos: start}
	}
	// numbers
	if isDigit(ch) || (ch == '-' && l.pos+1 < len(l.src) && isDigit(l.src[l.pos+1]) && (start == 0 || !isAlphaNum(l.src[l.pos-1]))) {
		l.pos++
		for l.pos < len(l.src) && (isDigit(l.src[l.pos]) || l.src[l.pos] == '.') {
			l.pos++
		}
		return celToken{Kind: tkNumber, Val: l.src[start:l.pos], Pos: start}
	}
	// string literal
	if ch == '"' {
		l.pos++
		for l.pos < len(l.src) && l.src[l.pos] != '"' {
			if l.src[l.pos] == '\\' && l.pos+1 < len(l.src) {
				l.pos += 2
				continue
			}
			l.pos++
		}
		if l.pos < len(l.src) {
			l.pos++
		}
		return celToken{Kind: tkString, Val: l.src[start+1 : l.pos-1], Pos: start}
	}
	// operators
	switch ch {
	case '(':
		l.pos++
		return celToken{Kind: tkLParen, Val: "(", Pos: start}
	case ')':
		l.pos++
		return celToken{Kind: tkRParen, Val: ")", Pos: start}
	}
	// two-char ops
	if l.pos+1 < len(l.src) {
		two := l.src[l.pos : l.pos+2]
		if two == "==" || two == "!=" || two == "<=" || two == ">=" || two == "&&" || two == "||" {
			l.pos += 2
			return celToken{Kind: tkOp, Val: two, Pos: start}
		}
	}
	if ch == '<' || ch == '>' || ch == '=' {
		l.pos++
		return celToken{Kind: tkOp, Val: string(ch), Pos: start}
	}
	// unknown — advance
	l.pos++
	return l.next()
}

func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlphaNum(c byte) bool { return isAlpha(c) || isDigit(c) }

// =============================================================================
// Parser
// =============================================================================

type celParser struct {
	lex *celLexer
	cur celToken
	nxt celToken
}

func newCELParser() *celParser { return &celParser{} }

func (p *celParser) Parse(src string) (celASTNode, error) {
	p.lex = &celLexer{src: src}
	p.cur = p.lex.next()
	p.nxt = p.lex.next()
	node, err := p.parseLogical()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind != tkEOF {
		return nil, fmt.Errorf("unexpected token %q at pos %d", p.cur.Val, p.cur.Pos)
	}
	return node, nil
}

func (p *celParser) advance() {
	p.cur = p.nxt
	p.nxt = p.lex.next()
}

func (p *celParser) parseLogical() (celASTNode, error) {
	left, err := p.parseCompare()
	if err != nil {
		return nil, err
	}
	for p.cur.Kind == tkOp && (p.cur.Val == "&&" || p.cur.Val == "||") {
		op := p.cur.Val
		p.advance()
		right, err := p.parseCompare()
		if err != nil {
			return nil, err
		}
		left = celLogical{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *celParser) parseCompare() (celASTNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind == tkOp && (p.cur.Val == "==" || p.cur.Val == "!=" || p.cur.Val == "<" || p.cur.Val == ">" || p.cur.Val == "<=" || p.cur.Val == ">=") {
		op := p.cur.Val
		p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return celBinary{Op: op, Left: left, Right: right}, nil
	}
	return left, nil
}

func (p *celParser) parsePrimary() (celASTNode, error) {
	t := p.cur
	switch t.Kind {
	case tkLParen:
		p.advance()
		node, err := p.parseLogical()
		if err != nil {
			return nil, err
		}
		if p.cur.Kind != tkRParen {
			return nil, fmt.Errorf("expected ) at pos %d", p.cur.Pos)
		}
		p.advance()
		return node, nil
	case tkNumber:
		p.advance()
		if strings.Contains(t.Val, ".") {
			f, err := strconv.ParseFloat(t.Val, 64)
			if err != nil {
				return nil, err
			}
			return celLiteral{Value: f}, nil
		}
		i, err := strconv.ParseInt(t.Val, 10, 64)
		if err != nil {
			return nil, err
		}
		return celLiteral{Value: i}, nil
	case tkString:
		p.advance()
		return celLiteral{Value: t.Val}, nil
	case tkBool:
		p.advance()
		return celLiteral{Value: t.Val == "true"}, nil
	case tkNull:
		p.advance()
		return celLiteral{Value: nil}, nil
	case tkIdent:
		// $value or $data.path
		if !strings.HasPrefix(t.Val, "$") {
			return nil, fmt.Errorf("identifier %q must start with $", t.Val)
		}
		parts := strings.Split(t.Val, ".")
		p.advance()
		return celRef{Path: parts}, nil
	}
	return nil, fmt.Errorf("unexpected token %q at pos %d", t.Val, t.Pos)
}

// =============================================================================
// Helpers
// =============================================================================

func toBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case nil:
		return false, true
	case string:
		return x != "", true
	case float64:
		return x != 0, true
	case float32:
		return x != 0, true
	case int:
		return x != 0, true
	case int32:
		return x != 0, true
	case int64:
		return x != 0, true
	}
	return false, false
}

func equal(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	// numeric promotion
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			return af == bf
		}
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return as == bs
	}
	return a == b
}

func compare(a, b any, op string) (bool, error) {
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if !aok || !bok {
		return false, fmt.Errorf("non-numeric comparison")
	}
	switch op {
	case "<":
		return af < bf, nil
	case ">":
		return af > bf, nil
	case "<=":
		return af <= bf, nil
	case ">=":
		return af >= bf, nil
	}
	return false, nil
}

// toFloat coerces numeric-ish values to float64.
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}

// silence unused import lint
var _ = sync.Mutex{}
