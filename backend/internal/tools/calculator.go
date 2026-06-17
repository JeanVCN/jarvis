package tools

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

var ErrInvalidExpression = errors.New("invalid expression")

type CalculatorTool struct{}

func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

func (t *CalculatorTool) Name() string {
	return "calculator"
}

func (t *CalculatorTool) Description() string {
	return "Executa calculos basicos"
}

func (t *CalculatorTool) Execute(ctx context.Context, input string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	parser := newExpressionParser(input)
	value, err := parser.parse()
	if err != nil {
		return "", err
	}

	return formatNumber(value), nil
}

type expressionParser struct {
	input string
	pos   int
}

func newExpressionParser(input string) *expressionParser {
	return &expressionParser{input: strings.TrimSpace(input)}
}

func (p *expressionParser) parse() (float64, error) {
	if p.input == "" {
		return 0, ErrInvalidExpression
	}

	value, err := p.parseExpression()
	if err != nil {
		return 0, err
	}

	p.skipSpaces()
	if p.pos != len(p.input) {
		return 0, fmt.Errorf("%w: unexpected character %q", ErrInvalidExpression, p.input[p.pos])
	}

	return value, nil
}

func (p *expressionParser) parseExpression() (float64, error) {
	value, err := p.parseTerm()
	if err != nil {
		return 0, err
	}

	for {
		p.skipSpaces()
		if p.match('+') {
			next, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			value += next
			continue
		}
		if p.match('-') {
			next, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			value -= next
			continue
		}
		return value, nil
	}
}

func (p *expressionParser) parseTerm() (float64, error) {
	value, err := p.parseFactor()
	if err != nil {
		return 0, err
	}

	for {
		p.skipSpaces()
		if p.match('*') {
			next, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			value *= next
			continue
		}
		if p.match('/') {
			next, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			if next == 0 {
				return 0, fmt.Errorf("%w: division by zero", ErrInvalidExpression)
			}
			value /= next
			continue
		}
		return value, nil
	}
}

func (p *expressionParser) parseFactor() (float64, error) {
	p.skipSpaces()

	if p.match('+') {
		return p.parseFactor()
	}
	if p.match('-') {
		value, err := p.parseFactor()
		return -value, err
	}
	if p.match('(') {
		value, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		p.skipSpaces()
		if !p.match(')') {
			return 0, fmt.Errorf("%w: missing closing parenthesis", ErrInvalidExpression)
		}
		return value, nil
	}

	return p.parseNumber()
}

func (p *expressionParser) parseNumber() (float64, error) {
	p.skipSpaces()
	start := p.pos
	dotCount := 0

	for p.pos < len(p.input) {
		r := rune(p.input[p.pos])
		if unicode.IsDigit(r) {
			p.pos++
			continue
		}
		if r == '.' {
			dotCount++
			if dotCount > 1 {
				return 0, fmt.Errorf("%w: malformed number", ErrInvalidExpression)
			}
			p.pos++
			continue
		}
		break
	}

	if start == p.pos {
		return 0, fmt.Errorf("%w: expected number", ErrInvalidExpression)
	}

	value, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil || math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, fmt.Errorf("%w: malformed number", ErrInvalidExpression)
	}

	return value, nil
}

func (p *expressionParser) skipSpaces() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *expressionParser) match(expected byte) bool {
	if p.pos >= len(p.input) || p.input[p.pos] != expected {
		return false
	}
	p.pos++
	return true
}

func formatNumber(value float64) string {
	if math.Mod(value, 1) == 0 {
		return strconv.FormatInt(int64(value), 10)
	}

	return strconv.FormatFloat(value, 'f', -1, 64)
}
