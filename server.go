package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	maxLinksPerRequest = 100
	maxBodySize        = 1 << 20
	checkTimeout       = 30 * time.Second
)

type checkRequest struct {
	Links []string `json:"links"`
}

type checkResponse struct {
	Links    map[string]string `json:"links"`
	LinksNum int               `json:"links_num"`
}

type reportRequest struct {
	LinksList []int `json:"links_list"`
}

type Server struct {
	storage *Storage
	checker *LinkChecker
	pdf     *PDFGenerator
}

func NewServer(storage *Storage, checker *LinkChecker, pdf *PDFGenerator) *Server {
	return &Server{storage: storage, checker: checker, pdf: pdf}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /check", s.handleCheck)
	mux.HandleFunc("POST /report", s.handleReport)
	return mux
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	var req checkRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	switch {
	case len(req.Links) == 0:
		writeError(w, http.StatusBadRequest, "links must not be empty")
		return
	case len(req.Links) > maxLinksPerRequest:
		writeError(w, http.StatusBadRequest, "too many links, max "+strconv.Itoa(maxLinksPerRequest))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), checkTimeout)
	defer cancel()
	results := s.checker.CheckLinks(ctx, req.Links)

	id, err := s.storage.SaveLinkSet(results)
	if err != nil {
		// The result is still valid for the client; only persistence failed.
		log.Printf("persist link set %d: %v", id, err)
	}
	writeJSON(w, http.StatusOK, checkResponse{Links: results, LinksNum: id})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	var req reportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.LinksList) == 0 {
		writeError(w, http.StatusBadRequest, "links_list must not be empty")
		return
	}

	sets := s.storage.GetLinkSets(req.LinksList)
	if len(sets) == 0 {
		writeError(w, http.StatusNotFound, "no link sets found for given ids")
		return
	}

	pdf, err := s.pdf.GeneratePDF(sets)
	if err != nil {
		log.Printf("generate pdf: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to generate report")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="link_report.pdf"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	_, _ = w.Write(pdf)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
