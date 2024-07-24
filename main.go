package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/romanpickl/pdf"
)

type WebScrape struct {
	LotteryName string `json:"lottery_name"`
	LotteryDate string `json:"lottery_date"`
	PdfLink     string `json:"pdf_link"`
}

type LotteryResults struct {
	LastUpdated time.Time                      `json:"latest_draw"`
	Results     map[string]map[string][]string `json:"results"`
}

var (
	lotteryResults LotteryResults

	numbersRegex      = regexp.MustCompile(`\d+`)
	alphanumericRegex = regexp.MustCompile(`\[([A-Z]+ \d+)\]`)
	seriesRegex       = regexp.MustCompile(`\[([A-Z])\]`)

	headerPattern             = `KERALA.*?( 1st)`
	footerPattern             = `Page \d  IT Support : NIC Kerala  \d{2}\/\d{2}\/\d{4} \d{2}:\d{2}:\d{2}`
	EndFooterPattern          = `The prize winners?.*`
	trailingWhiteSpacePattern = `\s{2}.\s`
	bulletPattern             = `(?:\d|\d{2})\)`
	podiumSplit               = `FOR +.* NUMBERS`
	lotteryTicketFull         = `[A-Z]{2} \d{6}`
	locationString            = `\(\S+\)`
	prizePositionString       = `((\d(st|rd|nd|th))|Cons)`
	prizeString               = `(Prize Rs :\d+/-)|(Prize-Rs :\d+/-)`
	seriesSelection           = `(?:\[)(.)`
)

func getAllResults(w http.ResponseWriter, r *http.Request) {
	if err := crawlAndSaveResults(false); err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch results: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lotteryResults)
}

func listLotteries(w http.ResponseWriter, r *http.Request) {
	lotteryList, err := getLotteryList(false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch lotteries: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lotteryList)
}

func checkTickets(w http.ResponseWriter, r *http.Request) {
	var tickets []string
	if err := json.NewDecoder(r.Body).Decode(&tickets); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := crawlAndSaveResults(false); err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch results: %v", err), http.StatusInternalServerError)
		return
	}

	winners := make(map[string]map[string][]string)
	for lotteryName, results := range lotteryResults.Results {
		currentWinners := checkWinningTickets(results, tickets)
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

func checkWinningTickets(results map[string][]string, tickets []string) map[string][]string {
	winners := make(map[string][]string)
	series := results["Series"]

	for _, ticket := range tickets {
		if !isMatchingSeries(series, ticket) {
			continue
		}
		checkTicketForWinningPositions(ticket, results, winners)
	}

	return winners
}

func isMatchingSeries(series []string, ticket string) bool {
	return len(series) > 0 && series[0] == string(ticket[0])
}

func checkTicketForWinningPositions(ticket string, results map[string][]string, winners map[string][]string) {
	for pos, nums := range results {
		if pos == "Series" {
			continue
		}
		if isWinningTicket(ticket, nums) {
			winners[pos] = append(winners[pos], ticket)
		}
	}
}

func isWinningTicket(ticket string, nums []string) bool {
	for _, num := range nums {
		if strings.Contains(ticket, num) {
			return true
		}
	}
	return false
}

func crawlAndSaveResults(firstVisit bool) error {
	lotteryList, err := getLotteryList(firstVisit)
	if err != nil {
		return fmt.Errorf("failed to fetch lottery list: %w", err)
	}
	if len(lotteryList) == 0 {
		return fmt.Errorf("no lottery list found")
	}

	// Update last updated date
	lotteryResults.LastUpdated, _ = time.Parse("02/01/2006", lotteryList[0].LotteryDate)

	// Process lottery results concurrently
	results, err := processLotteryResults(lotteryList)
	if err != nil {
		return err
	}

	lotteryResults.Results = results
	log.Println("Refreshed lottery results")

	return nil
}

func processLotteryResults(lotteryList []WebScrape) (map[string]map[string][]string, error) {
	results := make(map[string]map[string][]string)
	resultChan := make(chan struct {
		lotteryName string
		data        map[string][]string
		err         error
	}, len(lotteryList))

	for _, lottery := range lotteryList {
		go func(lottery WebScrape) {
			data, err := processLottery(lottery)
			resultChan <- struct {
				lotteryName string
				data        map[string][]string
				err         error
			}{lotteryName: lottery.LotteryName, data: data, err: err}
		}(lottery)
	}

	for range lotteryList {
		result := <-resultChan
		if result.err != nil {
			log.Printf("Error processing lottery %s: %v", result.lotteryName, result.err)
			continue
		}
		results[result.lotteryName] = result.data
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	return results, nil
}

func processLottery(lottery WebScrape) (map[string][]string, error) {
	if lottery.LotteryName == "" {
		return nil, nil
	}

	resp, err := http.Get(lottery.PdfLink)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download PDF for %s: %v", lottery.LotteryName, err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF content for %s: %v", lottery.LotteryName, err)
	}

	text, err := ExtractTextFromPDFContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from PDF for %s: %v", lottery.LotteryName, err)
	}

	return parseLotteryNumbers(text), nil
}

