package server

import (
	"github.com/AlekseySP/onec/onec"
	"github.com/gorilla/mux"
	"net/http"
)

type server struct {
	router *mux.Router
	base   *onec.BaseOnec
}

func NewServer(router *mux.Router, b *onec.BaseOnec) *server {
	s := &server{
		router: router,
		base:   b,
	}

	s.configureRouter()

	return s
}

func (s *server) configureRouter() {
	s.router.Handle("/", s.index())
	s.router.Handle("/table/{table}", s.table())
	s.router.Handle("/tabledescription/{table}", s.tabledescription())
}

func (s *server) index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl := PageIndex()
		data := PageIndexData(s.base)
		tmpl.Execute(w, data)
	}
}

func (s *server) table() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		table := mux.Vars(r)["table"]
		tmpl := PageTable()
		data := PageTableData(s.base, table)
		tmpl.Execute(w, data)
	}
}

func (s *server) tabledescription() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		table := mux.Vars(r)["table"]
		tmpl := PageTableDescription()
		data := PageTableDescriptionData(s.base, table)
		tmpl.Execute(w, data)
	}
}

func Start(b *onec.BaseOnec, port int) error {
	router := mux.NewRouter()
	server := NewServer(router, b)
	//_ = server
	return http.ListenAndServe(":80", server)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
