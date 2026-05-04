package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("SLACK_WEBHOOK_URL is required")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRetryMaxAttempts(10),
		config.WithRetryMode(aws.RetryModeAdaptive),
	)
	if err != nil {
		log.Fatal(err)
	}
	ce := costexplorer.NewFromConfig(cfg)

	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)
	today := now.Format("2006-01-02")
	monthStart := now.Format("2006-01") + "-01"

	// 今月累計
	monthResp, err := ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod:  &types.DateInterval{Start: aws.String(monthStart), End: aws.String(today)},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []types.GroupDefinition{
			{Type: types.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	var total float64
	type entry struct {
		name   string
		amount float64
	}
	var entries []entry
	if len(monthResp.ResultsByTime) > 0 {
		for _, group := range monthResp.ResultsByTime[0].Groups {
			amount, _ := strconv.ParseFloat(*group.Metrics["UnblendedCost"].Amount, 64)
			if amount > 0 {
				total += amount
				entries = append(entries, entry{group.Keys[0], amount})
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].amount > entries[j].amount })
	var lines []string
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("  %s: $%.2f", e.name, e.amount))
	}
	msg := fmt.Sprintf("💰 AWS Cost Report\n今月累計: $%.2f\n%s", total, strings.Join(lines, "\n"))
	if err := postSlack(webhookURL, msg); err != nil {
		log.Fatal(err)
	}
	log.Println("notified:", msg)
}

func postSlack(url, text string) error {
	body, _ := json.Marshal(map[string]string{"text": text})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned %d", resp.StatusCode)
	}
	return nil
}
