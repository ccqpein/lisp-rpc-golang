package parser

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Parser tokenizes and parses Lisp S-expressions.
type Parser struct {
	// readNumberConfig: will read numbers if true; otherwise numbers are parsed as symbols.
	readNumberConfig bool

	// Tokens is the token queue populated by Tokenize.
	Tokens []string

	// Status represents the current intermediate parsing status.
	Status ParsingStatus

	// Exprs holds parsed expression nodes populated by Parse.
	Exprs []Expr

	recording bool
	recorded  []string

	// byteCache holds unprocessed bytes buffered across chunk boundaries (e.g. split multi-byte UTF-8 sequences).
	byteCache []byte
}

// New creates a new Parser instance with default configuration.
func New() *Parser {
	return &Parser{
		readNumberConfig: true,
		Status:           NewStatusClean(),
	}
}

// ConfigReadNumber configures whether numeric strings are parsed into numbers rather than symbols.
func (p *Parser) ConfigReadNumber(v bool) *Parser {
	p.readNumberConfig = v
	return p
}

// Tokenize tokenizes the input reader into the token queue.
//
// Reads in buffered chunks, handles multi-byte UTF-8 split across chunk boundaries,
// skips empty chunks, and optimizes token extraction without intermediate allocations.
func (p *Parser) Tokenize(sourceCode io.Reader) error {
	buf := make([]byte, 4096)
	for {
		n, err := sourceCode.Read(buf)
		if n > 0 {
			p.byteCache = append(p.byteCache, buf[:n]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
	}

	if len(p.byteCache) == 0 {
		return nil
	}

	// Validate and separate valid UTF-8 slice from any incomplete trailing multi-byte sequence
	var validLen int
	for validLen < len(p.byteCache) {
		if utf8.FullRune(p.byteCache[validLen:]) {
			r, size := utf8.DecodeRune(p.byteCache[validLen:])
			if r == utf8.RuneError && size == 1 {
				return NewErrCorruptData("invalid utf-8 sequence")
			}
			validLen += size
		} else {
			// Incomplete rune at end of byteCache
			break
		}
	}

	validStr := string(p.byteCache[:validLen])
	remainingLen := len(p.byteCache) - validLen

	var res []string
	lastStart := 0

	for i := 0; i < len(validStr); i++ {
		c := validStr[i]
		switch c {
		case '(', ' ', ')', '\'', '"', ':', '\n', ';':
			if lastStart < i {
				res = append(res, validStr[lastStart:i])
			}

			if c == ' ' {
				prevIsSpace := false
				if len(res) > 0 {
					prevIsSpace = res[len(res)-1] == " "
				} else if len(p.Tokens) > 0 {
					prevIsSpace = p.Tokens[len(p.Tokens)-1] == " "
				}
				if !prevIsSpace {
					res = append(res, " ")
				}
			} else {
				res = append(res, string(c))
			}

			lastStart = i + 1
		}
	}

	if lastStart < len(validStr) {
		res = append(res, validStr[lastStart:])
	}

	// Keep any trailing incomplete UTF-8 bytes for the next chunk
	if remainingLen > 0 {
		remStart := len(p.byteCache) - remainingLen
		p.byteCache = append([]byte(nil), p.byteCache[remStart:]...)
	} else {
		p.byteCache = p.byteCache[:0]
	}

	p.Tokens = append(p.Tokens, res...)
	return nil
}

// PopToken pops the next token from the queue, recording it if token recording is active.
func (p *Parser) PopToken() (string, bool) {
	if len(p.Tokens) == 0 {
		return "", false
	}
	tok := p.Tokens[0]
	p.Tokens = p.Tokens[1:]
	if p.recording {
		p.recorded = append(p.recorded, tok)
	}
	return tok, true
}

// RestoreScannedTokens restores all scanned tokens from an incomplete status back into the front of the token queue.
func (p *Parser) RestoreScannedTokens() {
	status := p.Status
	p.Status = NewStatusClean()
	restored := status.CollectScannedTokens()
	p.Tokens = append(restored, p.Tokens...)
}

// Clear clears the tokens, expressions, byte cache, and resets parsing status.
func (p *Parser) Clear() {
	p.ClearExprs()
	p.ClearTokens()
	p.byteCache = nil
	p.Status = NewStatusClean()
	p.recorded = nil
	p.recording = false
}

// ClearTokens clears the tokens in the parser.
func (p *Parser) ClearTokens() {
	p.Tokens = nil
}

// ClearExprs clears the expressions in the parser.
func (p *Parser) ClearExprs() {
	p.Exprs = nil
}

// Parse parses all tokens in the parser into expression nodes.
func (p *Parser) Parse() error {
	if p.Status.IsError() {
		return NewErrCorruptData("parser is in an error state")
	}

	for {
		if len(p.Tokens) == 0 && p.Status.IsClean() {
			break
		}

		res, err := p.ParseOne()
		if err != nil {
			p.Status = NewStatusError()
			return err
		}
		if res.IsIncomplete() {
			break
		}
	}

	return nil
}

// ParseOne parses a single expression from the token queue.
func (p *Parser) ParseOne() (ParsedExpr, error) {
	if p.Status.IsError() {
		return ParsedExpr{}, NewErrCorruptData("parser is in an error state")
	}

	if !p.Status.IsClean() {
		p.RestoreScannedTokens()
	}

	p.recorded = nil
	p.recording = false

	for {
		if len(p.Tokens) == 0 {
			p.recorded = nil
			p.recording = false
			return NewParsedIncomplete(NewStatusClean()), nil
		}

		front := p.Tokens[0]
		if front == " " || front == "\n" {
			p.PopToken()
			continue
		}

		router, err := p.ReadRouter(front)
		if err != nil {
			p.Status = NewStatusError()
			return ParsedExpr{}, err
		}

		res, err := router()
		if err != nil {
			p.Status = NewStatusError()
			return ParsedExpr{}, err
		}

		if res.IsCompleted() {
			p.Exprs = append(p.Exprs, res.Expr)
			p.Status = NewStatusClean()
		} else {
			p.Status = res.Status
		}

		p.recorded = nil
		p.recording = false
		return res, nil
	}
}

// ReadRouter returns the appropriate parse function for the given opening token.
func (p *Parser) ReadRouter(token string) (func() (ParsedExpr, error), error) {
	switch token {
	case "(":
		return p.ReadExp, nil
	case "'":
		return p.ReadQuote, nil
	case "\"":
		return p.ReadString, nil
	case ":":
		return p.ReadKeyword, nil
	case ";":
		return p.ReadComment, nil
	default:
		return p.ReadAtom, nil
	}
}

// ReadAtom reads an atom expression from the token queue.
func (p *Parser) ReadAtom() (ParsedExpr, error) {
	tok, ok := p.PopToken()
	if !ok {
		return NewParsedIncomplete(NewStatusInReadAtom(nil, nil)), nil
	}

	if p.readNumberConfig {
		if n, err := strconv.ParseInt(tok, 10, 64); err == nil {
			return NewParsedCompleted(NewExprAtom(NewAtomNumber(NewInt(n)))), nil
		}
		if len(tok) > 0 {
			c := tok[0]
			if (c >= '0' && c <= '9') || c == '.' || c == '+' || c == '-' {
				if f, err := strconv.ParseFloat(tok, 64); err == nil {
					return NewParsedCompleted(NewExprAtom(NewAtomNumber(NewFloat(f)))), nil
				}
			}
		}
	}

	return NewParsedCompleted(NewExprAtom(NewAtomSymbol(tok))), nil
}

// ReadQuote reads a quoted expression from the token queue.
func (p *Parser) ReadQuote() (ParsedExpr, error) {
	quoteTok, ok := p.PopToken()
	if !ok {
		return NewParsedIncomplete(NewStatusInReadQuote(nil, nil)), nil
	}
	if quoteTok != "'" {
		return ParsedExpr{}, NewErrInvalidToken("expected ' in read_quote")
	}

	scanned := []string{quoteTok}

	if len(p.Tokens) == 0 {
		return NewParsedIncomplete(NewStatusInReadQuote(scanned, nil)), nil
	}

	router, err := p.ReadRouter(p.Tokens[0])
	if err != nil {
		return ParsedExpr{}, err
	}

	res, err := router()
	if err != nil {
		return ParsedExpr{}, err
	}

	if res.IsCompleted() {
		return NewParsedCompleted(NewExprQuote(res.Expr)), nil
	}

	childStatus := res.Status
	return NewParsedIncomplete(NewStatusInReadQuote(scanned, &childStatus)), nil
}

// ReadExp reads a list expression enclosed in parentheses.
func (p *Parser) ReadExp() (ParsedExpr, error) {
	openParen, ok := p.PopToken()
	if !ok {
		return NewParsedIncomplete(NewStatusInReadExpr(nil, nil)), nil
	}
	if openParen != "(" {
		return ParsedExpr{}, NewErrInvalidToken("expected '(' in read_exp")
	}

	scanned := []string{openParen}
	var res []Expr

	for {
		if len(p.Tokens) == 0 {
			return NewParsedIncomplete(NewStatusInReadExpr(scanned, nil)), nil
		}

		front := p.Tokens[0]
		if front == ")" {
			closing, _ := p.PopToken()
			scanned = append(scanned, closing)
			break
		}
		if front == " " || front == "\n" {
			sp, _ := p.PopToken()
			scanned = append(scanned, sp)
			continue
		}

		router, err := p.ReadRouter(front)
		if err != nil {
			return ParsedExpr{}, err
		}

		prevRec := len(p.recorded)
		p.recording = true
		childRes, err := router()
		if err != nil {
			if prevRec < len(p.recorded) {
				p.recorded = p.recorded[:prevRec]
			}
			return ParsedExpr{}, err
		}

		if childRes.IsCompleted() {
			newlyRecorded := p.recorded[prevRec:]
			scanned = append(scanned, newlyRecorded...)
			p.recorded = p.recorded[:prevRec]
			res = append(res, childRes.Expr)
		} else {
			p.recorded = p.recorded[:prevRec]
			childStatus := childRes.Status
			return NewParsedIncomplete(NewStatusInReadExpr(scanned, &childStatus)), nil
		}
	}

	return NewParsedCompleted(NewExprList(res)), nil
}

// ReadString reads a string literal enclosed in double quotes.
func (p *Parser) ReadString() (ParsedExpr, error) {
	openQuote, ok := p.PopToken()
	if !ok {
		return NewParsedIncomplete(NewStatusInReadString(nil, nil)), nil
	}
	if openQuote != "\"" {
		return ParsedExpr{}, NewErrInvalidToken("expected '\"' in read_string")
	}

	scanned := []string{openQuote}
	escape := false
	var res strings.Builder

	for {
		thisToken, ok := p.PopToken()
		if !ok {
			return NewParsedIncomplete(NewStatusInReadString(scanned, nil)), nil
		}

		if escape {
			res.WriteString(thisToken)
			escape = false
			scanned = append(scanned, thisToken)
			continue
		}

		isQuote := thisToken == "\""
		isEscape := thisToken == "\\"

		if isEscape {
			escape = true
		} else if !isQuote {
			res.WriteString(thisToken)
		}

		scanned = append(scanned, thisToken)

		if isQuote {
			break
		}
	}

	return NewParsedCompleted(NewExprAtom(NewAtomString(res.String()))), nil
}

// ReadKeyword reads a keyword token prefixed with a colon.
func (p *Parser) ReadKeyword() (ParsedExpr, error) {
	colon, ok := p.PopToken()
	if !ok {
		return NewParsedIncomplete(NewStatusInReadKeyword(nil, nil)), nil
	}
	if colon != ":" {
		return ParsedExpr{}, NewErrInvalidToken("expected ':' in read_keyword")
	}

	tok, ok := p.PopToken()
	if !ok {
		scanned := []string{colon}
		return NewParsedIncomplete(NewStatusInReadKeyword(scanned, nil)), nil
	}

	return NewParsedCompleted(NewExprAtom(NewAtomKeyword(tok))), nil
}

// ReadComment reads a comment line prefixed with a semicolon.
func (p *Parser) ReadComment() (ParsedExpr, error) {
	semi, ok := p.PopToken()
	if !ok {
		return NewParsedIncomplete(NewStatusInReadComment(nil, nil)), nil
	}
	if semi != ";" {
		return ParsedExpr{}, NewErrInvalidToken("expected ';' in read_comment")
	}

	start := false
	var res strings.Builder
	scanned := []string{semi}

	for {
		thisToken, ok := p.PopToken()
		if !ok {
			break
		}

		if !start {
			switch thisToken {
			case ";", " ":
				scanned = append(scanned, thisToken)
				continue
			default:
				start = true
			}
		}

		isNewline := thisToken == "\n"
		if !isNewline {
			res.WriteString(thisToken)
		}
		scanned = append(scanned, thisToken)

		if isNewline {
			break
		}
	}

	trimmed := strings.TrimRight(res.String(), " \t\r\n")
	return NewParsedCompleted(NewExprComment(trimmed)), nil
}

// IterExpr returns the slice of parsed expressions.
func (p *Parser) IterExpr() []Expr {
	return p.Exprs
}

// PopExpr pops the front completed expression from the expression queue in FIFO order.
func (p *Parser) PopExpr() (*Expr, bool) {
	if len(p.Exprs) == 0 {
		return nil, false
	}
	e := p.Exprs[0]
	p.Exprs = p.Exprs[1:]
	return &e, true
}
