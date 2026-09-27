package main

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/ccqpein/lisp-rpc-golang/rawdata"
)

func main() {
	pr, pw := io.Pipe()

	// 1. Simulate a client streaming S-expressions in arbitrary chunk fragments over a stream
	go func() {
		defer pw.Close()

		chunks := []string{
			`(login :user "alice" :role "admin") `,
			`(telemetry :device "sensor-42" `,
			`:temp 24) `,
			`(message :text "Streaming with UTF-8: `,
			`Hello 世界 🚀" :seq 1)`,
		}

		for i, chunk := range chunks {
			time.Sleep(20 * time.Millisecond)
			fmt.Printf("[Stream Writer] Sending chunk #%d: %s\n", i+1, chunk)
			if _, err := pw.Write([]byte(chunk)); err != nil {
				return
			}
		}
	}()

	// 2. Server consumes parsed Data items incrementally as they arrive
	gen := rawdata.NewRawDataGenerator(pr)
	itemNum := 1

	for {
		item, err := gen.Next()
		if err != nil {
			if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "closed") {
				break
			}
			log.Fatalf("Stream parsing error: %v", err)
		}

		fmt.Printf("\n[Stream Receiver] Successfully assembled item #%d:\n", itemNum)
		fmt.Printf("  Raw: %s\n", item.String())
		if item.IsExpr() {
			expr := item.Expr()
			fmt.Printf("  Expression Name: %s\n", expr.GetName())
			for _, arg := range expr.RestArgs {
				fmt.Printf("    %s => %s\n", arg.Key.String(), arg.Val.String())
			}
		}
		itemNum++
	}

	fmt.Println("\n[Stream] Stream completed cleanly.")
}
