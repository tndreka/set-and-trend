package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"set-and-trend/backend/internal/patterns"

	"github.com/gorilla/websocket"
)

// WebSocketFeeder - COMPLETE broker integration
type WebSocketFeeder struct {
	symbol        string
	ws            *websocket.Conn
	feeder        chan patterns.Candle
	heartbeat     *time.Ticker
	reconnects    int
	maxReconnects int
	wsURL         string
	apiKey        string
	apiSecret     string
	mu            sync.RWMutex
	closed        bool
}

// NewWebSocketFeeder - Generic broker WebSocket
func NewWebSocketFeeder(symbol string, wsURL string, apiKey, apiSecret string) (*WebSocketFeeder, error) {
	feeder := &WebSocketFeeder{
		symbol:        symbol,
		feeder:        make(chan patterns.Candle, 100),
		heartbeat:     time.NewTicker(30 * time.Second),
		maxReconnects: 5,
		wsURL:         wsURL,
		apiKey:        apiKey,
		apiSecret:     apiSecret,
	}

	err := feeder.connect(wsURL, apiKey, apiSecret)
	if err != nil {
		return nil, err
	}

	go feeder.listen()
	return feeder, nil
}

func (f *WebSocketFeeder) connect(wsURL, apiKey, apiSecret string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ws != nil {
		f.ws.Close()
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{}
	if apiKey != "" {
		headers.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
		headers.Set("X-API-KEY", apiKey)
	}

	ws, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		log.Printf("[WS] Connect failed: %v", err)
		return err
	}

	// Subscribe to candles
	subscribeMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": []string{fmt.Sprintf("%s@kline_1h", f.symbol)}, // 1H candles
		"id":     1,
	}

	if err := ws.WriteJSON(subscribeMsg); err != nil {
		ws.Close()
		return err
	}

	f.ws = ws
	log.Printf("[WS] Connected to %s", f.symbol)
	return nil
}

func (f *WebSocketFeeder) listen() {
	defer f.heartbeat.Stop()

	for {
		f.mu.RLock()
		closed := f.closed
		f.mu.RUnlock()

		if closed {
			return
		}

		select {
		case <-f.heartbeat.C:
			f.mu.RLock()
			ws := f.ws
			f.mu.RUnlock()
			if ws != nil {
				ws.WriteMessage(websocket.PingMessage, nil)
			}
		default:
			f.mu.RLock()
			ws := f.ws
			f.mu.RUnlock()

			if ws == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Set read deadline to allow heartbeat checks
			ws.SetReadDeadline(time.Now().Add(35 * time.Second))

			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				log.Printf("[WS] Read error: %v", err)
				f.reconnect()
				continue
			}

			if msgType == websocket.CloseMessage {
				log.Printf("[WS] Connection closed")
				f.reconnect()
				continue
			}

			candle := f.parseCandle(msg)
			if candle.Timestamp != (time.Time{}) {
				select {
				case f.feeder <- candle:
				default:
					// Drop oldest
					select {
					case <-f.feeder:
					default:
					}
					f.feeder <- candle
				}
			}
		}
	}
}

func (f *WebSocketFeeder) parseCandle(rawMsg []byte) patterns.Candle {
	// Binance/MT5/OANDA format parser
	var data map[string]interface{}
	if err := json.Unmarshal(rawMsg, &data); err != nil {
		return patterns.Candle{}
	}

	// Handle different broker formats

	// Binance Kline format
	if kline, ok := data["k"].(map[string]interface{}); ok {
		return parseBinanceKline(kline)
	}

	// MT5/OANDA generic OHLC format
	if hasOHLC(data) {
		return parseGenericOHLC(data)
	}

	// Nested data format (some brokers)
	if d, ok := data["data"].(map[string]interface{}); ok {
		if hasOHLC(d) {
			return parseGenericOHLC(d)
		}
	}

	return patterns.Candle{}
}

func parseBinanceKline(kline map[string]interface{}) patterns.Candle {
	ts, _ := kline["t"].(float64)
	open := parseFloat(kline["o"])
	high := parseFloat(kline["h"])
	low := parseFloat(kline["l"])
	close := parseFloat(kline["c"])
	volume := parseInt64(kline["v"])

	return patterns.Candle{
		Timestamp: time.Unix(int64(ts/1000), 0),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}
}

