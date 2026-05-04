package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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

	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	monthStart := now.Format("2006-01") + "-01"

	// 昨日のコスト
	yesterdayResp, err := ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{Start: aws.String(yesterday), End: aws.String(today)},
		Granularity: types.GranularityDaily,
		Metrics:    []string{"UnblendedCost"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 今月累計
	monthResp, err := ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{Start: aws.String(monthStart), End: aws.String(today)},
		Granularity: types.GranularityMonthly,
		Metrics:    []string{"UnblendedCost"},
	})
	if err != nil {
		log.Fatal(err)
	}

	yesterdayCost := "0.00"
	if len(yesterdayResp.ResultsByTime) > 0 {
		yesterdayCost = *yesterdayResp.ResultsByTime[0].Total["UnblendedCost"].Amount
	}
	monthCost := "0.00"
	if len(monthResp.ResultsByTime) > 0 {
		monthCost = *monthResp.ResultsByTime[0].Total["UnblendedCost"].Amount
	}

	yesterday_f, _ := strconv.ParseFloat(yesterdayCost, 64)
	month_f, _ := strconv.ParseFloat(monthCost, 64)
	msg := fmt.Sprintf("💰 AWS Cost Report (%s)\n昨日: $%.2f\n今月累計: $%.2f", yesterday, yesterday_f, month_f)
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
