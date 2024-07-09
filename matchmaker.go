package main

import "log/slog"

type node struct {
	order Order
	// todo add previous, it would simplify reading
	next *node
}

type matchmaker interface {
	VerifyMatch() []Order
	AddOrderAndMatch(order Order) []Order
}

type linkedListMatchmaker struct {
	sell, buy *node
}

func newMatchMaker(orders []Order) linkedListMatchmaker {
	m := linkedListMatchmaker{
		sell: &node{},
		buy:  &node{},
	}
	for _, order := range orders {
		m.addOrder(order)
	}
	return m
}

func (m linkedListMatchmaker) match(prev *node, head *node) (matches []Order) {
	for len(matches) != 2 && head.next != nil && head.next.order.Price <= prev.next.order.Price {
		if head.next.order.Price == prev.next.order.Price && head.next.order.Amount == prev.next.order.Amount {
			matches = append(matches, prev.next.order, head.next.order)
			slog.Info("match",
				"id1", prev.next.order.id,
				"id2", head.next.order.id,
				"pair", prev.next.order.AssetPair,
				"price", prev.next.order.Price,
				"amount", prev.next.order.Amount,
			)
			// delete node
			prev.next, head.next = prev.next.next, head.next.next
		}
		head = head.next
	}
	return matches
}

func (m linkedListMatchmaker) VerifyMatch() (matches []Order) {
	cur := m.buy
	for cur != nil && cur.next != nil {
		// todo we could use a sliding window here, and go from o(n2) to o(n)
		match := m.match(cur, m.sell)
		matches = append(matches, match...)
		cur = cur.next
	}
	return
}

func (m linkedListMatchmaker) AddOrderAndMatch(order Order) []Order {
	prev := m.addOrder(order)
	head := m.buy
	if order.Side == "BUY" {
		head = m.sell
	}
	return m.match(prev, head)
}

func (m linkedListMatchmaker) addOrder(order Order) (prev *node) {
	cur := m.buy
	if order.Side == "SELL" {
		cur = m.sell
	}
	for cur.next != nil && cur.next.order.Price <= order.Price {
		cur = cur.next
	}
	newNode := node{
		order: order,
	}
	if cur != nil {
		newNode.next = cur.next
		cur.next = &newNode
	}
	return cur
}
