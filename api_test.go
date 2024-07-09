package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func Test_api(t *testing.T) {
	tests := []struct {
		name   string
		db     string
		assets []Asset
		orders []Order
	}{
		{
			name:   "mem",
			db:     "mem",
			assets: []Asset{{Asset: "EUR", Amount: 1}, {Asset: "USD", Amount: 2}},
			orders: []Order{{Side: "BUY", AssetPair: "EUR-USD", Amount: 1, Price: 1, Status: statusPending}},
		},
		{
			name:   "postgres",
			db:     "postgres",
			assets: []Asset{{Asset: "EUR", Amount: 1}, {Asset: "USD", Amount: 2}},
			orders: []Order{{Side: "BUY", AssetPair: "EUR-USD", Amount: 1, Price: 1, Status: statusPending}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := storeFactory(t, tt.db)
			user, id := randomTestUser(t, db)
			api := api{db: db, matchmaker: fakeMatcher{}}
			server := httptest.NewServer(api.routes())
			defer server.Close()

			t.Run("no auth", func(t *testing.T) {
				req, _ := http.NewRequest("GET", server.URL+"/assets", nil)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}

				if got, want := resp.StatusCode, http.StatusForbidden; got != want {
					t.Errorf("got %v want %v", got, want)
				}
			})

			t.Run("wrong auth", func(t *testing.T) {
				req, _ := http.NewRequest("GET", server.URL+"/assets", nil)
				req.Header.Add("Authorization", "Basic "+basicAuth("foo", "bar"))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}

				if got, want := resp.StatusCode, http.StatusUnauthorized; got != want {
					t.Errorf("got %v want %v", got, want)
				}
			})

			t.Run("assets", func(t *testing.T) {
				for _, asset := range tt.assets {
					asset.userID = id
					_ = db.SaveAsset(asset)
				}

				req, _ := http.NewRequest("GET", server.URL+"/assets", nil)
				req.Header.Add("Authorization", "Basic "+basicAuth(user, user))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}

				var assets []Asset
				if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
					t.Fatal(err)
				}
				if got, want := assets, tt.assets; !reflect.DeepEqual(got, want) {
					t.Errorf("got %v want %v", got, want)
				}
			})

			t.Run("orders", func(t *testing.T) {
				for _, order := range tt.orders {
					b, _ := json.Marshal(order)

					req, _ := http.NewRequest("POST", server.URL+"/orders", bytes.NewBuffer(b))
					req.Header.Add("Authorization", "Basic "+basicAuth(user, user))

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						t.Fatal(err)
					}

					if got, want := resp.StatusCode, http.StatusOK; got != want {
						t.Errorf("got %v want %v", got, want)
					}
				}

				req, _ := http.NewRequest("GET", server.URL+"/orders", nil)
				req.Header.Add("Authorization", "Basic "+basicAuth(user, user))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}

				var orders []Order
				if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
					t.Fatal(err)
				}
				if got, want := orders, tt.orders; !reflect.DeepEqual(got, want) {
					t.Errorf("got %v want %v", got, want)
				}
			})

			t.Run("orders insufficient funds", func(t *testing.T) {
				user, _ := randomTestUser(t, db)
				for _, side := range []string{"SELL", "BUY"} {
					b, _ := json.Marshal(Order{Amount: 1, AssetPair: "EUR-USD", Side: side})

					req, _ := http.NewRequest("POST", server.URL+"/orders", bytes.NewBuffer(b))
					req.Header.Add("Authorization", "Basic "+basicAuth(user, user))

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					b, _ = io.ReadAll(resp.Body)
					if !strings.Contains(string(b), ErrInsufficientFunds.Error()) {
						t.Errorf("wrong error message")
					}
					if got, want := resp.StatusCode, http.StatusBadRequest; got != want {
						t.Errorf("got %v want %v", got, want)
					}
				}
			})
		})
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

type fakeMatcher struct {
}

func (f fakeMatcher) VerifyMatch() []Order {
	return nil
}

func (f fakeMatcher) AddOrderAndMatch(Order) []Order {
	return nil
}
