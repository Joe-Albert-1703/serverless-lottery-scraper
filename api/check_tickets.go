package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"main.go/common"
)

func CheckTickets(w http.ResponseWriter, r *http.Request) {
	var tickets []string
	if err := json.NewDecoder(r.Body).Decode(&tickets); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := common.CrawlAndSaveResults(false); err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch results: %v", err), http.StatusInternalServerError)
		return
	}

	winners := make(map[string]map[string][]string)
	for lotteryName, results := range common.LotteryResultsData.Results {
		currentWinners := common.CheckWinningTickets(results, tickets)
		for pos, winningTickets := range currentWinners {
			if winners[pos] == nil {
				winners[pos] = make(map[string][]string)
			}
			winners[pos][lotteryName] = append(winners[pos][lotteryName], winningTickets...)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(winners)
}
