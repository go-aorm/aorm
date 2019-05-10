package export_sql_funcs

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func Run() (err error) {
	pth := filepath.Join(os.Getenv("GOROOT"), filepath.FromSlash("src/database/sql/githubComMoisespsenaGoAorm.go"))
	return ioutil.WriteFile(pth, []byte(`package sql

var (
	GithubComMoisespsenaGoAormConvertAssign = convertAssign
	GithubComMoisespsenaGoAormConvertAssignRows = convertAssignRows
)
`), 0664)
}
