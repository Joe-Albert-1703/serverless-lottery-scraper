package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"main.go/common"
)

func GetAllResults(w http.ResponseWriter, r *http.Request) {
	if err := common.CrawlAndSaveResults(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch results: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(common.LotteryResultsData)
}
