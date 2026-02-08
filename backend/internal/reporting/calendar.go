package reporting

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type TradeDirection string

const (
	Long  TradeDirection = "LONG"
	Short TradeDirection = "SHORT"
)

type TradeEntry struct {
	ID         string         `json:"id"`
	EntryTime  time.Time      `json:"entry_time"`
	ExitTime   time.Time      `json:"exit_time"`
	Symbol     string         `json:"symbol"`
	Direction  TradeDirection `json:"direction"`
	EntryPrice float64        `json:"entry_price"`
	ExitPrice  float64        `json:"exit_price"`
	StopLoss   float64        `json:"stop_loss"`
	TakeProfit float64        `json:"take_profit"`
	PnL        float64        `json:"pnl"`
	PnLPips    float64        `json:"pnl_pips"`
	Confidence float64        `json:"confidence"`
	Pattern    string         `json:"pattern"`
	IsWinner   bool           `json:"is_winner"`
}

type DayEntry struct {
	Date       time.Time    `json:"date"`
	DayOfWeek  string       `json:"day_of_week"`
	Trades     []TradeEntry `json:"trades"`
	TradeCount int          `json:"trade_count"`
	DailyPnL   float64      `json:"daily_pnl"`
	Winners    int          `json:"winners"`
	Losers     int          `json:"losers"`
}

