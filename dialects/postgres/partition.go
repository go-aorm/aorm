package postgres

import (
	"fmt"
	"regexp"
	"time"

	"github.com/moisespsena-go/bid"

	"github.com/moisespsena-go/aorm"
)

const (
	DatePartitionYearly DatePartitionType = iota
	DatePartitionMonthly
	DatePartitionMonthlyDiv2
	DatePartitionDaily
)

type DatePartitionType uint8

type TableDatePartitionSpec struct {
	Name     string
	From, To time.Time
}

func (this TableDatePartitionSpec) TableName(parentTable string) string {
	return parentTable + "_" + this.Name
}

func (this TableDatePartitionSpec) CreateSQL(parentTable, schema string) string {
	if schema == "" {
		schema = "public"
	}
	return `CREATE TABLE IF NOT EXISTS "` + schema + `"."` + this.TableName(parentTable) + `" PARTITION OF "` + parentTable + `"` +
		` FOR VALUES FROM ('` + this.From.Format("2006-01-02") + `') TO ('` + this.To.Format("2006-01-02") + `')`
}

func TablePartitionsByBid2Weeks(values ...bid.BID) (partitions []TableDatePartitionSpec) {
	var weeks = map[string]interface{}{}
	for _, b := range values {
		var (
			t = b.Time().UTC()
			y = t.Year()
			m = t.Month()
			w = 1
		)
		if t.Day() > 15 {
			w = 2
		}
		name := fmt.Sprintf("y%04dm%02dmw%d", y, m, w)
		if _, ok := weeks[name]; !ok {
			weeks[name] = nil
			var (
				start = time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
				end   = time.Date(y, m, 16, 0, 0, 0, 0, t.Location())
			)
			if w == 2 {
				end = start.AddDate(0, 1, 0)
				start = time.Date(y, m, 16, 0, 0, 0, 0, t.Location())
			}
			partitions = append(partitions, TableDatePartitionSpec{name, start, end})
		}
	}
	return
}

func TablePartitionsByYear(values ...bid.BID) (partitions []TableDatePartitionSpec) {
	var yearls = map[string]interface{}{}
	for _, b := range values {
		var (
			t = b.Time().UTC()
			y = t.Year()
		)
		name := fmt.Sprintf("y%04d", y)
		if _, ok := yearls[name]; !ok {
			yearls[name] = nil
			var (
				start = time.Date(y, 1, 1, 0, 0, 0, 0, t.Location())
				end   = start.AddDate(1, 0, 0)
			)
			partitions = append(partitions, TableDatePartitionSpec{name, start, end})
		}
	}
	return
}

func TablePartitionsByMonth(values ...bid.BID) (partitions []TableDatePartitionSpec) {
	var yearls = map[string]interface{}{}
	for _, b := range values {
		var (
			t = b.Time().UTC()
			y = t.Year()
			m = t.Month()
		)
		name := fmt.Sprintf("y%04dm%02d", y, m)
		if _, ok := yearls[name]; !ok {
			yearls[name] = nil
			var (
				start = time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
				end   = start.AddDate(0, 1, 0)
			)
			partitions = append(partitions, TableDatePartitionSpec{name, start, end})
		}
	}
	return
}

func TablePartitionsByDay(values ...bid.BID) (partitions []TableDatePartitionSpec) {
	var days = map[string]interface{}{}
	for _, b := range values {
		var (
			t = b.Time().UTC()
			y = t.Year()
			m = t.Month()
			d = t.Day()
		)
		name := fmt.Sprintf("y%04dm%02dd%02d", y, m, d)
		if _, ok := days[name]; !ok {
			days[name] = nil
			var (
				start = time.Date(y, m, d, 0, 0, 0, 0, t.Location())
				end   = start.AddDate(0, 0, 1)
			)
			partitions = append(partitions, TableDatePartitionSpec{name, start, end})
		}
	}
	return
}

func BidPartition(model *aorm.ModelStruct, typ DatePartitionType, schema string) {
	model.Indexes["!"+model.PrimaryField().Name] = &aorm.StructIndex{
		Fields: model.PrimaryFields,
	}
	model.CreateTable(aorm.Before, func(data *aorm.TypeCallbackData) {
		data.Scope.Query.Query = regexp.MustCompile(`(?ims),\s*primary\s+key\s+\([^)]+\)\s*`).ReplaceAllString(data.Scope.Query.Query, "")
		data.Scope.Query.Query += fmt.Sprintf(" PARTITION BY RANGE (public.pgbid_get_utc_date(%s))", model.PrimaryField().DBName)
	})
	model.BeforeCreate(aorm.Before, func(data *aorm.ScopeCallbackData) {
		pk := data.Scope.PrimaryKey().(bid.BID)
		var partition TableDatePartitionSpec
		switch typ {
		case DatePartitionDaily:
			partition = TablePartitionsByDay(pk)[0]
		case DatePartitionMonthlyDiv2:
			partition = TablePartitionsByBid2Weeks(pk)[0]
		case DatePartitionMonthly:
			partition = TablePartitionsByMonth(pk)[0]
		default:
			partition = TablePartitionsByYear(pk)[0]
		}
		if schema != "" && schema != "public" {
			if _, err := data.Scope.SQLDB().Exec(`CREATE SCHEMA IF NOT EXISTS "` + schema + `"`); err != nil {
				data.Scope.Err(err)
				return
			}
		}
		mainTbName := data.Scope.TableName()
		query := partition.CreateSQL(mainTbName, schema)
		if _, err := data.Scope.SQLDB().Exec(query); err != nil {
			data.Scope.Err(err)
		}
	})
}
