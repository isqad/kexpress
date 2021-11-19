package service

import (
	"math/rand"
	"net/http"
)

const REFERER = "https://kazanexpress.ru/"

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.54 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:93.0) Gecko/20100101 Firefox/93.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.54 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; rv:91.0) Gecko/20100101 Firefox/91.0",
}

func newRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	userAgent := userAgents[rand.Intn(len(userAgents))]
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", REFERER)
	req.Header.Set("Origin", REFERER)
	req.Header.Set("Authorization", "Basic a2F6YW5leHByZXNzLWN1c3RvbWVyOmN1c3RvbWVyU2VjcmV0S2V5")
	req.Header.Set("sec-ch-ua", `;Not A Brand";v="99", "Chromium";v="94`)
	req.Header.Set("sec-ch-ua-mobile", `?0`)
	req.Header.Set("Content-Type", `application/json`)
	req.Header.Set("Accept", `application/json`)

	return req, nil
}