type WeekStats struct {
	WeekNumber int       `json:"week_number"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	TradeCount int       `json:"trade_count"`
	Winners    int       `json:"winners"`
	Losers     int       `json:"losers"`
	WeeklyPnL  float64   `json:"weekly_pnl"`
	WinRate    float64   `json:"win_rate"`
	AvgPnL     float64   `json:"avg_pnl"`
	BestTrade  float64   `json:"best_trade"`
	WorstTrade float64   `json:"worst_trade"`
}

type MonthlyCalendar struct {
	Year        int         `json:"year"`
	Month       time.Month  `json:"month"`
	Days        []DayEntry  `json:"days"`
	Weeks       []WeekStats `json:"weeks"`
	TradeCount  int         `json:"trade_count"`
	Winners     int         `json:"winners"`
	Losers      int         `json:"losers"`
	MonthlyPnL  float64     `json:"monthly_pnl"`
	WinRate     float64     `json:"win_rate"`
	AvgPnL      float64     `json:"avg_pnl"`
	BestDay     *DayEntry   `json:"best_day"`
	WorstDay    *DayEntry   `json:"worst_day"`
	TradingDays int         `json:"trading_days"`
}

type CalendarReport struct {
	StartDate    time.Time         `json:"start_date"`
	EndDate      time.Time         `json:"end_date"`
	Months       []MonthlyCalendar `json:"months"`
	TotalTrades  int               `json:"total_trades"`
	TotalWinners int               `json:"total_winners"`
	TotalLosers  int               `json:"total_losers"`
	TotalPnL     float64           `json:"total_pnl"`
	WinRate      float64           `json:"win_rate"`
	AvgPnL       float64           `json:"avg_pnl"`
}

type CalendarGenerator struct {
	trades []TradeEntry
}

func NewCalendarGenerator() *CalendarGenerator {
	return &CalendarGenerator{
		trades: make([]TradeEntry, 0),
	}
}

func (g *CalendarGenerator) AddTrade(trade TradeEntry) {
	g.trades = append(g.trades, trade)
}

func (g *CalendarGenerator) AddTrades(trades []TradeEntry) {
	g.trades = append(g.trades, trades...)
}

func (g *CalendarGenerator) GenerateReport() *CalendarReport {
	if len(g.trades) == 0 {
		return &CalendarReport{}
	}

	sort.Slice(g.trades, func(i, j int) bool {
		return g.trades[i].EntryTime.Before(g.trades[j].EntryTime)
	})

	report := &CalendarReport{
		StartDate: g.trades[0].EntryTime,
		EndDate:   g.trades[len(g.trades)-1].EntryTime,
		Months:    make([]MonthlyCalendar, 0),
	}

	monthlyTrades := g.groupByMonth()

	for _, key := range sortedMonthKeys(monthlyTrades) {
		trades := monthlyTrades[key]
		month := g.buildMonthlyCalendar(trades)
		report.Months = append(report.Months, month)

		report.TotalTrades += month.TradeCount
		report.TotalWinners += month.Winners
		report.TotalLosers += month.Losers
		report.TotalPnL += month.MonthlyPnL
	}

	if report.TotalTrades > 0 {
		report.WinRate = float64(report.TotalWinners) / float64(report.TotalTrades) * 100
		report.AvgPnL = report.TotalPnL / float64(report.TotalTrades)
	}

	return report
}

func (g *CalendarGenerator) groupByMonth() map[string][]TradeEntry {
	result := make(map[string][]TradeEntry)
	for _, trade := range g.trades {
		key := trade.EntryTime.Format("2006-01")
		result[key] = append(result[key], trade)
	}
	return result
}

func sortedMonthKeys(m map[string][]TradeEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (g *CalendarGenerator) buildMonthlyCalendar(trades []TradeEntry) MonthlyCalendar {
	if len(trades) == 0 {
		return MonthlyCalendar{}
	}

	firstTrade := trades[0]
	calendar := MonthlyCalendar{
		Year:  firstTrade.EntryTime.Year(),
		Month: firstTrade.EntryTime.Month(),
		Days:  make([]DayEntry, 0),
		Weeks: make([]WeekStats, 0),
	}

	dailyTrades := g.groupByDay(trades)
	weeklyTrades := g.groupByWeek(trades)

	for _, key := range sortedDayKeys(dailyTrades) {
		dayTrades := dailyTrades[key]
		day := g.buildDayEntry(dayTrades)
		calendar.Days = append(calendar.Days, day)

		calendar.TradeCount += day.TradeCount
		calendar.Winners += day.Winners
		calendar.Losers += day.Losers
		calendar.MonthlyPnL += day.DailyPnL

		if day.TradeCount > 0 {
			calendar.TradingDays++
		}

		if calendar.BestDay == nil || day.DailyPnL > calendar.BestDay.DailyPnL {
			dayCopy := day
			calendar.BestDay = &dayCopy
		}
		if calendar.WorstDay == nil || day.DailyPnL < calendar.WorstDay.DailyPnL {
			dayCopy := day
			calendar.WorstDay = &dayCopy
		}
	}

	for _, key := range sortedWeekKeys(weeklyTrades) {
		weekTrades := weeklyTrades[key]
		week := g.buildWeekStats(weekTrades, key)
		calendar.Weeks = append(calendar.Weeks, week)
	}

	if calendar.TradeCount > 0 {
		calendar.WinRate = float64(calendar.Winners) / float64(calendar.TradeCount) * 100
		calendar.AvgPnL = calendar.MonthlyPnL / float64(calendar.TradeCount)
	}

	return calendar
}

func (g *CalendarGenerator) groupByDay(trades []TradeEntry) map[string][]TradeEntry {
	result := make(map[string][]TradeEntry)
	for _, trade := range trades {
		key := trade.EntryTime.Format("2006-01-02")
		result[key] = append(result[key], trade)
	}
	return result
}

func (g *CalendarGenerator) groupByWeek(trades []TradeEntry) map[int][]TradeEntry {
	result := make(map[int][]TradeEntry)
	for _, trade := range trades {
		_, week := trade.EntryTime.ISOWeek()
		result[week] = append(result[week], trade)
	}
	return result
}

func sortedDayKeys(m map[string][]TradeEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedWeekKeys(m map[int][]TradeEntry) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func (g *CalendarGenerator) buildDayEntry(trades []TradeEntry) DayEntry {
	if len(trades) == 0 {
		return DayEntry{}
	}

	day := DayEntry{
		Date:       trades[0].EntryTime.Truncate(24 * time.Hour),
		DayOfWeek:  trades[0].EntryTime.Weekday().String(),
		Trades:     trades,
		TradeCount: len(trades),
	}

	for _, trade := range trades {
		day.DailyPnL += trade.PnL
		if trade.IsWinner {
			day.Winners++
		} else {
			day.Losers++
		}
	}

	return day
}

func (g *CalendarGenerator) buildWeekStats(trades []TradeEntry, weekNum int) WeekStats {
	if len(trades) == 0 {
		return WeekStats{WeekNumber: weekNum}
	}

	stats := WeekStats{
		WeekNumber: weekNum,
		StartDate:  trades[0].EntryTime,
		EndDate:    trades[len(trades)-1].EntryTime,
		TradeCount: len(trades),
	}

	for _, trade := range trades {
		stats.WeeklyPnL += trade.PnL
		if trade.IsWinner {
			stats.Winners++
		} else {
			stats.Losers++
		}

		if trade.PnL > stats.BestTrade {
			stats.BestTrade = trade.PnL
		}
		if trade.PnL < stats.WorstTrade {
			stats.WorstTrade = trade.PnL
		}
	}

	if stats.TradeCount > 0 {
		stats.WinRate = float64(stats.Winners) / float64(stats.TradeCount) * 100
		stats.AvgPnL = stats.WeeklyPnL / float64(stats.TradeCount)
	}

	return stats
}

func FormatCalendar(calendar *MonthlyCalendar) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%s %d Trading Calendar\n", calendar.Month.String(), calendar.Year))
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	sb.WriteString("      Mon    Tue    Wed    Thu    Fri    Sat    Sun\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	firstDay := time.Date(calendar.Year, calendar.Month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	startPadding := int(firstDay.Weekday())
	if startPadding == 0 {
		startPadding = 7
	}
	startPadding--

	dayMap := make(map[string]DayEntry)
	for _, day := range calendar.Days {
		key := day.Date.Format("2006-01-02")
		dayMap[key] = day
	}

	week := strings.Repeat("       ", startPadding)

	for d := 1; d <= lastDay.Day(); d++ {
		date := time.Date(calendar.Year, calendar.Month, d, 0, 0, 0, 0, time.UTC)
		key := date.Format("2006-01-02")

		dayStr := ""
		if entry, ok := dayMap[key]; ok && entry.TradeCount > 0 {
			prefix := "+"
			if entry.DailyPnL < 0 {
				prefix = "-"
			}
			dayStr = fmt.Sprintf("%2d%s%2d", d, prefix, entry.TradeCount)
		} else {
			dayStr = fmt.Sprintf("%2d    ", d)
		}
		week += dayStr + " "

		if date.Weekday() == time.Sunday || d == lastDay.Day() {
			sb.WriteString(week + "\n")
			week = ""
		}
	}

	sb.WriteString("\n" + strings.Repeat("-", 60) + "\n")
	sb.WriteString(fmt.Sprintf("Trades: %d | Winners: %d | Losers: %d | Win Rate: %.1f%%\n",
		calendar.TradeCount, calendar.Winners, calendar.Losers, calendar.WinRate))
	sb.WriteString(fmt.Sprintf("P&L: $%.2f | Avg: $%.2f | Trading Days: %d\n",
		calendar.MonthlyPnL, calendar.AvgPnL, calendar.TradingDays))

	if calendar.BestDay != nil {
		sb.WriteString(fmt.Sprintf("Best Day: %s ($%.2f)\n",
			calendar.BestDay.Date.Format("Jan 02"), calendar.BestDay.DailyPnL))
	}
	if calendar.WorstDay != nil {
		sb.WriteString(fmt.Sprintf("Worst Day: %s ($%.2f)\n",
			calendar.WorstDay.Date.Format("Jan 02"), calendar.WorstDay.DailyPnL))
	}

	return sb.String()
}

func FormatReport(report *CalendarReport) string {
	var sb strings.Builder

	sb.WriteString("\n" + strings.Repeat("=", 70) + "\n")
	sb.WriteString("                    TRADING CALENDAR REPORT\n")
	sb.WriteString(strings.Repeat("=", 70) + "\n")
	sb.WriteString(fmt.Sprintf("Period: %s to %s\n\n",
		report.StartDate.Format("Jan 02, 2006"),
		report.EndDate.Format("Jan 02, 2006")))

	for _, month := range report.Months {
		sb.WriteString(FormatCalendar(&month))
		sb.WriteString("\n")
	}

	sb.WriteString(strings.Repeat("=", 70) + "\n")
	sb.WriteString("                        SUMMARY\n")
	sb.WriteString(strings.Repeat("=", 70) + "\n")
	sb.WriteString(fmt.Sprintf("Total Trades: %d\n", report.TotalTrades))
	sb.WriteString(fmt.Sprintf("Winners: %d | Losers: %d\n", report.TotalWinners, report.TotalLosers))
	sb.WriteString(fmt.Sprintf("Win Rate: %.1f%%\n", report.WinRate))
	sb.WriteString(fmt.Sprintf("Total P&L: $%.2f\n", report.TotalPnL))
	sb.WriteString(fmt.Sprintf("Average P&L: $%.2f\n", report.AvgPnL))
	sb.WriteString(strings.Repeat("=", 70) + "\n")

	return sb.String()
}
