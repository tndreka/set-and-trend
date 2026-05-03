package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"set-and-trend/backend/internal/rules"
)

// TelegramAlerter pushes setup signals to a Telegram chat.
// Configuration via env: TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID,
// SAF_ACCOUNT_BALANCE_USD (defaults to 1000).
type TelegramAlerter struct {
	BotToken       string
	ChatID         string
	AccountBalance decimal.Decimal
	Client         *http.Client
}

// NewTelegramAlerterFromEnv returns nil if env is missing — caller treats
// nil as "alerts disabled" and continues without sending.
func NewTelegramAlerterFromEnv() *TelegramAlerter {
	tok := os.Getenv("TELEGRAM_BOT_TOKEN")
	chat := os.Getenv("TELEGRAM_CHAT_ID")
	if tok == "" || chat == "" {
		log.Warn().Msg("telegram alerter disabled — set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID to enable")
		return nil
	}
	bal := decimal.NewFromInt(1000)
	if v := os.Getenv("SAF_ACCOUNT_BALANCE_USD"); v != "" {
		if d, err := decimal.NewFromString(v); err == nil && d.IsPositive() {
			bal = d
		}
	}
	return &TelegramAlerter{
		BotToken:       tok,
		ChatID:         chat,
		AccountBalance: bal,
		Client:         &http.Client{Timeout: 10 * time.Second},
	}
}

// FormatSetup builds the alert text for a setup. score-scaled risk:
// s=8 → 1.5%, s=9 → 2.5%, s>=10 → 4.0%. Falls back to 1% if score unknown.
func (a *TelegramAlerter) FormatSetup(symbol string, setup *rules.Setup) string {
	riskPct := riskForScore(setup.ChecklistScore)
	riskUSD := a.AccountBalance.Mul(riskPct).Div(decimal.NewFromInt(100))
	lots := computeLots(symbol, setup.Entry, setup.StopLoss, riskUSD)

	dirEmoji := "🟢"
	if setup.Direction == "SHORT" {
		dirEmoji = "🔴"
	}

	scoreLabel := ""
	if setup.ChecklistScore > 0 {
		scoreLabel = fmt.Sprintf(" · score %d", setup.ChecklistScore)
	}

	return fmt.Sprintf(
		"%s *%s %s*%s\n"+
			"`Entry:` %s\n"+
			"`SL:   ` %s\n"+
			"`TP:   ` %s\n"+
			"`RR:   ` %s:1\n"+
			"`Risk: ` %s%% ($%s) → *%s lots*",
		dirEmoji, symbol, setup.Direction, scoreLabel,
		formatPrice(symbol, setup.Entry),
		formatPrice(symbol, setup.StopLoss),
		formatPrice(symbol, setup.TakeProfit),
		setup.RR.StringFixed(2),
		riskPct.StringFixed(1), riskUSD.StringFixed(0), lots.StringFixed(2),
	)
}

// Send posts a message to the configured Telegram chat. Errors are logged
// but not returned — alerter failures never block signal persistence.
func (a *TelegramAlerter) Send(ctx context.Context, text string) {
	if a == nil {
		return
	}
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", a.BotToken)
	body := url.Values{}
	body.Set("chat_id", a.ChatID)
	body.Set("text", text)
	body.Set("parse_mode", "Markdown")
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBufferString(body.Encode()))
	if err != nil {
		log.Error().Err(err).Msg("telegram: build request failed")
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.Client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("telegram: send failed")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		log.Error().Int("status", resp.StatusCode).Interface("response", payload).Msg("telegram: non-2xx")
	}
}

// riskForScore maps checklist score to risk %. Validated regime "Standard
// scaled" — see iteration analysis in vault for justification.
func riskForScore(score int) decimal.Decimal {
	switch {
	case score >= 10:
		return decimal.NewFromFloat(4.0)
	case score == 9:
		return decimal.NewFromFloat(2.5)
	case score == 8:
		return decimal.NewFromFloat(1.5)
	default:
		return decimal.NewFromInt(1) // safety fallback for non-checklist strategies
	}
}

// pipSizeForSymbol returns price-per-pip for a forex/metal symbol.
// JPY pairs: 0.01. XAUUSD: 0.1. Everything else: 0.0001.
func pipSizeForSymbol(symbol string) decimal.Decimal {
	if len(symbol) >= 6 && (symbol[3:6] == "JPY") {
		return decimal.NewFromFloat(0.01)
	}
	if symbol == "XAUUSD" {
		return decimal.NewFromFloat(0.1)
	}
	return decimal.NewFromFloat(0.0001)
}

// pipValuePerLotUSD returns the USD value of 1 pip on 1.00 standard lot for
// a given symbol with USD-denominated account. This is approximate — exact
// values depend on cross rates, but for the listed pairs the error is <2%.
func pipValuePerLotUSD(symbol string) decimal.Decimal {
	// USD quote currency: pip value = 10 USD per standard lot.
	// JPY quote: pip = 0.01 → ~1000 JPY per lot ≈ varies by USDJPY rate.
	// Approximations chosen to be conservative (slight under-sizing).
	switch symbol {
	case "EURUSD", "GBPUSD", "AUDUSD", "NZDUSD":
		return decimal.NewFromInt(10)
	case "USDJPY", "USDCAD", "USDCHF":
		return decimal.NewFromInt(10) // approximate; live conversion would refine
	case "EURJPY", "GBPJPY", "AUDJPY", "CADJPY", "CHFJPY", "NZDJPY":
		return decimal.NewFromInt(9)
	case "EURGBP", "EURAUD", "EURCAD", "EURCHF", "GBPCHF", "GBPAUD", "AUDCHF":
		return decimal.NewFromInt(10)
	case "XAUUSD":
		return decimal.NewFromInt(10) // gold: lot=100oz, pipSize=0.1 → $10 per pip per standard lot
	}
	return decimal.NewFromInt(10)
}

// computeLots returns standard-lot size given entry, SL, and dollar risk budget.
// formula: lots = riskUSD / (slPips * pipValuePerLot)
func computeLots(symbol string, entry, sl, riskUSD decimal.Decimal) decimal.Decimal {
	risk := entry.Sub(sl).Abs()
	if risk.IsZero() {
		return decimal.Zero
	}
	pipSize := pipSizeForSymbol(symbol)
	pipVal := pipValuePerLotUSD(symbol)
	if pipSize.IsZero() || pipVal.IsZero() {
		return decimal.Zero
	}
	slPips := risk.Div(pipSize)
	if slPips.IsZero() {
		return decimal.Zero
	}
	// 4 decimal places of intermediate precision is plenty for lot math.
	lots := riskUSD.Div(slPips.Mul(pipVal))
	// Round down to broker minimum granularity (0.01).
	rounded := lots.Truncate(2)
	min := decimal.NewFromFloat(0.01)
	if rounded.LessThan(min) {
		return min
	}
	return rounded
}

// formatPrice renders a price with the right number of decimals per symbol.
func formatPrice(symbol string, p decimal.Decimal) string {
	if len(symbol) >= 6 && symbol[3:6] == "JPY" {
		return p.StringFixed(3)
	}
	if symbol == "XAUUSD" {
		return p.StringFixed(2)
	}
	return p.StringFixed(5)
}
