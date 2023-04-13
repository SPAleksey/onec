package server

import (
	"github.com/AlekseySP/onec/onec"
	"html/template"
	"strconv"
)

type IndexTable struct {
	Title          string
	Hyperlink      string
	NumberOfFields string
	RowLenth       string
	DataOffset     string
	BlobOffset     string
	Done           bool
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
		"          <th class=\"done\"><a href={{.Hyperlink}}>{{.Title}}</a></th>\n        " +
		"     {{else}}\n            " +
		"          <th><a href={{.Hyperlink}}>{{.Title}}</a></th>\n        " +
		"     {{end}}\n    " +
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
			Title:          "Name",
			Hyperlink:      "",
			NumberOfFields: "NumberOfFields",
			RowLenth:       "RowLenth",
			DataOffset:     "DataOffset",
			BlobOffset:     "BlobOffset",
		}},
	}

	for _, v := range b.TablesName {
		ts := b.TableDescription[v]
		IndexT := IndexTable{
			Title:          ts.Name,
			Hyperlink:      "table/" + ts.Name,
			NumberOfFields: strconv.Itoa(len(ts.FieldsName)),
			RowLenth:       strconv.Itoa(ts.RowLength),
			DataOffset:     strconv.Itoa(ts.DataOffset),
			BlobOffset:     strconv.Itoa(ts.BlobOffset),
			Done:           true,
		}
		data.Tables = append(data.Tables, IndexT)
	}

	return data
}

func PageTable(table string) *template.Template {

	pageIndex := "<h1>{{.PageTitle}}</h1>\n" +
		"<ul>\n" +
		"{{range .Tables}}\n        " +
		"{{if .Done}}\n            " +
		"<li class=\"done\"><a href={{.Hyperlink}}>{{.Title}}</a></li>\n        " +
		"{{else}}\n            " +
		"<li><a href={{.Hyperlink}}>{{.Title}}</a></li>\n        " +
		"{{end}}\n    " +
		"{{end}}\n" +
		"</ul>"

	tmpl := template.New("index")
	tmpl, err := tmpl.Parse(pageIndex)
	if err != nil {
		panic("err parse table template")
	}

	return tmpl
}

func PageTableData(b *onec.BaseOnec, table string) IndexPageData {

	data := IndexPageData{
		PageTitle: "table: " + b.TableDescription[table].Name,
		Tables:    []IndexTable{},
	}
	//IndexTables := make([]IndexTable, len(b.TableDescription))
	for k, _ := range b.TableDescription[table].Fields {
		IndexTable := IndexTable{
			Title:     k,
			Hyperlink: "table/" + k,
			Done:      true,
		}
		data.Tables = append(data.Tables, IndexTable)
	}

	return data
}
