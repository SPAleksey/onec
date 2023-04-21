package server

import (
	"github.com/AlekseySP/onec/onec"
	"html/template"
	"strconv"
)

type IndexTable struct {
	Title                string
	Hyperlink            string
	HyperlinkDescription string
	NumberOfFields       string
	RowLenth             string
	DataOffset           string
	BlobOffset           string
	Done                 bool
}

type IndexPageData struct {
	PageTitle string
	Tables    []IndexTable
}

func PageIndex() *template.Template {

	pageIndex := "<h1>{{.PageTitle}}</h1>\n" +
		"<table>\n" +
		"  {{range .Tables}}\n        " +
		"   <tr>" +
		"     {{if .Done}}\n            " +
		"          <th align=\"left\" class=\"done\"><a href={{.Hyperlink}}>{{.Title}}</a></th>\n        " +
		"     {{else}}\n            " +
		"          <th align=\"left\"><a href={{.Hyperlink}}>{{.Title}}</a></th>\n        " +
		"     {{end}}\n    " +
		"          <th align=\"left\"><a href={{.HyperlinkDescription}}>description</a></th>\n        " +
		"          <th>{{.NumberOfFields}}</th>\n        " +
		"          <th>{{.RowLenth}}</th>\n        " +
		"          <th>{{.DataOffset}}</th>\n        " +
		"          <th>{{.BlobOffset}}</th>\n        " +
		"   </tr>\n" +
		"  {{end}}\n" +
		"</table>"

	tmpl := template.New("index")
	tmpl, err := tmpl.Parse(pageIndex)
	if err != nil {
		panic("err parse index template")
	}

	return tmpl
}

func PageIndexData(b *onec.BaseOnec) IndexPageData {

	data := IndexPageData{
		PageTitle: "1CV8.1CD page size: " + strconv.Itoa(int(b.HeadDB.PageSize)),
		Tables: []IndexTable{{
			Title:                "Name",
			Hyperlink:            "",
			HyperlinkDescription: "",
			NumberOfFields:       "NumberOfFields",
			RowLenth:             "RowLenth",
			DataOffset:           "DataOffset",
			BlobOffset:           "BlobOffset",
		}},
	}

	for _, v := range b.TablesName {
		ts := b.TableDescription[v]
		IndexT := IndexTable{
			Title:                ts.Name,
			Hyperlink:            "table/" + ts.Name,
			HyperlinkDescription: "tabledescription/" + ts.Name,
			NumberOfFields:       strconv.Itoa(len(ts.FieldsName)),
			RowLenth:             strconv.Itoa(ts.RowLength),
			DataOffset:           strconv.Itoa(ts.DataOffset),
			BlobOffset:           strconv.Itoa(ts.BlobOffset),
			Done:                 true,
		}
		data.Tables = append(data.Tables, IndexT)
	}

	return data
}

type DataTableDescription struct {
	PageTitle         string
	Hyperlink         string
	TablesDescription []TableDescription
}

type TableDescription struct {
	Name            string
	FieldType       string
	NullExist       string
	Lenth           string
	Precision       string
	CaseSensitive   string
	DataFieldOffset string
	DataLength      string
}

func PageTableDescription() *template.Template {

	pageTableDescription := "<h1><a href=\"\\\">BASE </a>{{.PageTitle}}</h1>\n" +
		" <h1><a href={{.Hyperlink}}>data</a></h1>\n        " +
		"<table>\n" +
		"  {{range .TablesDescription}}\n        " +
		"   <tr>" +
		"          <th align=\"left\">{{.Name}}</th>\n        " +
		"          <th>{{.FieldType}}</th>\n        " +
		"          <th>{{.NullExist}}</th>\n        " +
		"          <th>{{.Lenth}}</th>\n        " +
		"          <th>{{.Precision}}</th>\n        " +
		"          <th>{{.CaseSensitive}}</th>\n        " +
		"          <th>{{.DataFieldOffset}}</th>\n        " +
		"          <th>{{.DataLength}}</th>\n        " +
		"   </tr>\n" +
		"  {{end}}\n" +
		"</table>"

	tmpl := template.New("tableDescription")
	tmpl, err := tmpl.Parse(pageTableDescription)
	if err != nil {
		panic("err parse table description template")
	}

	return tmpl
}

