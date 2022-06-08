package main

import (
	"tslist"

	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(tslist.Analyzer) }
