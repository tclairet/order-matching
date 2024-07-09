# Order Matching

This project implements a REST API backend in Golang for trading between two currency assets: EUR and USD. 
The API allows users to trade assets, with data stored in a PostgreSQL database.

## Features

- User authentication with Basic Authentication
- Asset balance retrieval for users
- Creation of limit buy and sell orders
- Real-time order matching and balance updates

## Setup

```
docker-compose up -d
 ```

The application should now be running on `http://localhost:8080`.

## Example

check assets balance of two pre-seeded user
```
curl -u user:password http://localhost:8080/assets
curl -u user2:password2 http://localhost:8080/orders
```

make an order between the two
```
curl -u user:password -X POST -d '{"side":"SELL", "asset_pair":"EUR-USD", "amount":1200, "price": 1.2}' http://localhost:8080/orders
curl -u user2:password2 -X POST -d '{"side":"BUY", "asset_pair":"EUR-USD", "amount":1200, "price": 1.2}' http://localhost:8080/orders
```

assets and orders have been updated
```
curl -u user:password http://localhost:8080/assets
curl -u user:password http://localhost:8080/orders

curl -u user2:password2 http://localhost:8080/assets
curl -u user2:password2 http://localhost:8080/orders
```

## Seed

The database is seeded with some prefunded accounts:
- user '1' and user '2' (repeat username as password) have pre seeded matching order 
- user '3' to '10' (repeat username as password) with 10000 USD and EUR
- user 'user', 'user1' and 'user2' with password 'password', 'password1' and 'password2' and with 10000 USD and EUR
- user 'admin' (repeat username as password) with 0 USD and 10000 EUR

For more details check `config.go`

## Limitation
There is some todo in the code, but to summarise:
- verify the status of the order in db before marking it has `filled`
- sanitize user inputs (asset, pair, amount, username, ...) 
- use a sliding window for matchMaker in `VerifyMatch`, because `match(order)` is o(n) but we call it n time in `VerifyMatch` 
- node should have prev pointer, that would simplify the readability
