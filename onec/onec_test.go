package onec

import (
	"fmt"
	"os"
	"testing"
)

func BenchmarkBOReader(b *testing.B) {
	//path := "C:/Del/UTDemo/KA/1Cv8.1CD"
	//path := "C:/GO/onec/py/tests/fixtures/Platform8Demo/8-3-8_4K.1CD"
	path := "C:/Del/UTDemo/ERP2/1Cv8.1CD"

	db, err := os.Open(path)
	if err != nil {
		fmt.Println("error ", err)
	}
	defer db.Close()

	for i := 0; i < b.N; i++ {
		BO := DatabaseReader(db)
		//_ = BO
		//fmt.Println(len(BO.TableDescription))
		_ = BO
	}
	//fmt.Println(len(BO.TableDescription))
}

/*
	{"IBVERSION",0,
	{"Fields",
	{"IBVERSION","N",0,10,0,"CS"},
	{"PLATFORMVERSIONREQ","N",0,10,0,"CS"}
	},
	{"Indexes"},
	{"Recordlock","0"},
	{"Files",20,0,0}
	}

Match: {"IBVERSION",0,
{"Fields",
v: 1
Match: {"IBVERSION","N",0,10,0,"CS"},
{"PLATFORMVERSIONREQ","N",0,10,0,"CS"}
v: 2
Match:
v: 3
Match: 0
v: 4
Match: 4,0,0
v: 5


*/
