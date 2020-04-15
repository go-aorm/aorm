Your issue may already be reported! Please search on the [issue track](https://github.com/moisespsena-go/aorm/issues) before creating one.

### What version of Go are you using (`go version`)?


### Which database and its version are you using?


### Please provide a complete runnable program to reproduce your issue. **IMPORTANT**

Need to runnable with [AORM's docker compose config](https://github.com/moisespsena-go/aorm/blob/master/docker-compose.yml) or please provides your config.

```go
package main

import (
	"github.com/moisespsena-go/aorm"
	_ "github.com/moisespsena-go/aorm/dialects/mssql"
	_ "github.com/moisespsena-go/aorm/dialects/mysql"
	_ "github.com/moisespsena-go/aorm/dialects/postgres"
	_ "github.com/moisespsena-go/aorm/dialects/sqlite"
)

var db *aorm.DB

func init() {
	var err error
	db, err = aorm.Open("sqlite3", "test.db")
	// db, err = Open("postgres", "user=gorm password=gorm DB.name=aorm port=9920 sslmode=disable")
	// db, err = Open("mysql", "aorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True")
	// db, err = Open("mssql", "sqlserver://aorm:LoremIpsum86@localhost:9930?database=aorm")
	if err != nil {
		panic(err)
	}
	db.LogMode(true)
}

func main() {
	if /* failure condition */ {
		fmt.Println("failed")
	} else {
		fmt.Println("success")
	}
}
```
