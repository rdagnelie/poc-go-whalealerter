package main

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const whaleIOUrl = "https://api.whale-alert.io/v1/transactions"
const scrapeInterval = 11    // +-60s, limit for the free plan
const screenerHistory = 3590 // +-3600s, maximum history for the free plan

var cursors = make(map[string]string)

func main() {
	varsEnv := envLoader([]string{"WHALEIO_TOKEN", "WHALEIO_SCOPE_CURRENCIES"})
	whaleioToken := varsEnv["WHALEIO_TOKEN"]
	scopeIter := strings.Split(varsEnv["WHALEIO_SCOPE_CURRENCIES"], ",")
	for {
		for i := range scopeIter {
			go whaleIOScrapper(whaleioToken, scopeIter[i])
			time.Sleep(scrapeInterval * time.Second)
		}
	}
}

func envLoader(vars []string) map[string]string {
	appVars := make(map[string]string)
	for i := range vars {
		if value, exists := os.LookupEnv(vars[i]); exists {
			appVars[vars[i]] = value
		} else {
			log.Trace().Msg("Error loading mandatory env variable: " + vars[i])
			os.Exit(1)
		}
	}
	if value, exists := os.LookupEnv("DEBUG"); exists {
		if value == "true" {
			log.Trace().Msg("Loading ENV variables...")
			log.Trace().Msg("[LOADED] WHALEIO Token -> " + appVars["WHALEIO_TOKEN"])
			log.Trace().Msg("[LOADED] WHALEIO Scope currencies -> " + appVars["WHALEIO_SCOPE_CURRENCIES"])
		}
	}
	return appVars
}

func whaleIOScrapper(whaleioToken string, whaleioScope string) {
	log.Trace().Msg("whaleIOScrapper URL from querybuilder: " + queryBuilder(whaleioScope))
	url := queryBuilder(whaleioScope)
	client := http.Client{Timeout: time.Duration(1) * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Warn().Msg("Cannot init Scrapper: http.NewRequest cannot be ignited")
	}
	req.Header.Add("X-WA-API-KEY", whaleioToken)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error %s", err)
	}
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		fmt.Println("Non-OK HTTP status:", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	textBytes := []byte(body)

	jsoned_resp := WhaleIOResponseJSON_struct{}
	jsonErr := json.Unmarshal(textBytes, &jsoned_resp)
	if jsonErr != nil {
		fmt.Printf("error %s", jsonErr)
	}
	screener(textBytes, whaleioScope)
}

func queryBuilder(whaleioScope string) string {
	var count int = screenerHistory
	now := time.Now()
	minused := now.Add(time.Duration(-count) * time.Second)
	timestamp := strconv.FormatInt(minused.Unix(), 10)
	//https://gosamples.dev/concatenate-strings
	//strings.Join([]string{hello, gosamples}, " ")
	url := []string{whaleIOUrl, "?start=", timestamp, "&currency=", whaleioScope, "&min_value=500000"}
	return strings.Join(url, "")
}

func screener(textBytes []byte, whaleioScope string) {
	jsoned_resp := WhaleIOResponseJSON_struct{}
	jsonErr := json.Unmarshal(textBytes, &jsoned_resp)
	if jsonErr != nil {
		fmt.Printf("error %s", jsonErr)
	}
	if jsoned_resp.Result == "success" {
		if jsoned_resp.Cursor != cursors[whaleioScope] {
			var symbol string = ""
			var tot_currency float64 = 0
			var tot_usd float64 = 0
			var count int = 0
			var sourceOwner string = "unknown"
			var sourceType string = "unknown"
			var destOwner string = "unknown"
			var destType string = "unknown"
			//var count int = 0
			for _, asset := range jsoned_resp.Transactions {
				blockchain := asset.Blockchain
				transacType := asset.TransactionType
				sourceOwner = asset.From.Owner
				sourceType = asset.From.OwnerType
				destOwner = asset.To.Owner
				destType = asset.To.OwnerType

				symbol = asset.Symbol
				amount := fmt.Sprint(asset.Amount)
				amountUsd := fmt.Sprint(asset.AmountUsd)
				tot_currency = tot_currency + asset.Amount
				tot_usd = tot_usd + asset.AmountUsd
				count = count + asset.TransactionCount
				log.Trace().Msg("ON " + blockchain + ": " + amount + symbol + "|" + amountUsd + "USD" + "|transacType:" + transacType + " " + sourceType + "/" + sourceOwner + "->" + destType + "/" + destOwner)
			}
			cursors[whaleioScope] = jsoned_resp.Cursor
			log.Trace().Msg("[Whale-Movement-last1h][500K+]" + symbol + ":" + fmt.Sprintf("%.5f", tot_currency) + "|usd:" + fmt.Sprintf("%.1f", tot_usd) + "|trx:" + fmt.Sprint(count))
		}
	}
}

type WhaleIOResponseJSON_struct struct {
	Result       string `json:"result"`
	Cursor       string `json:"cursor"`
	Count        int    `json:"count"`
	Transactions []struct {
		Blockchain      string `json:"blockchain"`
		Symbol          string `json:"symbol"`
		ID              string `json:"id"`
		TransactionType string `json:"transaction_type"`
		Hash            string `json:"hash"`
		From            struct {
			Address   string `json:"address"`
			Owner     string `json:"owner"`
			OwnerType string `json:"owner_type"`
		} `json:"from"`
		To struct {
			Address   string `json:"address"`
			Owner     string `json:"owner"`
			OwnerType string `json:"owner_type"`
		} `json:"to"`
		Timestamp        int     `json:"timestamp"`
		Amount           float64 `json:"amount"`
		AmountUsd        float64 `json:"amount_usd"`
		TransactionCount int     `json:"transaction_count"`
	} `json:"transactions"`
}
