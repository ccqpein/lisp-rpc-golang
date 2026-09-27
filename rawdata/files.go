package rawdata

import (
	"fmt"
	"os"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// DataFile represents a collection of S-expression data records parsed from a file.
type DataFile struct {
	// Datas contains all ExprData records in this file.
	Datas []ExprData
}

// NewDataFile parses all S-expression data records from the specified file path.
// Each expression in the file must be a named S-expression data record (ExprData).
func NewDataFile(path string) (*DataFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file at %s: %w", path, err)
	}
	defer file.Close()

	p := parser.New()
	if err := p.Tokenize(file); err != nil {
		return nil, err
	}
	if err := p.Parse(); err != nil {
		return nil, err
	}

	df := &DataFile{
		Datas: make([]ExprData, 0, len(p.Exprs)),
	}

	for i := range p.Exprs {
		d, err := DataFromExpr(&p.Exprs[i])
		if err != nil {
			return nil, fmt.Errorf("generate Data failed: %w", err)
		}
		if !d.IsExpr() {
			return nil, NewDataError("file can only contain expr data", ErrInvalidInput)
		}
		df.Datas = append(df.Datas, *d.Expr())
	}

	return df, nil
}

// GenTable generates a map indexing each ExprData record by its symbol name identifier.
func (df *DataFile) GenTable() map[string]*ExprData {
	table := make(map[string]*ExprData, len(df.Datas))
	for i := range df.Datas {
		record := &df.Datas[i]
		table[record.GetName()] = record
	}
	return table
}

