package main

import (
	"fmt"
	"github.com/AlekseySP/onec/onec"
	"os"
	"time"
)

func main() {

	path := "C:/GO/onec/py/tests/fixtures/Platform8Demo/8-3-8_4K.1CD"
	//path := "C:/Del/UTDemo/KA/1Cv8.1CD"
	//path := "C:/Del/UTDemo/ERP2/1Cv8.1CD"
	db, err := os.Open(path)
	if err != nil {
		fmt.Println("error ", err)
	}
	defer db.Close()

	dl := 0
	start := time.Now()

	BO := onec.DatabaseReader(db)
	for n := 0; n < 10; n++ {
		obj := BO.Rows("V8USERS", n)
		fmt.Println(obj.RepresentObject["NAME"])
	}
	/*
		obj := BO.Rows("V8USERS", 2)
		for k, v := range obj.RepresentObject {
			fmt.Println(k)
			fmt.Println(v)
		}
		/*
			for key, v := range BO.TableDescription["V8USERS"].Fields {
				fmt.Println(key)
				fmt.Println(v)
				dl += v.DataLength
			}
	*/

	fmt.Println(len(BO.TableDescription))
	//fmt.Println(BO.TableDescription)
	duration := time.Since(start)
	fmt.Println(duration)
	m := onec.ReadBlockOfReplacemant(BO, BO.TableDescription["V8USERS"])
	fmt.Println(m)

	fmt.Println("data lenth ", dl)

}
