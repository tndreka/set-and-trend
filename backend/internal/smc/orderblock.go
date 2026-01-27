package smc

import (
	"math"
	"time"

	"set-and-trend/backend/internal/database"
)

// OrderBlock represents an Area of Interest (AOI) where institutional orders accumulate
// Demand Zone (bullish OB): Last bearish candle before strong bullish move
// Supply Zone (bearish OB): Last bullish candle before strong bearish move
type OrderBlock struct {
	Direction int       // 1=demand (bullish), -1=supply (bearish)
	Top       float64   // Upper boundary (high of the candle)
	Bottom    float64   // Lower boundary (low of the candle)
	MidPoint  float64   // Center of the block
	Strength  float64   // Strength based on follow-through move
	Volume    float64   // Volume at the order block
	Tested    bool      // Whether price has returned to test this OB
	Broken    bool      // Whether price has broken through this OB
	Index     int       // Bar index of the order block candle
	Timestamp time.Time // Timestamp of the order block
}

// SwingPoint for order block detection
type SwingPoint struct {
	Index  int
	Price  float64
	IsHigh bool
}

// DetectOrderBlocks finds potential order block zones
// minMovePercent: minimum percentage move to qualify as "strong" impulse
func DetectOrderBlocks(candles []database.Candle, minMovePercent float64) []OrderBlock {
	if len(candles) < 5 {
		return nil
	}

	if minMovePercent <= 0 {
		minMovePercent = 0.005 // 0.5% minimum move
	}

	var orderBlocks []OrderBlock

	for i := 1; i < len(candles)-2; i++ {
		prevCandle := candles[i-1]
		currentCandle := candles[i]
		nextCandle := candles[i+1]
		followCandle := candles[i+2]

		// Check for Demand Zone (bullish order block)
		// Current candle is bearish, followed by strong bullish move
		if isBearishCandle(currentCandle) && isBullishCandle(nextCandle) {
			// Calculate the bullish move strength
			movePercent := (followCandle.High - currentCandle.Low) / currentCandle.Close
			
			// Need previous candle context to confirm institutional selling before reversal
			if movePercent >= minMovePercent && prevCandle.Close > currentCandle.Close {
				orderBlocks = append(orderBlocks, OrderBlock{
					Direction: 1, // Demand zone
					Top:       currentCandle.High,
					Bottom:    currentCandle.Low,
					MidPoint:  (currentCandle.High + currentCandle.Low) / 2,
					Strength:  math.Min(movePercent*100, 5.0), // Cap at 5
					Volume:    currentCandle.Volume,
					Tested:    false,
					Broken:    false,
					Index:     i,
					Timestamp: currentCandle.Timestamp,
				})
			}
		}

		// Check for Supply Zone (bearish order block)
		// Current candle is bullish, followed by strong bearish move
		if isBullishCandle(currentCandle) && isBearishCandle(nextCandle) {
			// Calculate the bearish move strength
			movePercent := (currentCandle.High - followCandle.Low) / currentCandle.Close
			
			// Need previous candle context to confirm institutional buying before reversal
			if movePercent >= minMovePercent && prevCandle.Close < currentCandle.Close {
				orderBlocks = append(orderBlocks, OrderBlock{
					Direction: -1, // Supply zone
					Top:       currentCandle.High,
					Bottom:    currentCandle.Low,
					MidPoint:  (currentCandle.High + currentCandle.Low) / 2,
					Strength:  math.Min(movePercent*100, 5.0),
					Volume:    currentCandle.Volume,
					Tested:    false,
					Broken:    false,
					Index:     i,
					Timestamp: currentCandle.Timestamp,
				})
			}
		}
	}

	return orderBlocks
}

