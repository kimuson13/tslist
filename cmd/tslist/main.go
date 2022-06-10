package main

import (
	"tslist"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(tslist.Analyzer) }
