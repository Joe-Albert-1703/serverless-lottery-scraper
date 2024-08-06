package common

import (
	"bytes"
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

// Data structures and global variables
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
	LotteryResultsData LotteryResults

	// Regex patterns and other constants
	numbersRegex      = regexp.MustCompile(`\d+`)
	alphanumericRegex = regexp.MustCompile(`\[([A-Z]+ \d+)\]`)
	seriesRegex       = regexp.MustCompile(`\[([A-Z])\]`)

	// Other patterns
	headerPattern             = `KERALA.*?( 1st)`
	footerPattern             = `Page \d  IT Support : NIC Kerala  \d{2}\/\d{2}\/\d{4} \d{2}:\d{2}:\d{2}`
	EndFooterPattern          = `The prize winners?.*`
	trailingWhiteSpacePattern = `\s{2}.\s`
	bulletPattern             = `(?:\d|\d{2})\)`
	podiumSplit               = `FOR +.* NUMBERS`
	lotteryTicketFull         = `[A-Z]{2} \d{6}`
	locationString            = `\(\S+\)`
	prizePositionString       = `(\d+(?:st|nd|rd|th) Prize-Rs :\d+\/-|\d+(?:st|nd|rd|th) Prize Rs :\d+\/-)`
	prizeString               = `(Prize-Rs)`
	seriesSelection           = `(?:\[)(.)`
)

// Utility functions
func CrawlAndSaveResults() error {
	lotteryList, err := GetLotteryList()
	if err != nil {
		return fmt.Errorf("failed to fetch lottery list: %w", err)
	}
	if len(lotteryList) == 0 {
		return fmt.Errorf("no lottery list found")
	}

	LotteryResultsData.LastUpdated, _ = time.Parse("02/01/2006", lotteryList[0].LotteryDate)
	results, err := ProcessLotteryResults(lotteryList)
	if err != nil {
		return err
	}

	LotteryResultsData.Results = results
	log.Println("Refreshed lottery results")
	return nil
}

func ProcessLotteryResults(lotteryList []WebScrape) (map[string]map[string][]string, error) {
	results := make(map[string]map[string][]string)
	resultChan := make(chan struct {
		lotteryName string
		data        map[string][]string
		err         error
	}, len(lotteryList))

	for _, lottery := range lotteryList {
		go func(lottery WebScrape) {
			data, err := ProcessLottery(lottery)
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

func ProcessLottery(lottery WebScrape) (map[string][]string, error) {
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

	return ParseLotteryNumbers(text), nil
}

func GetLotteryList() ([]WebScrape, error) {
	var datas []WebScrape
	c := colly.NewCollector(colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"))

	c.OnHTML("tr", func(e *colly.HTMLElement) {
		href := e.ChildAttr("td a", "href")
		text := e.ChildText("td:first-child")
		text2 := e.ChildText("td:nth-child(2)")
		if text != "" {
			datas = append(datas, WebScrape{LotteryName: text, LotteryDate: text2, PdfLink: href})
		}
	})

	c.Visit("https://statelottery.kerala.gov.in/index.php/lottery-result-view")
	return datas, nil
}

func ParseLotteryNumbers(input string) map[string][]string {
	result := make(map[string][]string)
	parts := strings.Split(input, "<")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		pos, numbersPart := ParsePositionAndNumbersPart(part)
		AddSeriesMatches(result, pos, numbersPart)
		AddAlphanumericMatches(result, pos, numbersPart)
		AddNumericMatches(result, pos, numbersPart)
	}

	return result
}

func ParsePositionAndNumbersPart(part string) (string, string) {
	pos := strings.TrimSpace(strings.Split(part, ">")[0])
	numbersPart := strings.TrimSpace(strings.SplitN(part, ">", 2)[1])
	return pos, numbersPart
}

func AddSeriesMatches(result map[string][]string, pos, numbersPart string) {
	seriesMatches := seriesRegex.FindAllStringSubmatch(numbersPart, -1)
	for _, match := range seriesMatches {
		result[pos] = append(result[pos], match[1])
	}
}

func AddAlphanumericMatches(result map[string][]string, pos, numbersPart string) {
	alphanumericMatches := alphanumericRegex.FindAllStringSubmatch(numbersPart, -1)
	for _, match := range alphanumericMatches {
		result[pos] = append(result[pos], match[1])
	}
}

func AddNumericMatches(result map[string][]string, pos, numbersPart string) {
	numbersPart = alphanumericRegex.ReplaceAllString(numbersPart, "")
	numbers := numbersRegex.FindAllString(numbersPart, -1)
	for _, num := range numbers {
		for i := 0; i < len(num); i += 4 {
			end := i + 4
			if end > len(num) {
				end = len(num)
			}
			result[pos] = append(result[pos], num[i:end])
		}
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
	patternsToRemove := []string{headerPattern, footerPattern, bulletPattern, EndFooterPattern, trailingWhiteSpacePattern, locationString, podiumSplit}
	//temp log
	log.Println(input)
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
	input = regexp.MustCompile(prizeString).ReplaceAllString(input,`Prize Rs`)
	input = regexp.MustCompile(`(Prize Rs :)`).ReplaceAllString(input,`Prize Rs`)
	input = regexp.MustCompile(prizePositionString).ReplaceAllString(input, ` < $0 > `)
	input = regexp.MustCompile(lotteryTicketFull).ReplaceAllString(input, "[$0]")

	seriesMatches := regexp.MustCompile(seriesSelection).FindAllStringSubmatch(input, -1)
	if len(seriesMatches) > 0 {
		series := seriesMatches[0][1]
		input = fmt.Sprintf(`< Series > [%s] %s`, series, input)
	}
	return input, nil
}

// CheckWinningTickets checks if any of the provided tickets are winners
func CheckWinningTickets(results map[string][]string, tickets []string) map[string][]string {
	winners := make(map[string][]string)

	for position, numbers := range results {
		for _, number := range numbers {
			for _, ticket := range tickets {
				if ticket == number {
					winners[position] = append(winners[position], ticket)
				}
			}
		}
	}

	return winners
}
