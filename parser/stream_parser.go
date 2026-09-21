package parser

import (
	"bytes"
	"errors"
	"io"
	"iter"
)

// StreamParser wraps an io.Reader byte stream and produces parsed Exprs on the fly.
type StreamParser struct {
	// P is the inner Parser holding the token queue and parsing status.
	P   *Parser
	r   io.Reader
	buf []byte
}

// NewStreamParser creates a new StreamParser wrapping the given reader with a default parser.
func NewStreamParser(r io.Reader) *StreamParser {
	return &StreamParser{
		P:   New(),
		r:   r,
		buf: make([]byte, 4096),
	}
}

// NewStreamParserWithParser creates a new StreamParser with a pre-configured parser.
func NewStreamParserWithParser(r io.Reader, p *Parser) *StreamParser {
	return &StreamParser{
		P:   p,
		r:   r,
		buf: make([]byte, 4096),
	}
}

// Next parses and returns the next complete Expr from the stream.
// Returns io.EOF when the stream is exhausted cleanly.
func (s *StreamParser) Next() (Expr, error) {
	if s.P.Status.IsError() {
		return Expr{}, io.EOF
	}

	for {
		// 1. If we already have a completed expr queued in P.Exprs, yield it immediately
		if e, ok := s.P.PopExpr(); ok {
			return *e, nil
		}

		// 2. Try to parse any tokens currently in the parser
		if err := s.P.Parse(); err != nil {
			return Expr{}, err
		}

		// If parsing produced complete expression(s), yield the first one
		if e, ok := s.P.PopExpr(); ok {
			return *e, nil
		}

		// 3. Need more data: read next chunk from inner reader
		n, err := s.r.Read(s.buf)
		if n > 0 {
			if terr := s.P.Tokenize(bytes.NewReader(s.buf[:n])); terr != nil {
				s.P.Status = NewStatusError()
				return Expr{}, NewErrCorruptData("failed to tokenize chunk from stream")
			}
			continue
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				// Inner stream reached EOF
				// If parser is left in an incomplete state, that's an unexpected EOF
				if s.P.Status.IsIncomplete() {
					s.P.Status = NewStatusError()
					s.P.Tokens = nil
					return Expr{}, NewErrInvalidToken("unexpected EOF in expression")
				}
				return Expr{}, io.EOF
			}
			s.P.Status = NewStatusError()
			return Expr{}, NewErrInvalidToken("stream read error")
		}
	}
}

// All reads and returns all expressions from the stream until io.EOF.
func (s *StreamParser) All() ([]Expr, error) {
	var results []Expr
	for {
		e, err := s.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		results = append(results, e)
	}
	return results, nil
}

// Iter returns an iterator yielding (Expr, error) pairs for range loops.
func (s *StreamParser) Iter() iter.Seq2[Expr, error] {
	return func(yield func(Expr, error) bool) {
		for {
			e, err := s.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				yield(Expr{}, err)
				return
			}
			if !yield(e, nil) {
				return
			}
		}
	}
}
