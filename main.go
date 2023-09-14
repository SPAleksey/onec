package main

import (
	"flag"
	"fmt"
	"github.com/AlekseySP/onec/cmd"
	"github.com/AlekseySP/onec/onec"
	"github.com/AlekseySP/onec/server"
	"os"
)

var flagS string
var flagI string

func init() {
	flag.StringVar(&flagS, "b", "", "Path to 1CV8.1CD base or run in base folder")
	flag.StringVar(&flagI, "p", "80", "Port of http server or 8081 default")
}

func main() {
	//debug.SetGCPercent(-1)
	flag.Parse()
	cmd.CheckFlag(&flagS)

	db, err := os.Open(flagS)
	if err != nil {
		panic("File not exist?")
		return
	}
	defer db.Close()

	BaseOnec, err := onec.OpenBaseOnec(db)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = server.Start(BaseOnec, flagI)
	if err != nil {
		fmt.Println(err)
		return
	}
}
