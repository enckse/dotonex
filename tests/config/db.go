package main

import (
	"flag"
	"fmt"

	"github.com/tidwall/buntdb"
)

func main() {
	file := flag.String("database", "", "database to read")
	flag.Parse()
	db, err := buntdb.Open(*file)
	if err != nil {
		panic(fmt.Sprintf("failed to open (%v)", err))
	}
	err = db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			fmt.Printf("%s:::%s\n", key, value)
			return true
		})
		return err
	})
	if err != nil {
		panic("db view failed")
	}
}
