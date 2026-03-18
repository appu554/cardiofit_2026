package services

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// ExpressionEvaluator evaluates YAML condition and formula strings used by PM and MD engines.
// It is a safe, restricted evaluator: only arithmetic, comparison, logical operators, and a small
// whitelist of built-in functions (normalize, abs) are permitted. All other function calls and
// Go keywords are rejected.
type ExpressionEvaluator struct{}

// NewExpressionEvaluator creates a new ExpressionEvaluator.
func NewExpressionEvaluator() *ExpressionEvaluator {
	return &ExpressionEvaluator{}
}

// EvaluateNumeric parses and evaluates expr as an arithmetic expression, returning float64.
// fields maps variable names (including dotted names like "twin_state.IS") to their float64 values.
func (e *ExpressionEvaluator) EvaluateNumeric(expr string, fields map[string]float64) (float64, error) {
	p, err := newParser(expr, fields)
	if err != nil {
		return 0, err
	}
	val, err := p.parseOr()
	if err != nil {
		return 0, err
	}
	if p.pos < len(p.tokens) {
		return 0, fmt.Errorf("unexpected token %q at position %d", p.tokens[p.pos].val, p.pos)
	}
	if val.kind == tokenKindBool {
		return 0, fmt.Errorf("expression returned a boolean, expected numeric")
	}
	return val.num, nil
}

// EvaluateBool parses and evaluates expr as a boolean expression, returning bool.
// fields maps variable names to float64 values.
func (e *ExpressionEvaluator) EvaluateBool(expr string, fields map[string]float64) (bool, error) {
	p, err := newParser(expr, fields)
	if err != nil {
		return false, err
	}
	val, err := p.parseOr()
	if err != nil {
		return false, err
	}
	if p.pos < len(p.tokens) {
		return false, fmt.Errorf("unexpected token %q at position %d", p.tokens[p.pos].val, p.pos)
	}
	if val.kind == tokenKindBool {
		return val.bval, nil
	}
	// If numeric, non-zero is truthy (supports boolean-encoded 1.0/0.0 from DataResolver)
	return val.num != 0, nil
}

// ── Token types ───────────────────────────────────────────────────────────────

type tokenKind int

const (
	tokenKindNum    tokenKind = iota // numeric literal
	tokenKindIdent                   // identifier / field name
	tokenKindOp                      // operator (+, -, *, /, (, ), comma, comparison, logical)
	tokenKindBool                    // boolean result (internal use only, not produced by lexer)
)

type token struct {
	kind tokenKind
	val  string // raw string value
	num  float64
}

// ── Lexer ─────────────────────────────────────────────────────────────────────

// disallowedKeywords lists Go keywords that must be rejected to prevent code injection.
var disallowedKeywords = map[string]bool{
	"import": true,
	"func":   true,
	"go":     true,
	"return": true,
	"var":    true,
	"type":   true,
	"struct": true,
	"map":    true,
	"range":  true,
	"for":    true,
	"if":     true,
	"else":   true,
	"switch": true,
	"case":   true,
	"chan":   true,
	"defer":  true,
	"select": true,
	"break":  true,
	"goto":   true,
	"package": true,
}

// whitelistedFunctions lists the only function names that may appear in expressions.
var whitelistedFunctions = map[string]bool{
	"normalize": true,
	"abs":       true,
}

func tokenize(expr string) ([]token, error) {
	var tokens []token
	i := 0
	runes := []rune(expr)
	n := len(runes)

	for i < n {
		ch := runes[i]

		// Skip whitespace
		if unicode.IsSpace(ch) {
			i++
			continue
		}

		// Numbers: digit or leading dot
		if unicode.IsDigit(ch) || (ch == '.' && i+1 < n && unicode.IsDigit(runes[i+1])) {
			j := i
			for j < n && (unicode.IsDigit(runes[j]) || runes[j] == '.') {
				j++
			}
			raw := string(runes[i:j])
			num, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q", raw)
			}
			tokens = append(tokens, token{kind: tokenKindNum, val: raw, num: num})
			i = j
			continue
		}

		// Identifiers: letters, digits, underscores, dots (for dotted names like twin_state.IS)
		// But only if the dot is surrounded by identifier characters.
		if unicode.IsLetter(ch) || ch == '_' {
			j := i
			for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_' || runes[j] == '.') {
				// Allow dot only if it is followed by a letter/underscore (field separator, not decimal)
				if runes[j] == '.' {
					if j+1 < n && (unicode.IsLetter(runes[j+1]) || runes[j+1] == '_') {
						j++
						continue
					}
					break
				}
				j++
			}
			raw := string(runes[i:j])

			// Check for disallowed Go keywords first
			if disallowedKeywords[raw] {
				return nil, fmt.Errorf("disallowed keyword in expression: %q", raw)
			}

			// AND / OR / NOT are logical operators, treat as op tokens
			upper := strings.ToUpper(raw)
			if upper == "AND" || upper == "OR" || upper == "NOT" {
				tokens = append(tokens, token{kind: tokenKindOp, val: upper})
				i = j
				continue
			}

			tokens = append(tokens, token{kind: tokenKindIdent, val: raw})
			i = j
			continue
		}

		// Two-character operators: >=, <=, ==, !=
		if i+1 < n {
			two := string(runes[i : i+2])
			switch two {
			case ">=", "<=", "==", "!=":
				tokens = append(tokens, token{kind: tokenKindOp, val: two})
				i += 2
				continue
			}
		}

		// Single-character operators
		switch ch {
		case '+', '-', '*', '/', '(', ')', ',', '<', '>':
			tokens = append(tokens, token{kind: tokenKindOp, val: string(ch)})
			i++
			continue
		}

		return nil, fmt.Errorf("unexpected character %q in expression", string(ch))
	}

	return tokens, nil
}

