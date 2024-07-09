package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

var ErrNotFound = errors.New("no long url associated to this short url")
var errNoRowsMsg = "no rows in result set" // can't use sql.ErrNoRows because it has a prefix "sql:" which is absent somehow

const (
	statusPending = "pending"
	statusFilled  = "filled"
)

type Asset struct {
	id     int
	userID int
	Asset  string  `json:"asset_type"`
	Amount float64 `json:"amount"`
}

type Order struct {
	id        int
	userID    int
	Side      string  `json:"side"`
	AssetPair string  `json:"asset_pair"`
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price"`
	Status    string  `json:"status"`
}

type store interface {
	SaveUser(username string, password []byte) (int, error)
	User(username string) (id int, password []byte, err error)
	SaveAsset(asset Asset) error
	Assets(userID int) (assets []Asset, err error)
	SaveOrder(order *Order) error
	UserOrders(userID int) ([]Order, error)
	PendingOrders(pair string) ([]Order, error)
	FillOrder(order Order) error
	Close()
}

type mem struct {
	userIDs   map[string]int
	passwords map[int][]byte
	assets    map[int]map[string]float64
	orders    []Order
}

func newMem() *mem {
	return &mem{
		userIDs:   make(map[string]int),
		passwords: make(map[int][]byte),
		assets:    make(map[int]map[string]float64),
	}
}

func (m *mem) FillOrder(order Order) error {
	if m.orders[order.id].Status == statusFilled {
		return fmt.Errorf("order already filled")
	}
	m.orders[order.id].Status = statusFilled
	return nil
}

func (m *mem) UserOrders(userID int) (orders []Order, err error) {
	for _, order := range m.orders {
		if order.userID != userID {
			continue
		}
		orders = append(orders, order)
	}
	return
}

func (m *mem) PendingOrders(pair string) (pendings []Order, err error) {
	for _, order := range m.orders {
		if order.AssetPair != pair || order.Status == statusFilled {
			continue
		}
		pendings = append(pendings, order)
	}
	return
}

func (m *mem) User(username string) (int, []byte, error) {
	id, ok := m.userIDs[username]
	if !ok {
		return 0, nil, ErrNotFound
	}
	return id, m.passwords[id], nil
}

func (m *mem) SaveUser(username string, password []byte) (int, error) {
	if _, exist := m.userIDs[username]; exist {
		return m.userIDs[username], nil
	}
	id := len(m.userIDs)
	m.userIDs[username] = id
	m.passwords[id] = password
	return id, nil
}

func (m *mem) Assets(userID int) ([]Asset, error) {
	if m.assets[userID] == nil {
		return []Asset{}, nil
	}
	var b []Asset
	for k, v := range m.assets[userID] {
		b = append(b, Asset{
			Asset:  k,
			Amount: v,
		})
	}
	return b, nil
}

func (m *mem) SaveAsset(asset Asset) error {
	if m.assets[asset.userID] == nil {
		m.assets[asset.userID] = make(map[string]float64)
	}
	m.assets[asset.userID][asset.Asset] = asset.Amount
	return nil
}

func (m *mem) SaveOrder(order *Order) error {
	order.Status = statusPending
	order.id = len(m.orders)
	m.orders = append(m.orders, *order)
	return nil
}

func (m *mem) Close() {}

type postgres struct {
	pool *pgxpool.Pool
}

func newPostgres(db string) (*postgres, error) {
	pool, err := pgxpool.New(context.Background(), db)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}

	return &postgres{
		pool: pool,
	}, nil
}

func (db postgres) User(username string) (id int, password []byte, err error) {
	err = db.pool.QueryRow(context.Background(), "select id, password from users where username=$1", username).Scan(&id, &password)
	if err != nil {
		if strings.Contains(err.Error(), errNoRowsMsg) {
			return -1, nil, ErrNotFound
		}
		return -1, nil, fmt.Errorf("cannot get user: %v", err)
	}
	return
}

