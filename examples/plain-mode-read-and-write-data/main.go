package main

import (
	"fmt"
	"log"

	"github.com/ccqpein/lisp-rpc-golang/rawdata"
)

func main() {
	// 1. Client creates structured S-expression data
	clientData, err := rawdata.NewData("rpc-call",
		rawdata.Pair{Key: "version", Val: int64(1)},
		rawdata.Pair{Key: "aa", Val: int64(2)},
	)
	if err != nil {
		log.Fatalf("Failed to create client data: %v", err)
	}

	rawString := clientData.String()
	fmt.Printf("[Client] Serialized request:\n  %s\n\n", rawString)

	// 2. Server parses the raw S-expression string into ExprData
	serverParsed, err := rawdata.DataFromRootStr(rawString, nil)
	if err != nil {
		log.Fatalf("Server parse error: %v", err)
	}

	fmt.Printf("[Server] Parsed data item: %s\n", serverParsed.String())
	fmt.Printf("[Server] Expression name: %s\n", serverParsed.Expr().GetName())

	versionVal := serverParsed.Get("version")
	aaVal := serverParsed.Get("aa")
	fmt.Printf("[Server] Extracted 'version': %s, 'aa': %s\n", versionVal.String(), aaVal.String())

	// 3. Server constructs response data
	respData, err := rawdata.DataFromStr(nil, `(response :args '(1 2) :result 3)`)
	if err != nil {
		log.Fatalf("Failed to construct response: %v", err)
	}
	fmt.Printf("[Server] Serialized response:\n  %s\n\n", respData.String())

	// 4. Client parses the response
	clientResp, err := rawdata.DataFromRootStr(respData.String(), nil)
	if err != nil {
		log.Fatalf("Client parse error: %v", err)
	}
	fmt.Printf("[Client] Received result: %s\n", clientResp.Get("result").String())
}