// ── Parser value ──────────────────────────────────────────────────────────────

// value holds the result of a parsed sub-expression.
type value struct {
	kind tokenKind // tokenKindNum or tokenKindBool
	num  float64
	bval bool
}

func numVal(n float64) value  { return value{kind: tokenKindNum, num: n} }
func boolVal(b bool) value    { return value{kind: tokenKindBool, bval: b} }

// asFloat converts any value to float64 (true=1.0, false=0.0).
func (v value) asFloat() float64 {
	if v.kind == tokenKindBool {
		if v.bval {
			return 1.0
		}
		return 0.0
	}
	return v.num
}

// ── Parser ────────────────────────────────────────────────────────────────────

type parser struct {
	tokens []token
	pos    int
	fields map[string]float64
}

func newParser(expr string, fields map[string]float64) (*parser, error) {
	tokens, err := tokenize(expr)
	if err != nil {
		return nil, err
	}
	return &parser{tokens: tokens, pos: 0, fields: fields}, nil
}

func (p *parser) peek() *token {
	if p.pos < len(p.tokens) {
		return &p.tokens[p.pos]
	}
	return nil
}

func (p *parser) consume() *token {
	if p.pos < len(p.tokens) {
		t := &p.tokens[p.pos]
		p.pos++
		return t
	}
	return nil
}

func (p *parser) expect(op string) error {
	t := p.consume()
	if t == nil || t.val != op {
		got := "<EOF>"
		if t != nil {
			got = t.val
		}
		return fmt.Errorf("expected %q, got %q", op, got)
	}
	return nil
}

// Grammar (highest to lowest precedence):
//
//   or-expr    → and-expr  ('OR'  and-expr)*
//   and-expr   → not-expr  ('AND' not-expr)*
//   not-expr   → 'NOT' not-expr | comparison
//   comparison → add-sub  (('<'|'<='|'>'|'>='|'=='|'!=') add-sub)?
//   add-sub    → mul-div  (('+'|'-') mul-div)*
//   mul-div    → unary    (('*'|'/') unary)*
//   unary      → '-' unary | primary
//   primary    → number | ident | ident'(' args ')' | '(' or-expr ')'

func (p *parser) parseOr() (value, error) {
	left, err := p.parseAnd()
	if err != nil {
		return value{}, err
	}
	for {
		t := p.peek()
		if t == nil || t.val != "OR" {
			break
		}
		p.consume()
		right, err := p.parseAnd()
		if err != nil {
			return value{}, err
		}
		result := left.asFloat() != 0 || right.asFloat() != 0
		left = boolVal(result)
	}
	return left, nil
}

func (p *parser) parseAnd() (value, error) {
	left, err := p.parseNot()
	if err != nil {
		return value{}, err
	}
	for {
		t := p.peek()
		if t == nil || t.val != "AND" {
			break
		}
		p.consume()
		right, err := p.parseNot()
		if err != nil {
			return value{}, err
		}
		result := left.asFloat() != 0 && right.asFloat() != 0
		left = boolVal(result)
	}
	return left, nil
}

func (p *parser) parseNot() (value, error) {
	t := p.peek()
	if t != nil && t.val == "NOT" {
		p.consume()
		val, err := p.parseNot()
		if err != nil {
			return value{}, err
		}
		return boolVal(val.asFloat() == 0), nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (value, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return value{}, err
	}
	t := p.peek()
	if t == nil || t.kind != tokenKindOp {
		return left, nil
	}
	switch t.val {
	case "<", "<=", ">", ">=", "==", "!=":
		p.consume()
		right, err := p.parseAddSub()
		if err != nil {
			return value{}, err
		}
		l, r := left.asFloat(), right.asFloat()
		var result bool
		switch t.val {
		case "<":
			result = l < r
		case "<=":
			result = l <= r
		case ">":
			result = l > r
		case ">=":
			result = l >= r
		case "==":
			result = l == r
		case "!=":
			result = l != r
		}
		return boolVal(result), nil
	}
	return left, nil
}

func (p *parser) parseAddSub() (value, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return value{}, err
	}
	for {
		t := p.peek()
		if t == nil || (t.val != "+" && t.val != "-") {
			break
		}
		op := t.val
		p.consume()
		right, err := p.parseMulDiv()
		if err != nil {
			return value{}, err
		}
		if op == "+" {
			left = numVal(left.asFloat() + right.asFloat())
		} else {
			left = numVal(left.asFloat() - right.asFloat())
		}
	}
	return left, nil
}

