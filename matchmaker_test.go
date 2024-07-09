package main

import (
	"reflect"
	"testing"
)

func Test_newMatchMaker(t *testing.T) {
	tests := []struct {
		name         string
		orders       []Order
		expectedBuy  []int
		expectedSell []int
		match        []int
	}{
		{
			name: "simple",
			orders: []Order{
				{id: 0, Side: "BUY", Price: 1, Amount: 100},
				{id: 1, Side: "SELL", Price: 1, Amount: 100},
			},
			expectedBuy:  []int{0},
			expectedSell: []int{1},
			match:        []int{0, 1},
		},
		{
			name: "more complex",
			orders: []Order{
				{id: 0, Side: "BUY", Price: 1},
				{id: 1, Side: "SELL", Price: 10},
				{id: 2, Side: "BUY", Price: 2},
				{id: 3, Side: "SELL", Price: 1},
				{id: 4, Side: "BUY", Price: 3},
			},
			expectedBuy:  []int{0, 2, 4},
			expectedSell: []int{3, 1},
			match:        []int{0, 3},
		},
		{
			name: "no match",
			orders: []Order{
				{id: 0, Side: "BUY", Price: 11},
				{id: 1, Side: "BUY", Price: 1},
				{id: 2, Side: "BUY", Price: 111},
				{id: 3, Side: "SELL", Price: 222},
				{id: 4, Side: "SELL", Price: 22},
				{id: 5, Side: "SELL", Price: 2},
			},
			expectedBuy:  []int{1, 0, 2},
			expectedSell: []int{5, 4, 3},
		},
		{
			name: "last match",
			orders: []Order{
				{id: 0, Side: "BUY", Price: 11},
				{id: 1, Side: "BUY", Price: 1},
				{id: 2, Side: "BUY", Price: 222},
				{id: 3, Side: "SELL", Price: 222},
				{id: 4, Side: "SELL", Price: 22},
				{id: 5, Side: "SELL", Price: 2},
			},
			expectedBuy:  []int{1, 0, 2},
			expectedSell: []int{5, 4, 3},
			match:        []int{2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMatchMaker(tt.orders)
			cur := m.buy
			var gotBuy []int
			for cur.next != nil {
				gotBuy = append(gotBuy, cur.next.order.id)
				cur = cur.next
			}
			if got, want := gotBuy, tt.expectedBuy; !reflect.DeepEqual(got, want) {
				t.Errorf("got %v want %v", got, want)
			}

			cur = m.sell
			var gotSell []int
			for cur.next != nil {
				gotSell = append(gotSell, cur.next.order.id)
				cur = cur.next
			}
			if got, want := gotSell, tt.expectedSell; !reflect.DeepEqual(got, want) {
				t.Errorf("got %v want %v", got, want)
			}

			matches := m.VerifyMatch()
			if len(tt.match) != 0 {
				if len(matches) == 0 {
					t.Fatal("empty matches")
				}
				if got, want := matches[0].id, tt.match[0]; got != want {
					t.Errorf("got %v want %v", got, want)
				}
				if got, want := matches[1].id, tt.match[1]; got != want {
					t.Errorf("got %v want %v", got, want)
				}
			}
		})
	}
}
