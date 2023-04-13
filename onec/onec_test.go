package onec

import (
	"fmt"
	"os"
	"testing"
)

func TestFromFormat1C(t *testing.T) {
	testCases := []struct {
		name          string
		value         []byte
		fieldType     string
		expectedValue string
	}{{
		name: "Менеджер по закупкам",
		value: []byte{20, 0, 28, 4, 53, 4, 61, 4, 53, 4, 52, 4, 54, 4, 53, 4, 64, 4, 32, 0, 63, 4, 62, 4, 32, 0,
			55, 4, 48, 4, 58, 4, 67, 4, 63, 4, 58, 4, 48, 4, 60, 4, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0,
			32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0,
			32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0,
			32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0,
			32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0,
			32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0,
			32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0, 32, 0,
			32, 0, 32, 0, 32, 0, 32, 0},
		fieldType:     "NVC",
		expectedValue: "Менеджер по закупкам",
	}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			Value := FromFormat1C(tc.value, tc.fieldType)
			if Value != tc.expectedValue {
				t.Error(
					"For", tc.value,
					"expected", tc.expectedValue,
					"got", Value,
				)
			}
		})
	}
}

func BenchmarkBOReader(b *testing.B) {
	//path := "C:/Del/UTDemo/KA/1Cv8.1CD"
	path := "C:/GO/onec/py/tests/fixtures/Platform8Demo/8-3-8_4K.1CD"
	//path := "C:/Del/UTDemo/ERP2/1Cv8.1CD"

	db, err := os.Open(path)
	if err != nil {
		fmt.Println("error ", err)
	}
	defer db.Close()

	for i := 0; i < b.N; i++ {
		BO, _ := DatabaseReader(db)
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
