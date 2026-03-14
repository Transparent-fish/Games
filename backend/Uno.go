package main

import (
	"math/rand"
	"time"
)

type Card struct {
	ID    int    `json:"id"` //每张牌的唯一id
	Color string `json:"color"`
	Val   string `json:"val"`
}

type Game struct {
	Players      []*Player `json:"players"`
	NowID        int       `json:"nowID"`
	FangXiang    int       `json:"fangXiang"` //1是顺时针 -1是逆时针
	Cards        []Card    `json:"cards"`
	MuoPai       []Card    `json:"-"`            //摸牌堆
	TopCard      Card      `json:"topCard"`      //最上面的那张牌
	DaDiaoDeCard []Card    `json:"daDiaoDeCard"` //被打掉的牌
}

type Player struct {
	ID    string `json:"ID"`
	Name  string `json:"name"`
	Cards []Card `json:"cards"`
}

func InitGame(x *Game) {
	//删除之前的数据
	x.MuoPai = []Card{}
	x.DaDiaoDeCard = []Card{}
	x.FangXiang = 1 //顺时针
	x.NowID = 0     //第一个人
	x.TopCard = Card{}

	colors := []string{"Red", "Yellow", "Blue", "Green"}
	val := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "Skip", "Reverse", "+2"}
	CardId := 1
	//数字牌
	for _, cl := range colors {
		x.MuoPai = append(x.MuoPai, Card{ID: CardId, Color: cl, Val: "0"})
		CardId++
		for _, vall := range val {
			for i := 1; i <= 2; i++ {
				x.MuoPai = append(x.MuoPai, Card{ID: CardId, Color: cl, Val: vall})
				CardId++
			}
		}
	}
	//+4和转色
	for i := 1; i <= 4; i++ {
		x.MuoPai = append(x.MuoPai, Card{ID: CardId, Color: "Black", Val: "Wild"})
		CardId++
		x.MuoPai = append(x.MuoPai, Card{ID: CardId, Color: "Black", Val: "+4"})
		CardId++
	}
	//洗牌
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(x.MuoPai), func(i, j int) {
		x.MuoPai[i], x.MuoPai[j] = x.MuoPai[j], x.MuoPai[i]
	})
	//发牌
	for _, p := range x.Players {
		p.Cards = append([]Card(nil), x.MuoPai[:7]...)
		x.MuoPai = x.MuoPai[7:]
	}
	//第一张牌去牌顶
	FirstId := 0
	for i, c := range x.MuoPai {
		if c.Color != "Black" {
			FirstId = i
			break
		}
	}
	x.TopCard = x.MuoPai[FirstId]
	x.MuoPai = append(x.MuoPai[:FirstId], x.MuoPai[FirstId+1:]...)
}

func CheckCard(x *Game, p *Player, cardID int) (bool, string) {
	if x.Players[x.NowID].ID != p.ID {
		return false, "没到你出牌"
	}
	var FindCard Card
	IsFind := false
	for _, c := range p.Cards {
		if c.ID == cardID {
			FindCard = c
			IsFind = true
			break
		}
	}
	if !IsFind {
		return false, "你没有这张牌！"
	}
	CanPlay := false
	if FindCard.Color == "Black" ||
		FindCard.Color == x.TopCard.Color ||
		FindCard.Val == x.TopCard.Val {
		CanPlay = true
	}
	if !CanPlay {
		return false, "出的牌不合法"
	}
	return true, "OK"
}

func PlayAction(x *Game, p *Player, CardId int) {
	n := len(x.Players)
	//删牌
	for i, c := range p.Cards {
		if c.ID == CardId {
			x.TopCard = c //现在桌面上的牌更新
			x.DaDiaoDeCard = append(x.DaDiaoDeCard, c)
			p.Cards = append(p.Cards[:i], p.Cards[i+1:]...)
			break
		}
	}
	//+4 +2 转向 转色
	skip := false
	draw := 0
	switch x.TopCard.Val {
	case "Reverse": //转向
		x.FangXiang *= -1
	case "Skip":
		skip = true
	case "+2":
		draw = 2
	case "+4":
		draw = 4
	}
	//换人
	x.NowID = (x.NowID + x.FangXiang + n) % n
	if skip || draw > 0 {
		if draw > 0 {
			for i := 0; i < draw; i++ {
				if len(x.MuoPai) > 0 {
					x.Players[x.NowID].Cards = append(x.Players[x.NowID].Cards, x.MuoPai[0])
					x.MuoPai = x.MuoPai[1:]
				}
			}
		}
		// 如果是 skip 或加牌，也要跳过该玩家的出牌回合
		x.NowID = (x.NowID + x.FangXiang + n) % n
	}
}

func DrawCard(x *Game, p *Player) {
	if x.Players[x.NowID].ID != p.ID {
		return
	}
	if len(x.MuoPai) == 0 {
		// 如果摸牌堆空了，把弃牌堆（不含当前牌顶）洗回去
		recycled := make([]Card, 0, len(x.DaDiaoDeCard))
		for _, c := range x.DaDiaoDeCard {
			if c.ID != x.TopCard.ID {
				recycled = append(recycled, c)
			}
		}
		x.MuoPai = recycled
		x.DaDiaoDeCard = []Card{}
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(x.MuoPai), func(i, j int) {
			x.MuoPai[i], x.MuoPai[j] = x.MuoPai[j], x.MuoPai[i]
		})
	}
	if len(x.MuoPai) > 0 {
		p.Cards = append(p.Cards, x.MuoPai[0])
		x.MuoPai = x.MuoPai[1:]
	}
	// 摸牌后通常也要换人，或者允许出刚摸到的牌（这里简单处理为换人）
	n := len(x.Players)
	x.NowID = (x.NowID + x.FangXiang + n) % n
}
