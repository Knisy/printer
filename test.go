package main

import (
	"fmt"
)

func main() {
	printer := new(Printer)
	printer.open("GP-5860III")
	printer.PrintPostScriptFile("t.txt", "printing")
	fmt.Println(printer)
}
