package parser

import "slices"

// ParsingStatusKind indicates the intermediate state of the parser.
type ParsingStatusKind int

const (
	// StatusClean indicates no in-progress expression. The parser is clean.
	StatusClean ParsingStatusKind = iota
	// StatusInReadExpr indicates an unclosed list expression `(...)`.
	StatusInReadExpr
	// StatusInReadString indicates an unclosed string literal `"..."`.
	StatusInReadString
	// StatusInReadQuote indicates a quote `'` waiting for its target expression.
	StatusInReadQuote
	// StatusInReadKeyword indicates a keyword prefix `:` waiting for its name.
	StatusInReadKeyword
	// StatusInReadComment indicates a comment `;...` waiting for terminating newline.
	StatusInReadComment
	// StatusInReadAtom indicates waiting for an atom token.
	StatusInReadAtom
	// StatusError indicates a syntax error, unexpected EOF, or corrupt data.
	StatusError
)

// ParsingStatus represents the intermediate state of the parser when an input
// token stream ends before a full expression could be parsed.
type ParsingStatus struct {
	Kind   ParsingStatusKind
	Tokens []string
	Child  *ParsingStatus
}

// NewStatusClean creates a clean ParsingStatus.
func NewStatusClean() ParsingStatus {
	return ParsingStatus{Kind: StatusClean}
}

// NewStatusError creates an error ParsingStatus.
func NewStatusError() ParsingStatus {
	return ParsingStatus{Kind: StatusError}
}

// NewStatusInReadExpr creates a StatusInReadExpr status.
func NewStatusInReadExpr(tokens []string, child *ParsingStatus) ParsingStatus {
	return ParsingStatus{Kind: StatusInReadExpr, Tokens: tokens, Child: child}
}

// NewStatusInReadString creates a StatusInReadString status.
func NewStatusInReadString(tokens []string, child *ParsingStatus) ParsingStatus {
	return ParsingStatus{Kind: StatusInReadString, Tokens: tokens, Child: child}
}

// NewStatusInReadQuote creates a StatusInReadQuote status.
func NewStatusInReadQuote(tokens []string, child *ParsingStatus) ParsingStatus {
	return ParsingStatus{Kind: StatusInReadQuote, Tokens: tokens, Child: child}
}

// NewStatusInReadKeyword creates a StatusInReadKeyword status.
func NewStatusInReadKeyword(tokens []string, child *ParsingStatus) ParsingStatus {
	return ParsingStatus{Kind: StatusInReadKeyword, Tokens: tokens, Child: child}
}

// NewStatusInReadComment creates a StatusInReadComment status.
func NewStatusInReadComment(tokens []string, child *ParsingStatus) ParsingStatus {
	return ParsingStatus{Kind: StatusInReadComment, Tokens: tokens, Child: child}
}

// NewStatusInReadAtom creates a StatusInReadAtom status.
func NewStatusInReadAtom(tokens []string, child *ParsingStatus) ParsingStatus {
	return ParsingStatus{Kind: StatusInReadAtom, Tokens: tokens, Child: child}
}

// IsClean returns true if the status is StatusClean.
func (s ParsingStatus) IsClean() bool {
	return s.Kind == StatusClean
}

// IsError returns true if the status is StatusError.
func (s ParsingStatus) IsError() bool {
	return s.Kind == StatusError
}

// IsIncomplete returns true if the status represents an in-progress expression.
func (s ParsingStatus) IsIncomplete() bool {
	return s.Kind != StatusClean && s.Kind != StatusError
}

// CollectScannedTokens collects all scanned tokens in this status hierarchy.
func (s ParsingStatus) CollectScannedTokens() []string {
	if s.Kind == StatusClean || s.Kind == StatusError {
		return nil
	}
	res := make([]string, len(s.Tokens))
	copy(res, s.Tokens)
	if s.Child != nil {
		res = append(res, s.Child.CollectScannedTokens()...)
	}
	return res
}

// Equal compares two ParsingStatus instances for equality.
func (s ParsingStatus) Equal(other ParsingStatus) bool {
	if s.Kind != other.Kind {
		return false
	}
	if !slices.Equal(s.Tokens, other.Tokens) {
		return false
	}
	if s.Child == nil && other.Child == nil {
		return true
	}
	if s.Child == nil || other.Child == nil {
		return false
	}
	return s.Child.Equal(*other.Child)
}

// ParsedExprKind indicates whether a parse attempt completed or was incomplete.
type ParsedExprKind int

const (
	// ParsedCompleted indicates an expression was completely parsed.
	ParsedCompleted ParsedExprKind = iota
	// ParsedIncomplete indicates the token stream ran out before completing.
	ParsedIncomplete
)

// ParsedExpr represents the result of attempting to parse an expression from the token queue.
type ParsedExpr struct {
	Kind   ParsedExprKind
	Expr   Expr
	Status ParsingStatus
}

// NewParsedCompleted creates a completed ParsedExpr.
func NewParsedCompleted(e Expr) ParsedExpr {
	return ParsedExpr{Kind: ParsedCompleted, Expr: e}
}

// NewParsedIncomplete creates an incomplete ParsedExpr with its ParsingStatus.
func NewParsedIncomplete(s ParsingStatus) ParsedExpr {
	return ParsedExpr{Kind: ParsedIncomplete, Status: s}
}

// IsCompleted returns true if this expression successfully completed.
func (p ParsedExpr) IsCompleted() bool {
	return p.Kind == ParsedCompleted
}

// IsIncomplete returns true if this expression is incomplete.
func (p ParsedExpr) IsIncomplete() bool {
	return p.Kind == ParsedIncomplete
}

// IntoExpr converts the ParsedExpr into an (Expr, bool), returning (Expr, true)
// if completed or (Expr{}, false) if incomplete.
func (p ParsedExpr) IntoExpr() (Expr, bool) {
	if p.Kind == ParsedCompleted {
		return p.Expr, true
	}
	return Expr{}, false
}

// StatusVal returns a pointer to the inner ParsingStatus if incomplete.
func (p ParsedExpr) StatusVal() *ParsingStatus {
	if p.Kind == ParsedIncomplete {
		return &p.Status
	}
	return nil
}

// Equal compares two ParsedExpr instances for equality.
func (p ParsedExpr) Equal(other ParsedExpr) bool {
	if p.Kind != other.Kind {
		return false
	}
	if p.Kind == ParsedCompleted {
		return p.Expr.Equal(other.Expr)
	}
	return p.Status.Equal(other.Status)
}
