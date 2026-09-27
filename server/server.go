package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

// HandlerFunc is the internal function signature for invoking a registered RPC handler.
type HandlerFunc func(ctx context.Context, rawData string) (string, error)

// Server is the Lisp-RPC dispatch server engine managing registered RPC handlers.
// It also implements http.Handler for native Go HTTP server integration.
type Server struct {
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
}

// New creates a new empty Server instance.
func New() *Server {
	return &Server{
		handlers: make(map[string]HandlerFunc),
	}
}

// ExtractCommandName extracts the RPC command symbol name from a raw S-expression string.
// For example, "(get-book :title \"foo\")" -> "get-book".
func ExtractCommandName(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "(") || !strings.HasSuffix(trimmed, ")") {
		return "", errors.New("Invalid RPC format")
	}
	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	fields := strings.Fields(inner)
	if len(fields) == 0 {
		return "", errors.New("Invalid RPC format")
	}
	return fields[0], nil
}

// Register registers an RPC handler function.
// The function signature can be:
//   - func(req T) (R, error)
//   - func(ctx context.Context, req T) (R, error)
func (s *Server) Register(fn any) (*Server, error) {
	return s.registerInternal("", fn)
}

// RegisterNamed registers an RPC handler function with an explicit command name.
func (s *Server) RegisterNamed(name string, fn any) (*Server, error) {
	return s.registerInternal(name, fn)
}


func (s *Server) registerInternal(explicitName string, fn any) (*Server, error) {
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		return nil, errors.New("handler must be a function")
	}
	fnType := fnVal.Type()
	if fnType.NumIn() < 1 || fnType.NumIn() > 2 {
		return nil, errors.New("handler function must take 1 or 2 arguments")
	}
	if fnType.NumOut() < 1 || fnType.NumOut() > 2 {
		return nil, errors.New("handler function must return 1 or 2 values")
	}

	var hasContext bool
	var reqType reflect.Type

	if fnType.NumIn() == 2 {
		ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if !fnType.In(0).Implements(ctxType) {
			return nil, errors.New("first argument must be context.Context when taking 2 arguments")
		}
		hasContext = true
		reqType = fnType.In(1)
	} else {
		reqType = fnType.In(0)
	}

	var commandName string
	if explicitName != "" {
		commandName = explicitName
	} else {
		name, err := determineCommandName(reqType)
		if err != nil {
			return nil, err
		}
		commandName = name
	}

	handler := func(ctx context.Context, rawData string) (string, error) {
		var reqVal reflect.Value
		var targetPtr any
		if reqType.Kind() == reflect.Pointer {
			reqVal = reflect.New(reqType.Elem())
			targetPtr = reqVal.Interface()
		} else {
			reqVal = reflect.New(reqType)
			targetPtr = reqVal.Interface()
		}

		if err := DeserializeLisp(rawData, targetPtr); err != nil {
			return "", fmt.Errorf("Deserialization failed: %w", err)
		}

		var inArgs []reflect.Value
		if hasContext {
			inArgs = append(inArgs, reflect.ValueOf(ctx))
		}
		if reqType.Kind() == reflect.Pointer {
			inArgs = append(inArgs, reqVal)
		} else {
			inArgs = append(inArgs, reqVal.Elem())
		}

		out := fnVal.Call(inArgs)
		if len(out) == 2 && !out[1].IsNil() {
			return "", out[1].Interface().(error)
		}

		return SerializeLisp(out[0].Interface())
	}

	s.mu.Lock()
	s.handlers[commandName] = handler
	s.mu.Unlock()

	return s, nil
}

func determineCommandName(t reflect.Type) (string, error) {
	toRPCType := reflect.TypeOf((*ToRPCType)(nil)).Elem()
	if t.Implements(toRPCType) {
		inst := reflect.Zero(t).Interface().(ToRPCType)
		rpcType := inst.ToRPCType()
		if rpcType.Kind == RPCTypeRPC {
			return rpcType.Name, nil
		}
		return "", errors.New("Handler function argument has to be RPCType::RPC")
	}
	ptrType := reflect.PointerTo(t)
	if ptrType.Implements(toRPCType) {
		inst := reflect.New(t).Interface().(ToRPCType)
		rpcType := inst.ToRPCType()
		if rpcType.Kind == RPCTypeRPC {
			return rpcType.Name, nil
		}
		return "", errors.New("Handler function argument has to be RPCType::RPC")
	}

	name := t.Name()
	if t.Kind() == reflect.Pointer {
		name = t.Elem().Name()
	}
	if name != "" {
		return toKebabCase(name), nil
	}
	return "", errors.New("cannot determine RPC command name for type")
}

// Handle dispatches a raw S-expression string and returns the serialized response.
func (s *Server) Handle(rawData string) (string, error) {
	return s.HandleContext(context.Background(), rawData)
}

// HandleContext dispatches a raw S-expression string with context and returns the serialized response.
func (s *Server) HandleContext(ctx context.Context, rawData string) (string, error) {
	cmd, err := ExtractCommandName(rawData)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	handler, ok := s.handlers[cmd]
	s.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("Unknown command: %s", cmd)
	}

	return handler(ctx, rawData)
}


// ServeHTTP implements net/http.Handler for native Go HTTP server integration.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	rawReq := string(bodyBytes)
	resp, err := s.HandleContext(r.Context(), rawReq)
	if err != nil {
		if strings.Contains(err.Error(), "Unknown command") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/x-lisp-rpc; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(resp))
}
