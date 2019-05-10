package main

import (
	"os"

	export_sql_funcs "github.com/moisespsena-go/aorm/build-hooks/pre-build/export-sql-funcs"
)

func main() {
	if err := export_sql_funcs.Run(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
