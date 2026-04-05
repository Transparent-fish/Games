package main

import (
	"math/rand"
	"time"
)

// ─── 数据结构 ─────────────────────────────────────────

type Card struct {
	ID    int    `json:"id"`
	Color string `json:"color"` // Red Yellow Blue Green Black
	Val   string `json:"val"`   // 0-9 Skip Reverse +2 Wild +4
}

type Player struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cards     []Card `json:"cards,omitempty"`     // 只在发给本人时填充
	CardCount int    `json:"cardCount"`           // 所有人可见
	IsHost    bool   `json:"isHost,omitempty"`    // 房主标记
	Online    bool   `json:"online"`              // 是否在线
}

type Game struct {
	Players     []*Player `json:"players"`
	NowIdx      int       `json:"nowIdx"`
	Direction   int       `json:"direction"`   // 1 顺时针 -1 逆时针
	TopCard     Card      `json:"topCard"`
	ChosenColor string    `json:"chosenColor"` // Wild 牌选定颜色, 空串表示按 TopCard 颜色
	Status      string    `json:"status"`      // waiting / playing / finished
	Winner      string    `json:"winner,omitempty"`
	LastAction  string    `json:"lastAction"`
	DrawPile    []Card    `json:"-"` // 摸牌堆（不发给前端）
	DiscardPile []Card    `json:"-"` // 弃牌堆
}

// ─── 初始化 ─────────────────────────────────────────

func BuildDeck() []Card {
	var deck []Card
	id := 1
	colors := []string{"Red", "Yellow", "Blue", "Green"}
	vals := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "Skip", "Reverse", "+2"}

	for _, c := range colors {
		// 每种颜色 1 张 0
		deck = append(deck, Card{ID: id, Color: c, Val: "0"})
		id++
		// 每种颜色每种值 2 张
		for _, v := range vals {
			for i := 0; i < 2; i++ {
				deck = append(deck, Card{ID: id, Color: c, Val: v})
				id++
			}
		}
	}
	// 4 张 Wild, 4 张 +4
	for i := 0; i < 4; i++ {
		deck = append(deck, Card{ID: id, Color: "Black", Val: "Wild"})
		id++
		deck = append(deck, Card{ID: id, Color: "Black", Val: "+4"})
		id++
	}
	return deck
}

func ShuffleDeck(deck []Card) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
}

// InitGame 初始化牌局（玩家已加入后调用）
func InitGame(g *Game) {
	g.DrawPile = BuildDeck()
	g.DiscardPile = nil
	g.Direction = 1
	g.NowIdx = 0
	g.Status = "playing"
	g.Winner = ""
	g.ChosenColor = ""
	g.LastAction = "游戏开始！"

	ShuffleDeck(g.DrawPile)

	// 每人发 7 张
	for _, p := range g.Players {
		p.Cards = make([]Card, 0, 7)
		for i := 0; i < 7; i++ {
			p.Cards = append(p.Cards, g.DrawPile[0])
			g.DrawPile = g.DrawPile[1:]
		}
		p.CardCount = len(p.Cards)
	}

	// 翻出第一张非黑牌作为牌顶
	for i, c := range g.DrawPile {
		if c.Color != "Black" {
			g.TopCard = c
			g.DrawPile = append(g.DrawPile[:i], g.DrawPile[i+1:]...)
			break
		}
	}
}

// ─── 摸牌堆回收 ───────────────────────────────────────

func ensureDrawPile(g *Game) {
	if len(g.DrawPile) > 0 {
		return
	}
	if len(g.DiscardPile) == 0 {
		return
	}
	g.DrawPile = g.DiscardPile
	g.DiscardPile = nil
	ShuffleDeck(g.DrawPile)
}

// ─── 出牌合法性检查 ──────────────────────────────────

func CheckCard(g *Game, p *Player, cardID int) (bool, string) {
	if g.Status != "playing" {
		return false, "游戏未在进行中"
	}
	if g.Players[g.NowIdx].ID != p.ID {
		return false, "还没轮到你出牌"
	}

	// 查找手牌
	var found *Card
	for i := range p.Cards {
		if p.Cards[i].ID == cardID {
			found = &p.Cards[i]
			break
		}
	}
	if found == nil {
		return false, "你没有这张牌"
	}

	// 黑牌任何时候可出
	if found.Color == "Black" {
		return true, ""
	}

	// 当前有效颜色
	effectiveColor := g.TopCard.Color
	if g.ChosenColor != "" {
		effectiveColor = g.ChosenColor
	}

	if found.Color == effectiveColor || found.Val == g.TopCard.Val {
		return true, ""
	}

	return false, "出的牌不合法：颜色或数字必须匹配"
}

