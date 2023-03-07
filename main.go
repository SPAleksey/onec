package main

import (
	"fmt"
	"github.com/AlekseySP/onec/onec"
	"os"
	"regexp"
	"time"
)

func main() {

	path := "C:/GO/onec/py/tests/fixtures/Platform8Demo/8-3-8_4K.1CD"
	path = "C:/Del/UTDemo/KA/1Cv8.1CD"
	db, err := os.Open(path)
	if err != nil {
		fmt.Println("error ", err)
	}
	defer db.Close()

	start := time.Now()
	BO := onec.DatabaseReader(db)
	fmt.Println(len(BO.TableDescription))
	duration := time.Since(start)
	fmt.Println(duration)

	/*
		{"IBVERSION",0,
		{"Fields",
		{"IBVERSION","N",0,10,0,"CS"},
		{"PLATFORMVERSIONREQ","N",0,10,0,"CS"}
		},
		{"Indexes"},
		{"Recordlock","0"},
		{"Files",20,0,0}
		}	 */

	//table_description_pattern_text := "{x*"
	//fmt.Println(BO.TableDescription[0])
	Pattern, err := regexp.Compile("{\"([a - zA-Z]*)\"...\n{\"([A-Za - z]*)")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("----")
	subStr := Pattern.FindStringSubmatch("{\"IBVERSION\",0,\n{\"Fields\"")
	//subStr := Pattern.FindStringSubmatch(BO.TableDescription[0])
	for v, s := range subStr {
		fmt.Println("Match:", s)
		fmt.Println("v:", v)
	}
	fmt.Println(subStr)
}
