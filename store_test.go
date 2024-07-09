package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

func TestPostgres(t *testing.T) {
	tests := []struct {
		name   string
		db     string
		assets []Asset
	}{
		{
			name: "two asset",
			db:   "postgres",
			assets: []Asset{
				{Asset: "EUR", Amount: 1}, {Asset: "USD", Amount: 2},
			},
		},
		{
			name:   "two asset",
			db:     "mem",
			assets: []Asset{{Asset: "EUR", Amount: 1}, {Asset: "USD", Amount: 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.db+"-"+tt.name, func(t *testing.T) {
			db := storeFactory(t, tt.db)
			username := "admin"

			randomTestUser(t, db)
			pwd := sha256.Sum256([]byte(username))
			_, err := db.SaveUser(username, pwd[:])
			if err != nil {
				t.Fatal(err)
			}

			id, passwordHash, err := db.User(username)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := pwd[:], passwordHash; !bytes.Equal(got, want) {
				t.Errorf("got %x want %x", got, want)
			}

			for _, asset := range tt.assets {
				asset.userID = id
				if err := db.SaveAsset(asset); err != nil {
					t.Fatal(err)
				}
			}
			assets, err := db.Assets(id)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := len(assets), len(tt.assets); got != want {
				t.Errorf("got %v want %v", got, want)
			}
			order := &Order{
				id:        -1, // false value to check if updated
				userID:    id,
				Side:      "SELL",
				AssetPair: "EUR-USD",
				Amount:    1,
				Price:     1,
			}
			err = db.SaveOrder(order)
			if err != nil {
				t.Fatal(err)
			}
			if order.id == -1 {
				t.Errorf("order.id not updated")
			}
			orders, err := db.UserOrders(id)
			if err != nil {
				t.Fatal(err)
			}

			if len(orders) != 1 {
				t.Fatal("no orders for user")
			}

			orders, err = db.PendingOrders("EUR-USD")
			if err != nil {
				t.Fatal(err)
			}
			if len(orders) != 1 {
				t.Fatal("no orders for user")
			}
		})
	}
}

func TestFillOrder(t *testing.T) {
	tests := []struct {
		name     string
		db       string
		assets   [][]Asset
		orders   [][]Order
		expected [][]Asset
	}{
		{
			name: "swap two asset",
			db:   "postgres",
			assets: [][]Asset{
				{{Asset: "EUR", Amount: 100}, {Asset: "USD", Amount: 0}},
				{{Asset: "EUR", Amount: 0}, {Asset: "USD", Amount: 100}},
			},
			orders: [][]Order{
				{{
					Side:      "SELL",
					AssetPair: "EUR-USD",
					Amount:    100,
					Price:     1,
				}},
				{{
					Side:      "BUY",
					AssetPair: "EUR-USD",
					Amount:    100,
					Price:     1,
				}},
			},
			expected: [][]Asset{
				{{Asset: "EUR", Amount: 0}, {Asset: "USD", Amount: 100}},
				{{Asset: "EUR", Amount: 100}, {Asset: "USD", Amount: 0}},
			},
		},
		{
			name: "trade two asset",
			db:   "postgres",
			assets: [][]Asset{
				{{Asset: "EUR", Amount: 100}, {Asset: "USD", Amount: 0}},
				{{Asset: "EUR", Amount: 0}, {Asset: "USD", Amount: 100}},
			},
			orders: [][]Order{
				{{
					Side:      "SELL",
					AssetPair: "EUR-USD",
					Amount:    100,
					Price:     0.5,
				}},
				{{
					Side:      "BUY",
					AssetPair: "EUR-USD",
					Amount:    100,
					Price:     1,
				}},
			},
			expected: [][]Asset{
				{{Asset: "EUR", Amount: 0}, {Asset: "USD", Amount: 50}},
				{{Asset: "EUR", Amount: 100}, {Asset: "USD", Amount: 50}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := storeFactory(t, tt.db)

			var users []int
			for i, asset := range tt.assets {
				_, id := randomTestUser(t, db, asset...)
				for _, order := range tt.orders[i] {
					order.userID = id
					if err := db.SaveOrder(&order); err != nil {
						t.Fatal(err)
					}

				}
				orders, err := db.UserOrders(id)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := len(orders), len(tt.orders[i]); got != want {
					t.Errorf("got %v want %v", got, want)
				}

				users = append(users, id)
			}

			orders, err := db.PendingOrders("EUR-USD")
			if err != nil {
				t.Fatal(err)
			}
			if len(orders) == 0 {
				t.Fatal("no pending orders")
			}
			for _, order := range orders {
				if err := db.FillOrder(order); err != nil {
					t.Fatal(err)
				}
			}
			for i, user := range users {
				asset, err := db.Assets(user)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := asset, tt.expected[i]; reflect.DeepEqual(got, want) {
					t.Errorf("got %v want %v", got, want)
				}
			}
		})
	}
}

func storeFactory(t *testing.T, storeType string) store {
	switch storeType {
	case "postgres":
		return newPostgresTest(t)
	case "mem":
		return newMem()
	}
	t.Fatalf("unknown store type %s", storeType)
	return nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// randomTestUser create a random username and save it to the store. The username and the password are equal.
func randomTestUser(t *testing.T, db store, assets ...Asset) (string, int) {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	user := string(b)
	pwd := sha256.Sum256([]byte(user))
	id, err := db.SaveUser(user, pwd[:])
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range assets {
		asset.userID = id
		err := db.SaveAsset(asset)
		if err != nil {
			t.Fatal(err)
		}
	}
	return user, id
}

func newPostgresTest(t *testing.T) store {
	t.Helper()
	if _, exists := os.LookupEnv("DB_URL"); !exists {
		t.Skip("DB_URL not configured")
	}
	db, err := newPostgres(os.Getenv("DB_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.pool.Exec(context.Background(), "TRUNCATE users CASCADE"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.pool.Exec(context.Background(), "TRUNCATE assets CASCADE"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.pool.Exec(context.Background(), "TRUNCATE orders CASCADE"); err != nil {
			t.Fatal(err)
		}
		db.Close()
	})
	return db
}