func PageTableDescriptionData(b *onec.BaseOnec, table string) DataTableDescription {

	data := DataTableDescription{
		PageTitle: "table: " + b.TableDescription[table].Name,
		Hyperlink: "/table/" + table,
		TablesDescription: []TableDescription{{
			Name:            "Name",
			FieldType:       "Field Type",
			NullExist:       "Null exist",
			Lenth:           "Lenth",
			Precision:       "Precision",
			CaseSensitive:   "Case Sensitive",
			DataFieldOffset: "Data Field Offset",
			DataLength:      "Data lenth",
		}},
	}

	for _, v := range b.TableDescription[table].FieldsName {
		ts := b.TableDescription[table].Fields[v]
		TD := TableDescription{
			Name:            ts.Name,
			FieldType:       ts.FieldType,
			NullExist:       strconv.FormatBool(ts.NullExist),
			Lenth:           strconv.Itoa(ts.Lenth),
			Precision:       strconv.Itoa(ts.Precision),
			CaseSensitive:   strconv.FormatBool(ts.CaseSensitive),
			DataFieldOffset: strconv.Itoa(ts.DataFieldOffset),
			DataLength:      strconv.Itoa(ts.DataLength),
		}
		data.TablesDescription = append(data.TablesDescription, TD)
	}

	return data
}

type FieldsN struct {
	FieldsName string
}

type ValuesF struct {
	Fields []FieldsN
}

type TablePageData struct {
	PageTitle            string
	HyperLinkDescription string
	Values               []ValuesF
}

func PageTable() *template.Template {

	pageTable := "<h1><a href=\"\\\">BASE </a>{{.PageTitle}}</h1>\n" +
		" <h1><a href={{.HyperLinkDescription}}>table description</a></h1>\n        " +
		"<table>\n" +
		"  {{range .Values}}\n        " + //rows
		"   <tr>" +
		"      {{range .Fields}}\n        " + //columns
		"          <th>{{.FieldsName}}</th>\n        " +
		"      {{end}}\n" +
		"   </tr>\n" +
		"  {{end}}\n" +
		"</table>"

	tmpl := template.New("table")
	tmpl, err := tmpl.Parse(pageTable)
	if err != nil {
		panic("err parse table template")
	}

	return tmpl
}

func PageTableData(b *onec.BaseOnec, table string) TablePageData {

	var dataValuesF []ValuesF
	//var dataFieldsN []FieldsN

	data := TablePageData{
		PageTitle:            "table: " + b.TableDescription[table].Name,
		HyperLinkDescription: "/tabledescription/" + table,
		Values:               []ValuesF{},
	}

	dataFieldsN := make([]FieldsN, len(b.TableDescription[table].FieldsName))

	for k, v := range b.TableDescription[table].FieldsName {
		dataFieldsN[k] = FieldsN{v}
	}
	dataValuesF = append(dataValuesF, ValuesF{dataFieldsN})

	if !b.TableDescription[table].NoRecords {
		for n := 0; n < 10; n++ {
			dataFieldsN := make([]FieldsN, len(dataFieldsN))
			obj := b.Rows(b.TableDescription[table].Name, n)
			if obj.Deleted { //do not show deleted object (lenth 5 byte{1}deleted{4}next free object)
				continue
			}
			if b.TableDescription[table].NoRecords {
				break
			}
			for k, v := range b.TableDescription[table].FieldsName {
				dataFieldsN[k] = FieldsN{obj.RepresentObject[v]}
			}
			dataValuesF = append(dataValuesF, ValuesF{dataFieldsN})
		}
	}
	data.Values = dataValuesF

	return data
}
