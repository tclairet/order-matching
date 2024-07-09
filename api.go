package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

type api struct {
	db         store
	matchmaker matchmaker
}

func newAPI(db store) api {
	pending, err := db.PendingOrders("EUR-USD")
	if err != nil {
		panic(err)
	}
	m := newMatchMaker(pending)
	matches := m.VerifyMatch()
	for _, match := range matches {
		if err := db.FillOrder(match); err != nil {
			panic(err)
		}
	}
	return api{db: db, matchmaker: m}
}

func (api api) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /assets", api.basicAuth(api.assets))
	mux.HandleFunc("POST /orders", api.basicAuth(api.order))
	mux.HandleFunc("GET /orders", api.basicAuth(api.orders))
	return mux
}

func (api api) assets(w http.ResponseWriter, r *http.Request) {
	userID, err := mustUserID(r)
	if err != nil {
		RespondWithError(w, http.StatusForbidden, err)
		return
	}
	assets, err := api.db.Assets(userID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err)
		return
	}
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Asset < assets[j].Asset
	})
	RespondWithJSON(w, http.StatusOK, assets)
}

func (api api) orders(w http.ResponseWriter, r *http.Request) {
	userID, err := mustUserID(r)
	if err != nil {
		RespondWithError(w, http.StatusForbidden, err)
		return
	}
	orders, err := api.db.UserOrders(userID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, orders)
}

func (api api) verifyLiquidity(order Order) error {
	assets, err := api.db.Assets(order.userID)
	if err != nil {
		return err
	}
	symbol := order.AssetPair[0:3]
	amount := order.Amount * order.Price
	if order.Side == "SELL" {
		symbol = order.AssetPair[4:]
		amount = order.Amount
	}
	for _, asset := range assets {
		if asset.Asset != symbol {
			continue
		}
		if asset.Amount >= amount {
			return nil
		}
	}
	return ErrInsufficientFunds
}

func (api api) order(w http.ResponseWriter, r *http.Request) {
	userID, err := mustUserID(r)
	if err != nil {
		RespondWithError(w, http.StatusForbidden, err)
		return
	}
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		RespondWithError(w, http.StatusBadRequest, err)
		return
	}
	order.userID = userID

	if err := api.verifyLiquidity(order); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInsufficientFunds) {
			status = http.StatusBadRequest
		}
		RespondWithError(w, status, err)
		return
	}

	if err := api.db.SaveOrder(&order); err != nil {
		RespondWithError(w, http.StatusInternalServerError, err)
		return
	}
	matches := api.matchmaker.AddOrderAndMatch(order)
	for _, match := range matches {
		if err := api.db.FillOrder(match); err != nil {
			RespondWithError(w, http.StatusInternalServerError, err)
			return
		}
	}
}

const userIDKey = "userID"

func mustUserID(r *http.Request) (int, error) {
	v := r.Context().Value(userIDKey)
	if v == nil {
		return -1, fmt.Errorf("invalid user id")
	}
	return v.(int), nil
}

func (api api) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			passwordHash := sha256.Sum256([]byte(password))
			id, hash, err := api.db.User(username)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			if subtle.ConstantTimeCompare(passwordHash[:], hash[:]) == 1 {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, id)))
				return
			}
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusForbidden)
	}
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "api/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
}

func RespondWithError(w http.ResponseWriter, code int, msg interface{}) {
	var message string
	switch m := msg.(type) {
	case error:
		message = m.Error()
	case string:
		message = m
	}
	RespondWithJSON(w, code, JSONError{Error: message})
}

type JSONError struct {
	Error string `json:"error,omitempty"`
}
