package rawdata

import (
	"errors"
	"io"
	"iter"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// RawDataGenerator wraps a StreamParser and yields parsed dynamic Data elements on the fly.
type RawDataGenerator struct {
	P *parser.StreamParser
}

// NewRawDataGenerator creates a new RawDataGenerator wrapping the given byte reader.
func NewRawDataGenerator(r io.Reader) *RawDataGenerator {
	return &RawDataGenerator{
		P: parser.NewStreamParser(r),
	}
}

// NewRawDataGeneratorWithParser creates a new RawDataGenerator wrapping an existing StreamParser.
func NewRawDataGeneratorWithParser(p *parser.StreamParser) *RawDataGenerator {
	return &RawDataGenerator{
		P: p,
	}
}

// Next parses and returns the next Data element from the stream.
// Returns io.EOF when the stream is exhausted cleanly.
func (g *RawDataGenerator) Next() (Data, error) {
	expr, err := g.P.Next()
	if err != nil {
		return Data{}, err
	}
	return DataFromExpr(&expr)
}

// All reads and returns all Data elements from the stream until io.EOF.
func (g *RawDataGenerator) All() ([]Data, error) {
	var results []Data
	for {
		d, err := g.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}

// Iter returns an iterator yielding (Data, error) pairs for range loops.
func (g *RawDataGenerator) Iter() iter.Seq2[Data, error] {
	return func(yield func(Data, error) bool) {
		for {
			d, err := g.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				yield(Data{}, err)
				return
			}
			if !yield(d, nil) {
				return
			}
		}
	}
}

