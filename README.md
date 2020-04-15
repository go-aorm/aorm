# AORM

The fantastic Advanced ORM library for Golang, aims to be developer friendly.

This is a fork of [GORM](https://github.com/jinzhu/gorm) with improved performance and many other features.

[![go report card](https://goreportcard.com/badge/github.com/moisespsena-go/aorm "go report card")](https://goreportcard.com/report/github.com/moisespsena-go/aorm)
[![wercker status](https://app.wercker.com/status/8596cace912c9947dd9c8542ecc8cb8b/s/master "wercker status")](https://app.wercker.com/project/byKey/8596cace912c9947dd9c8542ecc8cb8b)
[![Join the chat at https://gitter.im/moisespsena-go/aorm](https://img.shields.io/gitter/room/moisespsena-go/aorm.svg)](https://gitter.im/moisespsena-go/aorm?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Open Collective Backer](https://opencollective.com/gorm/tiers/backer/badge.svg?label=backer&color=brightgreen "Open Collective Backer")](https://opencollective.com/gorm)
[![Open Collective Sponsor](https://opencollective.com/gorm/tiers/sponsor/badge.svg?label=sponsor&color=brightgreen "Open Collective Sponsor")](https://opencollective.com/gorm)
[![MIT license](http://img.shields.io/badge/license-MIT-brightgreen.svg)](http://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/moisespsena-go/aorm?status.svg)](https://godoc.org/github.com/moisespsena-go/aorm)

## Overview

* Full-Featured ORM (almost)
* Associations (Has One, Has Many, Belongs To, Many To Many, Polymorphism)
* Hooks (Before/After Create/Save/Update/Delete/Find)
* Preloading (eager loading)
* Transactions
* Composite Primary Key
* SQL Builder
* Auto Migrations
* Logger
* Extendable, write Plugins based on GORM callbacks
* Every feature comes with tests
* Developer Friendly
* **NEW:** Inline model loader and auto loader
* **NEW:** Null values scaner without Nullable Field
* **NEW:** Log callbacks
* **NEW:** After scanner callbacks  
* **NEW:** Binary ID (using bid.BID)
* **NEW:** Index/Unique Index withe WHERE clause
* **NEW:** Raw args binder
* **NEW:** Custom arg binder by dialect
* **NEW:** Custom arg valuer by dialect
* **NEW:** POLYMORPHIC_VALUE tag: `sql:"POLYMORPHIC_VALUE:user"` or singular and plural: `sql:"POLYMORPHIC_VALUE:user,users"`
* **NEW:** Dry Run `db.DryRun().First(&user)`
* **NEW:** Model Struct Api: `aorm.StructOf(&User{})`
* **NEW:** Model Struct interface: `aorm.InstaceOf(&User{})`
* **NEW:** ID Api: `aorm.IdOf(&User{ID:1})` 
* **NEW:** Get executed query and args `fmt.Println(db.First(&user).Query)` or `fmt.Println(db.DryRun().First(&user).Query)`
* **NEW:** Readonly fields with select query
* **NEW:** Args and Query API `db.First(&user, aorm.Query{"name = ?", []interface{}{"joe"}})` or your type implements `aorm.Clauser`
* **NEW:** Money type
* **NEW:** Dynamic table namer
* **NEW:** and more other changes...

## Installation

**WINDOWS NOTE:** run these commands inside [CygWin](http://www.cygwin.org/) or [Git Bash](git-scm.com/download/win).

```bash
go get -u github.com/moisespsena-go/aorm
cd $GOPATH/github.com/moisespsena-go/aorm
go generate
```
    
### GO DEP

If uses [dep](https://golang.github.io/dep/) for manage your dependencies, runs:

```bash
cd YOUR_PROJECT_DIR
dep ensure -add github.com/moisespsena-go/aorm
cd vendor/github.com/moisespsena-go/aorm
go generate
```

## Getting Started

* AORM Guides [http://io](http://io)

## Contributing

[You can help to deliver a better GORM, check out things you can do](http://io/contribute.html)

## License

© Moises P. Sena, 2018~time.Now

Released under the [MIT License](https://github.com/moisespsena-go/aorm/blob/master/License)

## Related Projects

* [GORM](https://github.com/jinzhu/gorm)