// DetectOrderBlocksWithSwings uses swing points for more accurate OB detection
func DetectOrderBlocksWithSwings(candles []database.Candle, swingHighs, swingLows []SwingPoint) []OrderBlock {
	if len(candles) < 10 {
		return nil
	}

	var orderBlocks []OrderBlock

	// Find demand zones at swing lows
	for _, low := range swingLows {
		if low.Index < 1 || low.Index >= len(candles)-3 {
			continue
		}

		// Find the last bearish candle before the swing low
		obIndex := -1
		for j := low.Index; j >= low.Index-3 && j >= 0; j-- {
			if isBearishCandle(candles[j]) {
				obIndex = j
				break
			}
		}

		if obIndex >= 0 {
			// Check for bullish follow-through
			hasFollowThrough := false
			moveStrength := 0.0
			for k := low.Index + 1; k < len(candles) && k <= low.Index+5; k++ {
				if candles[k].Close > candles[obIndex].High {
					hasFollowThrough = true
					moveStrength = (candles[k].High - candles[obIndex].Low) / candles[obIndex].Close
					break
				}
			}

			if hasFollowThrough && moveStrength > 0.005 {
				orderBlocks = append(orderBlocks, OrderBlock{
					Direction: 1,
					Top:       candles[obIndex].High,
					Bottom:    candles[obIndex].Low,
					MidPoint:  (candles[obIndex].High + candles[obIndex].Low) / 2,
					Strength:  math.Min(moveStrength*100, 5.0),
					Volume:    candles[obIndex].Volume,
					Tested:    false,
					Broken:    false,
					Index:     obIndex,
					Timestamp: candles[obIndex].Timestamp,
				})
			}
		}
	}

	// Find supply zones at swing highs
	for _, high := range swingHighs {
		if high.Index < 1 || high.Index >= len(candles)-3 {
			continue
		}

		// Find the last bullish candle before the swing high
		obIndex := -1
		for j := high.Index; j >= high.Index-3 && j >= 0; j-- {
			if isBullishCandle(candles[j]) {
				obIndex = j
				break
			}
		}

		if obIndex >= 0 {
			// Check for bearish follow-through
			hasFollowThrough := false
			moveStrength := 0.0
			for k := high.Index + 1; k < len(candles) && k <= high.Index+5; k++ {
				if candles[k].Close < candles[obIndex].Low {
					hasFollowThrough = true
					moveStrength = (candles[obIndex].High - candles[k].Low) / candles[obIndex].Close
					break
				}
			}

			if hasFollowThrough && moveStrength > 0.005 {
				orderBlocks = append(orderBlocks, OrderBlock{
					Direction: -1,
					Top:       candles[obIndex].High,
					Bottom:    candles[obIndex].Low,
					MidPoint:  (candles[obIndex].High + candles[obIndex].Low) / 2,
					Strength:  math.Min(moveStrength*100, 5.0),
					Volume:    candles[obIndex].Volume,
					Tested:    false,
					Broken:    false,
					Index:     obIndex,
					Timestamp: candles[obIndex].Timestamp,
				})
			}
		}
	}

	return orderBlocks
}

// UpdateOrderBlockStatus updates tested/broken status based on price action
func UpdateOrderBlockStatus(orderBlocks []OrderBlock, candles []database.Candle, fromIndex int) []OrderBlock {
	for i := range orderBlocks {
		if orderBlocks[i].Broken {
			continue
		}

		startIdx := orderBlocks[i].Index + 1
		if startIdx < fromIndex {
			startIdx = fromIndex
		}

		for j := startIdx; j < len(candles); j++ {
			// Check if price tested the OB
			if !orderBlocks[i].Tested {
				if orderBlocks[i].Direction == 1 {
					// Demand: tested if price retraced into it
					if candles[j].Low <= orderBlocks[i].Top && candles[j].Low >= orderBlocks[i].Bottom {
						orderBlocks[i].Tested = true
					}
				} else {
					// Supply: tested if price retraced into it
					if candles[j].High >= orderBlocks[i].Bottom && candles[j].High <= orderBlocks[i].Top {
						orderBlocks[i].Tested = true
					}
				}
			}

			// Check if OB is broken
			if orderBlocks[i].Direction == 1 {
				// Demand broken if price closes below the OB bottom
				if candles[j].Close < orderBlocks[i].Bottom {
					orderBlocks[i].Broken = true
					break
				}
			} else {
				// Supply broken if price closes above the OB top
				if candles[j].Close > orderBlocks[i].Top {
					orderBlocks[i].Broken = true
					break
				}
			}
		}
	}

	return orderBlocks
}

// GetActiveOrderBlocks returns order blocks that haven't been broken
func GetActiveOrderBlocks(orderBlocks []OrderBlock) []OrderBlock {
	var active []OrderBlock
	for _, ob := range orderBlocks {
		if !ob.Broken {
			active = append(active, ob)
		}
	}
	return active
}

// IsPriceAtOrderBlock checks if price is within an order block zone
func IsPriceAtOrderBlock(price float64, orderBlocks []OrderBlock) *OrderBlock {
	for i := range orderBlocks {
		if orderBlocks[i].Broken {
			continue
		}
		if price >= orderBlocks[i].Bottom && price <= orderBlocks[i].Top {
			return &orderBlocks[i]
		}
	}
	return nil
}

// GetNearestOrderBlock finds the closest active order block to price
func GetNearestOrderBlock(price float64, orderBlocks []OrderBlock, direction int) *OrderBlock {
	var nearest *OrderBlock
	minDistance := float64(1e10)

	for i := range orderBlocks {
		if orderBlocks[i].Broken {
			continue
		}
		
		if direction != 0 && orderBlocks[i].Direction != direction {
			continue
		}

		distance := 0.0
		if price > orderBlocks[i].Top {
			distance = price - orderBlocks[i].Top
		} else if price < orderBlocks[i].Bottom {
			distance = orderBlocks[i].Bottom - price
		}

		if distance < minDistance {
			minDistance = distance
			nearest = &orderBlocks[i]
		}
	}

	return nearest
}

// Helper functions
func isBullishCandle(c database.Candle) bool {
	return c.Close > c.Open
}

func isBearishCandle(c database.Candle) bool {
	return c.Close < c.Open
}
