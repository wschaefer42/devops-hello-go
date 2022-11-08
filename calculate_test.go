package main

import (
	"encoding/json"
	"fmt"
	"github.com/cucumber/godog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
)

var (
	a, b, result int
)

func iHaveToNumbersAnd(arg1, arg2 int) error {
	a = arg1
	b = arg2
	return nil
}

func iReceiveAsAResult(arg1 int) error {
	if arg1 != result {
		_ = fmt.Errorf("calculate Feature Expected %d but got %d", arg1, result)
	}
	return nil
}

func theCalculatorSumsThem() error {
	url := fmt.Sprintf("%s/api/add?a=%d&b=%d", getEnv("CALC_URL", "http://localhost:8002"), a, b)
	log.Debug().Msgf("Calling URL %s", url)
	res, err := http.Get(url)
	if err != nil {
		log.Fatal().Msgf("Request to %s failed with %s", url, err)
	}
	defer func() { _ = res.Body.Close() }()
	data, _ := io.ReadAll(res.Body)
	log.Debug().Msgf("Received %s", string(data))
	var sum = struct {
		Sum int `json:"sum"`
	}{}
	if err = json.Unmarshal(data, &sum); err != nil {
		log.Fatal().Msgf("Unmarshal failed with %s", err)
	}
	log.Debug().Msgf("Get %v from %s", sum, url)
	result = sum.Sum
	return nil
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^I have to numbers: (\d+) and (\d+)$`, iHaveToNumbersAnd)
	ctx.Step(`^I receive (\d+) as a result$`, iReceiveAsAResult)
	ctx.Step(`^the calculator sums them$`, theCalculatorSumsThem)
}