func (db postgres) SaveUser(username string, password []byte) (int, error) {
	id := -1
	err := db.pool.QueryRow(context.Background(),
		`insert into users(username, password) values ($1, $2) returning id`, username, password,
	).Scan(&id)
	if err != nil {
		return id, fmt.Errorf("cannot save user: %v", err)
	}
	return id, nil
}

func (db postgres) Assets(userID int) (assets []Asset, err error) {
	rows, err := db.pool.Query(context.Background(), "select id, asset_type, balance from assets where userid=$1", userID)
	if err != nil {
		if strings.Contains(err.Error(), errNoRowsMsg) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get assets: %v", err)
	}
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(&asset.id, &asset.Asset, &asset.Amount); err != nil {
			return nil, fmt.Errorf("cannot read assets: %v", err)
		}
		assets = append(assets, asset)
	}
	return
}

func (db postgres) SaveAsset(asset Asset) error {
	_, err := db.pool.Exec(context.Background(), `insert into assets(userid, asset_type, balance) values ($1, $2, $3)`, asset.userID, asset.Asset, asset.Amount)
	if err != nil {
		return fmt.Errorf("cannot save asset: %v", err)
	}
	return nil
}

func (db postgres) SaveOrder(order *Order) error {
	order.Status = statusPending
	err := db.pool.QueryRow(context.Background(),
		`insert into orders(userid, side, asset_pair, amount, price, status) values ($1, $2, $3, $4, $5, $6) returning id`, order.userID, order.Side, order.AssetPair, order.Amount, order.Price, order.Status,
	).Scan(&order.id)
	if err != nil {
		return fmt.Errorf("cannot save order: %v", err)
	}
	return nil
}

func (db postgres) UserOrders(userID int) (orders []Order, err error) {
	rows, err := db.pool.Query(context.Background(), "select id, side, asset_pair, amount, price, status from orders where userid=$1", userID)
	if err != nil {
		if strings.Contains(err.Error(), errNoRowsMsg) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get order: %v", err)
	}
	for rows.Next() {
		order := Order{userID: userID}
		if err := rows.Scan(&order.id, &order.Side, &order.AssetPair, &order.Amount, &order.Price, &order.Status); err != nil {
			return nil, fmt.Errorf("cannot read order: %v", err)
		}
		orders = append(orders, order)
	}
	return
}

func (db postgres) PendingOrders(pair string) (orders []Order, err error) {
	rows, err := db.pool.Query(context.Background(), "select id, userid, side, asset_pair, amount, price, status from orders where status=$1 and asset_pair=$2", statusPending, pair)
	if err != nil {
		if strings.Contains(err.Error(), errNoRowsMsg) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get order: %v", err)
	}
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.id, &order.userID, &order.Side, &order.AssetPair, &order.Amount, &order.Price, &order.Status); err != nil {
			return nil, fmt.Errorf("cannot read order: %v", err)
		}
		orders = append(orders, order)
	}
	return
}

func (db postgres) FillOrder(order Order) error {
	// todo check if already filled
	_, err := db.pool.Exec(context.Background(), "update orders set status = $1 where id=$2", statusFilled, order.id)
	if err != nil {
		return err
	}

	bought, sold := order.Amount, order.Amount*order.Price
	soldAsset, boughtAsset := order.AssetPair[0:3], order.AssetPair[4:]
	if order.Side == "SELL" {
		bought, sold = sold, bought
		boughtAsset, soldAsset = soldAsset, boughtAsset
	}

	assets, err := db.Assets(order.userID)
	if err != nil {
		return err
	}
	for _, asset := range assets {
		if asset.Asset == boughtAsset {
			_, err := db.pool.Exec(context.Background(), "update assets set balance = $1 where id=$2", asset.Amount+bought, asset.id)
			if err != nil {
				return err
			}
		}
		if asset.Asset == soldAsset {
			_, err := db.pool.Exec(context.Background(), "update assets set balance = $1 where id=$2", asset.Amount-sold, asset.id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (db postgres) Close() {
	db.pool.Close()
}
