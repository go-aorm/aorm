package aorm

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Query struct {
	Query string
	Args  Vars
}

func (this Query) Wrap(prefix, sufix string) Query {
	this.Query = prefix + this.Query + sufix
	return this
}

func (this *Query) AddArgs(arg ...interface{}) *Query {
	this.Args = append(this.Args, arg...)
	return this
}

func (this Query) Build(appender ToVarsAppender) (query string, err error) {
	replacements := []string{}
	for _, arg := range this.Args {
		switch t := arg.(type) {
		case IDValuer:
			arg = t.Raw()
		case SqlArger:
			replacements = append(replacements, appender.AddToVars(t.SqlArg()))
			continue
		}
		switch reflect.ValueOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			if scanner, ok := arg.(driver.Valuer); ok {
				arg, err = scanner.Value()
				replacements = append(replacements, appender.AddToVars(arg))
			} else if b, ok := arg.([]byte); ok {
				replacements = append(replacements, appender.AddToVars(b))
			} else if as, ok := arg.([][]interface{}); ok {
				var tempMarks []string
				for _, a := range as {
					var arrayMarks []string
					for _, v := range a {
						arrayMarks = append(arrayMarks, appender.AddToVars(v))
					}

					if len(arrayMarks) > 0 {
						tempMarks = append(tempMarks, fmt.Sprintf("(%v)", strings.Join(arrayMarks, ",")))
					}
				}

				if len(tempMarks) > 0 {
					replacements = append(replacements, strings.Join(tempMarks, ","))
				}
			} else if values := reflect.ValueOf(arg); values.Len() > 0 {
				var tempMarks []string
				for i := 0; i < values.Len(); i++ {
					tempMarks = append(tempMarks, appender.AddToVars(values.Index(i).Interface()))
				}
				replacements = append(replacements, strings.Join(tempMarks, ","))
			} else {
				replacements = append(replacements, appender.AddToVars(Expr("NULL")))
			}
		default:
			replacements = append(replacements, appender.AddToVars(arg))
		}

		if err != nil {
			return
		}
	}

	buff := bytes.NewBuffer([]byte{})
	i := 0
	for _, s := range this.Query {
		if s == '?' && len(replacements) > i {
			buff.WriteString(replacements[i])
			i++
		} else {
			buff.WriteRune(s)
		}
	}

	return buff.String(), nil
}

func (this Query) String() string {
	if this.Query == "" {
		return ""
	}
	var b bytes.Buffer
	b.WriteString("<< " + this.Query + " >>")
	if len(this.Args) > 0 {
		b.WriteString("\nArgs:\n")

		for i, arg := range this.Args {
			var name string
			if named, ok := arg.(sql.NamedArg); ok {
				name, arg = named.Name, named.Value
			} else {
				name = strconv.Itoa(i + 1)
			}
			var theArg func(arg interface{})
			theArg = func(arg interface{}) {
				if arg == nil {
					fmt.Fprintf(&b, "  - %s: <nil>\n", name)
					return
				}
				typ := reflect.TypeOf(arg)
				line := fmt.Sprintf("  - %s: %v[%s] ", name, indirectType(typ).PkgPath(), typ)
				if isNil(reflect.ValueOf(arg)) {
					b.WriteString(line + "<nil>\n")
				} else {
					var empty bool
					switch at := arg.(type) {
					case ProtectedStringer:
						arg = HiddenStringerValue
					case string:
						if at == "" {
							empty = true
						}
					case *string:
						if *at == "" {
							empty = true
						}
					case driver.Valuer:
						if v, err := at.Value(); err == nil {
							if fmt.Sprintf("%v", v) == fmt.Sprintf("%v", at) {
								if z, ok := v.(Zeroer); ok && z.IsZero() {
									empty = true
								}
								goto ok
							}
							theArg(v)
							return
						}
					case Zeroer:
						if at.IsZero() {
							empty = true
						}
					}
				ok:
					if empty {
						b.WriteString(line + "<empty>\n")
						return
					}
					switch at := arg.(type) {
					case time.Time:
						arg = at.Format("2006-01-02T15:04:05-0700")
					case *time.Time:
						arg = at.Format("2006-01-02T15:04:05-0700")
					}
					b.WriteString(line + fmt.Sprint(arg) + "\n")
				}
			}
			theArg(arg)
		}
	}
	return b.String()
}
