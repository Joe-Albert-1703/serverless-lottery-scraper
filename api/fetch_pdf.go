package handler

import (
	"fmt"
	"io"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	pdfLink := r.URL.Query().Get("pdfLink")
	if pdfLink == "" {
		http.Error(w, "pdfLink query parameter is required", http.StatusBadRequest)
		return
	}

	resp, err := http.Get(pdfLink)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("failed to download PDF: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read PDF content: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}