func getLotteryList(firstVisit bool) ([]WebScrape, error) {
	var datas []WebScrape
	now := time.Now().Local()
	today3pm := time.Date(now.Year(), now.Month(), now.Day(), 16, 15, 0, 0, now.Location())
	c := colly.NewCollector(colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"))

	c.OnHTML("tr", func(e *colly.HTMLElement) {
		href := e.ChildAttr("td a", "href")
		text := e.ChildText("td:first-child")
		text2 := e.ChildText("td:nth-child(2)")
		if text != "" {
			datas = append(datas, WebScrape{LotteryName: text, LotteryDate: text2, PdfLink: href})
		}
	})

	if firstVisit {
		c.Visit("https://statelottery.kerala.gov.in/index.php/lottery-result-view")
		return datas, nil
	}

	for {
		c.Visit("https://statelottery.kerala.gov.in/index.php/lottery-result-view")
		if len(datas) == 0 {
			log.Println("Error fetching lottery list, retrying...")
			time.Sleep(time.Minute * 10)
			continue
		}
		latestDate, err := time.Parse("02/01/2006", datas[0].LotteryDate)
		if err != nil {
			return nil, err
		} else if latestDate.Day() >= now.Day() || lotteryResults.LastUpdated.Day() < latestDate.Day() {
			lotteryResults.LastUpdated = latestDate
			break
		} else if latestDate.Day() <= now.Day() && now.Before(today3pm) {
			log.Println("current data is up to date...")
			break
		}
		log.Println("Latest data not available, checking again in 15 minutes...")
		time.Sleep(time.Minute * 15)
	}
	return datas, nil
}

func parseLotteryNumbers(input string) map[string][]string {
	result := make(map[string][]string)
	parts := strings.Split(input, "<")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		pos, numbersPart := parsePositionAndNumbersPart(part)
		addSeriesMatches(result, numbersPart)
		addAlphanumericMatches(result, pos, numbersPart)
		addNumericMatches(result, pos, numbersPart)
	}

	return result
}

func parsePositionAndNumbersPart(part string) (string, string) {
	pos := strings.TrimSpace(strings.Split(part, ">")[0])
	numbersPart := strings.TrimSpace(strings.SplitN(part, ">", 2)[1])
	return pos, numbersPart
}

func addSeriesMatches(result map[string][]string, numbersPart string) {
	seriesMatches := seriesRegex.FindAllStringSubmatch(numbersPart, -1)
	for _, match := range seriesMatches {
		if len(match) > 1 {
			result["Series"] = append(result["Series"], match[1])
		}
	}
}

func addAlphanumericMatches(result map[string][]string, pos, numbersPart string) {
	alphanumericMatches := alphanumericRegex.FindAllStringSubmatch(numbersPart, -1)
	for _, match := range alphanumericMatches {
		if len(match) > 1 {
			result[pos] = append(result[pos], match[1])
		}
	}
}

func addNumericMatches(result map[string][]string, pos, numbersPart string) {
	numericMatches := numbersRegex.FindAllString(numbersPart, -1)
	for _, match := range numericMatches {
		result[pos] = append(result[pos], match)
	}
}

func ExtractTextFromPDFContent(content []byte) (string, error) {
	finalString := ""
	r, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", err
	}

	for pageIndex := 1; pageIndex <= r.NumPage(); pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		rows, _ := p.GetTextByRow()
		for _, row := range rows {
			for _, word := range row.Content {
				finalString += word.S + " "
			}
		}
	}

	return ProcessTextContent(finalString)
}

func ProcessTextContent(input string) (string, error) {
	patternsToRemove := []string{headerPattern, footerPattern, bulletPattern, EndFooterPattern, trailingWhiteSpacePattern, locationString, podiumSplit, prizeString}
	for _, pattern := range patternsToRemove {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", err
		}
		if pattern == headerPattern {
			input = re.ReplaceAllString(input, "1st")
		}
		input = re.ReplaceAllString(input, "")
	}
	input = regexp.MustCompile(prizePositionString).ReplaceAllString(input, ` < $0 > `)
	input = regexp.MustCompile(lotteryTicketFull).ReplaceAllString(input, "[$0]")

	seriesMatches := regexp.MustCompile(seriesSelection).FindAllStringSubmatch(input, -1)
	if len(seriesMatches) > 0 {
		series := seriesMatches[0][1]
		input = fmt.Sprintf(`< Series > [%s] %s`, series, input)
	}
	return input, nil
}

func main() {
	http.HandleFunc("/api/v1/all_results", getAllResults)
	http.HandleFunc("/api/v1/list_lotteries", listLotteries)
	http.HandleFunc("/api/v1/check_tickets", checkTickets)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
