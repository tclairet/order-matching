package main

import (
	"crypto/sha256"
	"log/slog"
)

func seed(db store) {
	seeds := []struct {
		name     string
		password string
		assets   []Asset
		orders   []Order
	}{
		{name: "admin", password: "admin", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "EUR", Amount: 0}}},
		{name: "1", password: "1", assets: []Asset{{Asset: "EUR", Amount: 300}, {Asset: "USD", Amount: 0}}, orders: []Order{{
			Side:      "BUY",
			AssetPair: "EUR-USD",
			Amount:    100,
			Price:     2,
		}}},
		{name: "2", password: "2", assets: []Asset{{Asset: "EUR", Amount: 0}, {Asset: "USD", Amount: 300}}, orders: []Order{{
			Side:      "SELL",
			AssetPair: "EUR-USD",
			Amount:    100,
			Price:     2,
		}}},
		{name: "3", password: "3", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "4", password: "4", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "5", password: "5", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "6", password: "6", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "7", password: "6", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "8", password: "6", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "9", password: "6", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "10", password: "6", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "user", password: "password", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "user1", password: "password1", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
		{name: "user2", password: "password2", assets: []Asset{{Asset: "EUR", Amount: 10000}, {Asset: "USD", Amount: 10000}}},
	}
	if _, pwd, _ := db.User("admin"); len(pwd) != 0 {
		slog.Info("db is already seeded")
		return
	}
	slog.Info("seed db")
	for _, s := range seeds {
		pwd := sha256.Sum256([]byte(s.password))
		id, err := db.SaveUser(s.name, pwd[:])
		if err != nil {
			panic(err)
		}
		for _, asset := range s.assets {
			asset.userID = id
			err := db.SaveAsset(asset)
			if err != nil {
				panic(err)
			}
		}
		for _, order := range s.orders {
			order.userID = id
			err := db.SaveOrder(&order)
			if err != nil {
				panic(err)
			}
		}
	}
}
