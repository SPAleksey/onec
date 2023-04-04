package main

import (
	"fmt"
	"github.com/AlekseySP/onec/onec"
	"os"
	"time"
)

func main() {

	//path := "C:/GO/onec/py/tests/fixtures/Platform8Demo/8-3-8_4K.1CD"
	//path = "C:/Del/UTDemo/KA/1Cv8.1CD"
	path := "C:/Del/UTDemo/ERP2/1Cv8.1CD"
	db, err := os.Open(path)
	if err != nil {
		fmt.Println("error ", err)
	}
	defer db.Close()

	start := time.Now()
	BO := onec.DatabaseReader(db)
	duration := time.Since(start)
	fmt.Println(len(BO.TableDescription))
	fmt.Println(duration)
}
