package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"main.go/common"
)

func ListLotteries(w http.ResponseWriter, r *http.Request) {
	lotteryList, err := common.GetLotteryList()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch lotteries: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lotteryList)
}
