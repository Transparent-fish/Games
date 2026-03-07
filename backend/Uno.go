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

	colors := []string{"Red", "Yello", "Blue", "Green"}
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
		p.Cards = x.MuoPai[:7]
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