// ─── 执行出牌 ─────────────────────────────────────────

func PlayCard(g *Game, p *Player, cardID int, chosenColor string) {
	n := len(g.Players)

	// 从手牌中移除
	var played Card
	for i, c := range p.Cards {
		if c.ID == cardID {
			played = c
			p.Cards = append(p.Cards[:i], p.Cards[i+1:]...)
			break
		}
	}
	p.CardCount = len(p.Cards)

	// 更新牌顶
	g.DiscardPile = append(g.DiscardPile, g.TopCard)
	g.TopCard = played

	// 处理选色
	if played.Color == "Black" {
		if chosenColor == "" {
			chosenColor = "Red" // 默认
		}
		g.ChosenColor = chosenColor
	} else {
		g.ChosenColor = ""
	}

	// 胜利判定
	if len(p.Cards) == 0 {
		g.Status = "finished"
		g.Winner = p.ID
		g.LastAction = p.Name + " 打完所有牌，获胜！🎉"
		return
	}

	// 功能牌处理
	skip := false
	draw := 0
	switch played.Val {
	case "Reverse":
		if n == 2 {
			// 两人时 Reverse = Skip
			skip = true
		} else {
			g.Direction *= -1
		}
	case "Skip":
		skip = true
	case "+2":
		draw = 2
	case "+4":
		draw = 4
	}

	// 前进到下一个玩家
	g.NowIdx = (g.NowIdx + g.Direction + n) % n

	// 如果有加牌或跳过
	if skip || draw > 0 {
		target := g.Players[g.NowIdx]
		if draw > 0 {
			for i := 0; i < draw; i++ {
				ensureDrawPile(g)
				if len(g.DrawPile) > 0 {
					target.Cards = append(target.Cards, g.DrawPile[0])
					g.DrawPile = g.DrawPile[1:]
				}
			}
			target.CardCount = len(target.Cards)
		}
		// 跳过该玩家
		g.NowIdx = (g.NowIdx + g.Direction + n) % n
	}

	// 生成操作日志
	actionName := played.Val
	colorName := played.Color
	if played.Color == "Black" {
		colorName = chosenColor
	}
	g.LastAction = p.Name + " 打出 " + colorName + " " + actionName
}

// ─── 摸牌 ─────────────────────────────────────────────

// DrawCard 摸一张牌，摸牌后自动跳过回合
func DrawCard(g *Game, p *Player) *Card {
	if g.Status != "playing" {
		return nil
	}
	if g.Players[g.NowIdx].ID != p.ID {
		return nil
	}

	ensureDrawPile(g)
	if len(g.DrawPile) == 0 {
		return nil
	}

	drawn := g.DrawPile[0]
	g.DrawPile = g.DrawPile[1:]
	p.Cards = append(p.Cards, drawn)
	p.CardCount = len(p.Cards)

	g.LastAction = p.Name + " 摸了一张牌"

	// 摸牌后自动换人
	n := len(g.Players)
	g.NowIdx = (g.NowIdx + g.Direction + n) % n
	return &drawn
}



// ─── 视图序列化 ───────────────────────────────────────

// GameViewForPlayer 为指定玩家生成视图（隐藏其他人手牌）
type GameView struct {
	Players     []PlayerView `json:"players"`
	NowIdx      int          `json:"nowIdx"`
	Direction   int          `json:"direction"`
	TopCard     Card         `json:"topCard"`
	ChosenColor string       `json:"chosenColor"`
	Status      string       `json:"status"`
	Winner      string       `json:"winner,omitempty"`
	LastAction  string       `json:"lastAction"`
	YourCards   []Card       `json:"yourCards"`
	YourIndex   int          `json:"yourIndex"`
	DrawPileNum int          `json:"drawPileNum"`
}

type PlayerView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CardCount int    `json:"cardCount"`
	IsHost    bool   `json:"isHost,omitempty"`
	Online    bool   `json:"online"`
}

func GameViewForPlayer(g *Game, playerID string) GameView {
	view := GameView{
		NowIdx:      g.NowIdx,
		Direction:   g.Direction,
		TopCard:     g.TopCard,
		ChosenColor: g.ChosenColor,
		Status:      g.Status,
		Winner:      g.Winner,
		LastAction:  g.LastAction,
		DrawPileNum: len(g.DrawPile),
		YourIndex:   -1,
	}

	for i, p := range g.Players {
		pv := PlayerView{
			ID:        p.ID,
			Name:      p.Name,
			CardCount: p.CardCount,
			IsHost:    p.IsHost,
			Online:    p.Online,
		}
		view.Players = append(view.Players, pv)

		if p.ID == playerID {
			view.YourCards = p.Cards
			view.YourIndex = i
		}
	}

	return view
}
