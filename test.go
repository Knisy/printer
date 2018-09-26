package main

import (
	"fmt"
)

func main() {
	printer := new(Printer)
	printer.open("print name")
	printer.PrintPostScriptFile("test.txt", "printing")
	fmt.Println(printer)
}
