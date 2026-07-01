package main

import (
	"os"

	"github.com/neolao/batocera-scrap-manager/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout))
}
