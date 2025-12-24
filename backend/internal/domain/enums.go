package domain

type TradeBias string

const (
	BiasLong  TradeBias = "long"
	BiasShort TradeBias = "short"
)

func (b TradeBias) IsValid() bool {
	return b == BiasLong || b == BiasShort
}

type TradeResult string

const (
	ResultWin       TradeResult = "win"
	ResultLoss      TradeResult = "loss"
	ResultBreakeven TradeResult = "breakeven"
)

func (r TradeResult) IsValid() bool {
	switch r {
	case ResultWin, ResultLoss, ResultBreakeven:
		return true
	default:
		return false
	}
}

type Session string

const (
	SessionLondon   Session = "london"
	SessionNewYork  Session = "new_york"
	SessionAsian    Session = "asian"
	SessionCustom   Session = "custom"
)

func (s Session) IsValid() bool {
	switch s {
	case SessionLondon, SessionNewYork, SessionAsian, SessionCustom:
		return true
	default:
		return false
	}
}

type Emotion string

const (
	EmotionCalm    Emotion = "calm"
	EmotionAnxious Emotion = "anxious"
	EmotionFOMO    Emotion = "fomo"
	EmotionRevenge Emotion = "revenge"
	EmotionOther   Emotion = "other"
)

func (e Emotion) IsValid() bool {
	switch e {
	case EmotionCalm, EmotionAnxious, EmotionFOMO, EmotionRevenge, EmotionOther:
		return true
	default:
		return false
	}
}