func parseGenericOHLC(data map[string]interface{}) patterns.Candle {
	var ts time.Time

	// Try different timestamp formats
	if t, ok := data["timestamp"].(string); ok {
		ts, _ = time.Parse(time.RFC3339, t)
	} else if t, ok := data["time"].(float64); ok {
		ts = time.Unix(int64(t), 0)
	} else if t, ok := data["t"].(float64); ok {
		ts = time.Unix(int64(t), 0)
	} else {
		ts = time.Now()
	}

	return patterns.Candle{
		Timestamp: ts,
		Open:      parseFloat(data["open"]),
		High:      parseFloat(data["high"]),
		Low:       parseFloat(data["low"]),
		Close:     parseFloat(data["close"]),
		Volume:    parseInt64(data["volume"]),
	}
}

func hasOHLC(data map[string]interface{}) bool {
	_, hasOpen := data["open"]
	_, hasHigh := data["high"]
	_, hasLow := data["low"]
	_, hasClose := data["close"]
	return hasOpen && hasHigh && hasLow && hasClose
}

func parseFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}

func parseInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case string:
		var i int64
		fmt.Sscanf(val, "%d", &i)
		return i
	case int64:
		return val
	case int:
		return int64(val)
	default:
		return 0
	}
}

func (f *WebSocketFeeder) reconnect() {
	f.mu.Lock()
	if f.closed || f.reconnects >= f.maxReconnects {
		f.mu.Unlock()
		if f.reconnects >= f.maxReconnects {
			log.Printf("[WS] Max reconnects reached for %s", f.symbol)
		}
		return
	}

	f.reconnects++
	reconnectNum := f.reconnects
	wsURL := f.wsURL
	apiKey := f.apiKey
	apiSecret := f.apiSecret
	f.mu.Unlock()

	log.Printf("[WS] Reconnect attempt %d/%d for %s", reconnectNum, f.maxReconnects, f.symbol)

	// Exponential backoff
	time.Sleep(time.Duration(reconnectNum*2) * time.Second)

	if err := f.connect(wsURL, apiKey, apiSecret); err != nil {
		log.Printf("[WS] Reconnect failed: %v", err)
	} else {
		f.mu.Lock()
		f.reconnects = 0
		f.mu.Unlock()
		log.Printf("[WS] Reconnected successfully to %s", f.symbol)
	}
}

// ================================================================================
// CandleFeeder interface implementation
// ================================================================================

// FeedCandle manually injects a candle (for testing/simulation)
func (f *WebSocketFeeder) FeedCandle(candle patterns.Candle) {
	select {
	case f.feeder <- candle:
	default:
		select {
		case <-f.feeder:
		default:
		}
		f.feeder <- candle
	}
}

// NextCandle blocks until a candle is available
func (f *WebSocketFeeder) NextCandle(ctx context.Context) (patterns.Candle, error) {
	select {
	case <-ctx.Done():
		return patterns.Candle{}, ctx.Err()
	case candle, ok := <-f.feeder:
		if !ok {
			return patterns.Candle{}, fmt.Errorf("feeder closed")
		}
		return candle, nil
	}
}

// Close closes the WebSocket connection and feeder
func (f *WebSocketFeeder) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return
	}

	f.closed = true
	if f.ws != nil {
		f.ws.Close()
		f.ws = nil
	}
	close(f.feeder)
	log.Printf("[WS] Closed feeder for %s", f.symbol)
}

// ================================================================================
// BROKER-SPECIFIC FACTORY FUNCTIONS
// ================================================================================

// NewBinanceFeeder creates a Binance WebSocket feeder
func NewBinanceFeeder(symbol string) (*WebSocketFeeder, error) {
	// Binance uses lowercase symbols
	wsURL := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s@kline_1h", symbol)
	return NewWebSocketFeeder(symbol, wsURL, "", "")
}

