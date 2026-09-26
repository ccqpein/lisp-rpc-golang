package server

// RPCTypeKind represents the classification category of an RPC data structure or command.
type RPCTypeKind int

const (
	// RPCTypeMsg represents a named message type.
	RPCTypeMsg RPCTypeKind = iota
	// RPCTypeRPC represents a named RPC command.
	RPCTypeRPC
	// RPCTypeMap represents a quoted anonymous map.
	RPCTypeMap
	// RPCTypeList represents a quoted list sequence.
	RPCTypeList
	// RPCTypeV represents a primitive atom value.
	RPCTypeV
)

// RPCType classifies RPC data structures and commands, matching Rust RPCType.
type RPCType struct {
	Kind RPCTypeKind
	Name string
}

// NewRPCTypeMsg creates an RPCType for a named message.
func NewRPCTypeMsg(name string) RPCType {
	return RPCType{Kind: RPCTypeMsg, Name: name}
}

// NewRPCTypeRPC creates an RPCType for a named RPC command.
func NewRPCTypeRPC(name string) RPCType {
	return RPCType{Kind: RPCTypeRPC, Name: name}
}

// NewRPCTypeMap creates an RPCType for a quoted map.
func NewRPCTypeMap() RPCType {
	return RPCType{Kind: RPCTypeMap}
}

// NewRPCTypeList creates an RPCType for a quoted list.
func NewRPCTypeList() RPCType {
	return RPCType{Kind: RPCTypeList}
}

// NewRPCTypeV creates an RPCType for a primitive atom value.
func NewRPCTypeV() RPCType {
	return RPCType{Kind: RPCTypeV}
}

// ToRPCType is the trait interface for types that define an RPCType classification.
type ToRPCType interface {
	ToRPCType() RPCType
}

// LispSerializer is the interface for types that provide custom Lisp-RPC serialization.
type LispSerializer interface {
	SerializeLisp() (string, error)
}
