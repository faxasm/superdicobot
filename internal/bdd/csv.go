package bdd

import (
	"strconv"
	"time"
)

type EventRewardRedeem struct {
	Channel   string    `csv:"channel"`
	RewardId  string    `csv:"rewardId"`
	Name      string    `csv:"name"`
	Cost      int       `csv:"cost"`
	UserId    string    `csv:"userId"`
	UserLogin string    `csv:"userLogin"`
	UserName  string    `csv:"userName"`
	DateEvent time.Time `csv:"dateEvent"`
	Status    string    `csv:"status"`
	RedeemId  string    `csv:"redeemId"`
	UpdateAt  time.Time `csv:"updatedAt"`
}

type ListOfEvents []EventRewardRedeem

func MapToEventRewardId(record []string) EventRewardRedeem {
	event := EventRewardRedeem{}
	event.Channel = record[0]
	event.RewardId = record[1]
	event.Name = record[2]
	cost, _ := strconv.Atoi(record[3])
	event.Cost = cost
	event.UserId = record[4]
	event.UserLogin = record[5]
	event.UserName = record[6]
	t, _ := time.Parse(time.RFC3339, record[7])
	event.DateEvent = t
	event.Status = record[8]
	event.RedeemId = record[9]

	if len(record) > 10 {
		updateAt, _ := time.Parse(time.RFC3339, record[10])
		event.UpdateAt = updateAt
	} else {
		event.UpdateAt = t
	}
	return event
}

func MapToEventsRewardId(records [][]string) []EventRewardRedeem {
	events := make([]EventRewardRedeem, 0)
	for _, record := range records {
		events = append(events, MapToEventRewardId(record))
	}
	return events
}

func LastByRedeemId(events []EventRewardRedeem) ListOfEvents {

	eventsGrouped := make(map[string]EventRewardRedeem, 0)
	for _, eventRewardRedeem := range events {
		eventsGrouped[eventRewardRedeem.RedeemId] = eventRewardRedeem
	}
	list := ListOfEvents{}
	for _, eventGrouped := range eventsGrouped {
		list = append(list, eventGrouped)
	}

	return list
}

// Len is part of sort.Interface.
func (d ListOfEvents) Len() int {
	return len(d)
}

// Swap is part of sort.Interface.
func (d ListOfEvents) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (d ListOfEvents) Less(i, j int) bool {
	return d[i].DateEvent.Before(d[j].DateEvent)
}