// NewMT5Feeder creates an MT5 WebSocket feeder (via bridge)
// MT5 doesn't have native WebSocket - requires a bridge server
func NewMT5Feeder(symbol string, bridgeURL string, login, password string) (*WebSocketFeeder, error) {
	wsURL := fmt.Sprintf("%s/ws/%s", bridgeURL, symbol)
	return NewWebSocketFeeder(symbol, wsURL, login, password)
}

// NewOANDAFeeder creates an OANDA WebSocket feeder
func NewOANDAFeeder(symbol string, accountID, apiKey string) (*WebSocketFeeder, error) {
	wsURL := fmt.Sprintf("wss://stream-fxtrade.oanda.com/v3/accounts/%s/pricing/stream?instruments=%s", accountID, symbol)
	return NewWebSocketFeeder(symbol, wsURL, apiKey, "")
}

// ================================================================================
// LIVE TRADING LAUNCHER
// ================================================================================

// StartLiveTrading launches the engine with WebSocket data
func StartLiveTrading(ctx context.Context, symbol string, wsFeeder *WebSocketFeeder, config EngineConfig) (*TradingEngine, error) {
	engine := NewTradingEngine(symbol, config)

	// Create WebSocket data source
	dataSource := NewWebSocketDataSource(symbol, wsFeeder)

	// Set up signal handler
	engine.SetSignalCallback(func(signal *patterns.TradeSignal) {
		log.Printf("[LIVE SIGNAL] %s %s @ %.5f | SL: %.5f | TP: %.5f | Conf: %.1f%%",
			signal.Direction, signal.Symbol, signal.EntryPrice,
			signal.StopLoss, signal.TakeProfit, signal.Confidence*100)

		// TODO: Send to MT5 executor or webhook
	})

	// Start engine in background
	go func() {
		if err := engine.Start(ctx, dataSource); err != nil {
			log.Printf("[ENGINE] Stopped: %v", err)
		}
	}()

	log.Printf("[LIVE] Started trading engine for %s", symbol)
	return engine, nil
}

// ================================================================================
// MULTI-SYMBOL MANAGER
// ================================================================================

// MultiSymbolManager manages multiple live trading engines
type MultiSymbolManager struct {
	engines map[string]*TradingEngine
	feeders map[string]*WebSocketFeeder
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewMultiSymbolManager creates a manager for multiple symbols
func NewMultiSymbolManager() *MultiSymbolManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &MultiSymbolManager{
		engines: make(map[string]*TradingEngine),
		feeders: make(map[string]*WebSocketFeeder),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// AddSymbol adds a symbol with its WebSocket feeder
func (m *MultiSymbolManager) AddSymbol(symbol string, feeder *WebSocketFeeder, config EngineConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.engines[symbol]; exists {
		return fmt.Errorf("symbol %s already registered", symbol)
	}

	engine, err := StartLiveTrading(m.ctx, symbol, feeder, config)
	if err != nil {
		return err
	}

	m.engines[symbol] = engine
	m.feeders[symbol] = feeder
	return nil
}

// RemoveSymbol stops and removes a symbol
func (m *MultiSymbolManager) RemoveSymbol(symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if engine, ok := m.engines[symbol]; ok {
		engine.Stop()
		delete(m.engines, symbol)
	}

	if feeder, ok := m.feeders[symbol]; ok {
		feeder.Close()
		delete(m.feeders, symbol)
	}
}

// GetEngine returns the engine for a symbol
func (m *MultiSymbolManager) GetEngine(symbol string) *TradingEngine {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.engines[symbol]
}

// GetAllPositions returns positions from all engines
func (m *MultiSymbolManager) GetAllPositions() map[string][]*Position {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]*Position)
	for symbol, engine := range m.engines {
		result[symbol] = engine.GetPositions()
	}
	return result
}

// StopAll stops all engines and feeders
func (m *MultiSymbolManager) StopAll() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for symbol, engine := range m.engines {
		engine.Stop()
		log.Printf("[MANAGER] Stopped engine for %s", symbol)
	}

	for symbol, feeder := range m.feeders {
		feeder.Close()
		log.Printf("[MANAGER] Closed feeder for %s", symbol)
	}

	m.engines = make(map[string]*TradingEngine)
	m.feeders = make(map[string]*WebSocketFeeder)
}
