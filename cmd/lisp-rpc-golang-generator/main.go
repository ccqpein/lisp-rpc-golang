package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ccqpein/lisp-rpc-golang/generator"
)

func main() {
	var inputFile string
	flag.StringVar(&inputFile, "input", "", "Path to input .lisprpc specification file")
	flag.StringVar(&inputFile, "i", "", "Path to input .lisprpc specification file (shorthand)")
	flag.StringVar(&inputFile, "input-file", "", "Path to input .lisprpc specification file (compatibility)")

	var outputPath string
	flag.StringVar(&outputPath, "output", ".", "Directory to generate output package into")
	flag.StringVar(&outputPath, "o", ".", "Directory to generate output package into (shorthand)")
	flag.StringVar(&outputPath, "output-path", ".", "Directory to generate output package into (compatibility)")

	var withServer bool
	flag.BoolVar(&withServer, "with-server", false, "Impl the struct and rpc trait for rpc server")
	flag.BoolVar(&withServer, "w", false, "Impl the struct and rpc trait for rpc server (shorthand)")

	flag.Parse()

	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: input specification file is required (-input or -i)")
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening spec file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	sf, err := generator.ParseSpecFile(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing spec file: %v\n", err)
		os.Exit(1)
	}

	genArg := generator.GenerateArgFromBool(withServer)
	if err := sf.GenCode(outputPath, genArg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated Lisp-RPC Go library '%s' in %s\n", sf.GetTargetPkgName(), outputPath)
}