func (p *parser) parseMulDiv() (value, error) {
	left, err := p.parseUnary()
	if err != nil {
		return value{}, err
	}
	for {
		t := p.peek()
		if t == nil || (t.val != "*" && t.val != "/") {
			break
		}
		op := t.val
		p.consume()
		right, err := p.parseUnary()
		if err != nil {
			return value{}, err
		}
		if op == "*" {
			left = numVal(left.asFloat() * right.asFloat())
		} else {
			if right.asFloat() == 0 {
				return value{}, fmt.Errorf("division by zero")
			}
			left = numVal(left.asFloat() / right.asFloat())
		}
	}
	return left, nil
}

func (p *parser) parseUnary() (value, error) {
	t := p.peek()
	if t != nil && t.val == "-" {
		p.consume()
		val, err := p.parseUnary()
		if err != nil {
			return value{}, err
		}
		return numVal(-val.asFloat()), nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (value, error) {
	t := p.peek()
	if t == nil {
		return value{}, fmt.Errorf("unexpected end of expression")
	}

	// Parenthesised sub-expression
	if t.val == "(" {
		p.consume()
		val, err := p.parseOr()
		if err != nil {
			return value{}, err
		}
		if err := p.expect(")"); err != nil {
			return value{}, err
		}
		return val, nil
	}

	// Numeric literal
	if t.kind == tokenKindNum {
		p.consume()
		return numVal(t.num), nil
	}

	// Identifier: could be a field name or a function call
	if t.kind == tokenKindIdent {
		p.consume()
		name := t.val

		// Check if next token is '(' → function call
		next := p.peek()
		if next != nil && next.val == "(" {
			// Function call — check whitelist
			// Handle dotted names like "os.Exit" — the dot is baked into the ident token
			// by the lexer, but only if the part after the dot is a letter/underscore.
			// We need to check the base name for whitelisting.
			baseName := name
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				// dotted name — never whitelisted (e.g., os.Exit)
				baseName = name[idx+1:]
				_ = baseName
				return value{}, fmt.Errorf("unknown function: %s (only normalize and abs are allowed)", name)
			}
			if !whitelistedFunctions[baseName] {
				return value{}, fmt.Errorf("unknown function: %s (only normalize and abs are allowed)", name)
			}

			// Parse argument list
			p.consume() // consume '('
			args, err := p.parseArgList()
			if err != nil {
				return value{}, err
			}
			if err := p.expect(")"); err != nil {
				return value{}, err
			}

			return p.callBuiltin(name, args)
		}

		// Field name lookup
		// Check for dotted names that look like function calls (e.g. "os.Exit" without parens
		// already handled above by disallowedKeywords or the dot check). Just look up in fields.
		val, ok := p.fields[name]
		if !ok {
			return value{}, fmt.Errorf("undefined field: %q", name)
		}
		return numVal(val), nil
	}

	return value{}, fmt.Errorf("unexpected token %q", t.val)
}

// parseArgList parses a comma-separated list of expressions (without surrounding parens).
func (p *parser) parseArgList() ([]value, error) {
	var args []value

	// Empty argument list
	t := p.peek()
	if t != nil && t.val == ")" {
		return args, nil
	}

	for {
		val, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		args = append(args, val)

		t := p.peek()
		if t == nil || t.val != "," {
			break
		}
		p.consume() // consume ','
	}
	return args, nil
}

// callBuiltin dispatches to whitelisted built-in functions.
func (p *parser) callBuiltin(name string, args []value) (value, error) {
	switch name {
	case "normalize":
		if len(args) != 3 {
			return value{}, fmt.Errorf("normalize requires 3 arguments, got %d", len(args))
		}
		v, minV, maxV := args[0].asFloat(), args[1].asFloat(), args[2].asFloat()
		if maxV == minV {
			return value{}, fmt.Errorf("normalize: min and max must be different (both are %v)", minV)
		}
		result := (v - minV) / (maxV - minV)
		// Clamp to [0, 1]
		result = math.Max(0.0, math.Min(1.0, result))
		return numVal(result), nil

	case "abs":
		if len(args) != 1 {
			return value{}, fmt.Errorf("abs requires 1 argument, got %d", len(args))
		}
		return numVal(math.Abs(args[0].asFloat())), nil

	default:
		return value{}, fmt.Errorf("unknown function: %s", name)
	}
}
