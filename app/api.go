package app

import (
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

func HandleInfo(w http.ResponseWriter, r *http.Request) interface{} {
	return info{
		Name: "Harry Brown",
		Age:  math.Round(GetAge()),
	}
}

type info struct {
	Name      string        `json:"name,omitempty"`
	Age       float64       `json:"age,omitempty"`
	Uptime    time.Duration `json:"uptime,omitempty"`
	GOVersion string        `json:"goversion,omitempty"`
	Error     string        `json:"error,omitempty"`
}

var birthTimestamp = time.Date(
	1998, time.August, 4, // 1998-08-04
	4, 40, 0, 0, // 4:40 AM
	mustLoadLocation("America/Los_Angeles"),
)

func mustLoadLocation(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

func GetAge() float64 {
	return time.Since(birthTimestamp).Seconds() / 60 / 60 / 24 / 365
}

func GetBirthday() time.Time { return birthTimestamp }

type Quote struct {
	Body   string `json:"body"`
	Author string `json:"author"`
}

var (
	quotesMu sync.Mutex
	quotes   = []Quote{
		{Body: "Do More", Author: "Casey Neistat"},
		{Body: "Imagination is something you do alone.", Author: "Steve Wazniak"},
		{Body: "I was never really good at anything except for the ability to learn.", Author: "Kanye West"},
		{Body: "I love sleep; It's my favorite.", Author: "Kanye West"},
		{Body: "I'm gunna be the next hokage!", Author: "Naruto Uzumaki"},
		{
			Body: "I am so clever that sometimes I don't understand a single word of " +
				"what I am saying.",
			Author: "Oscar Wilde",
		},
		{
			Body: "Have you ever had a dream that, that, um, that you had, uh, " +
				"that you had to, you could, you do, you wit, you wa, you could " +
				"do so, you do you could, you want, you wanted him to do you so much " +
				"you could do anything?",
			Author: "That one kid",
		},
		// {Body: "I did not have sexual relations with that woman.", Author: "Bill Clinton"},
		// {Body: "Bush did 911.", Author: "A very intelligent internet user"},
	}
)

func RandomQuote() Quote {
	quotesMu.Lock()
	defer quotesMu.Unlock()
	return quotes[rand.Intn(len(quotes))]
}

func GetQuotes() []Quote {
	return quotes
}
