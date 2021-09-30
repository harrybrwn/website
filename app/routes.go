package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

// Debug cooresponds with the debug flag
var (
	Debug = false
)

func init() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.PanicOnError)
	BoolFlag(&Debug, "debug", "turn on debugging options")

	web.TemplateDir = "templates/"
	web.BaseTemplates = []string{"/index.html", "/nav.html"} // included in all pages
}

// Routes is a list of all the app's routes
var Routes = []web.Route{
	&web.Page{
		Title:     "Harry Brown",
		Template:  "pages/home.html",
		RoutePath: "/",
		RequestHook: func(self *web.Page, w http.ResponseWriter, r *http.Request) {
			self.Data = &struct {
				Age   string
				Quote Quote
			}{
				Age:   getAge(),
				Quote: randomQuote(),
			}
		},
		HotReload: Debug,
	},
	// &web.Page{
	// 	Title:     "Freelancing",
	// 	Template:  "pages/freelance.html",
	// 	RoutePath: "/freelance",
	// },
	// &web.Page{
	// 	Title:     "Resume",
	// 	Template:  "pages/resume.html",
	// 	RoutePath: "/resume",
	// },
	web.NewNestedRoute("/api", apiroutes...).SetHandler(&web.JSONRoute{
		Static: func() interface{} { return info{Error: "Not implimented"} },
	}),
	// web.NewRoute("/github", http.RedirectHandler("https://github.com/harrybrwn", 301)),
}

var apiroutes = []web.Route{
	web.APIRoute("info", func(w http.ResponseWriter, r *http.Request) interface{} {
		return info{
			Age:    time.Since(bday).Hours() / 24 / 365,
			Uptime: time.Since(serverStart),
		}
	}),
	web.APIRoute("quote", func(rw http.ResponseWriter, r *http.Request) interface{} {
		return randomQuote()
	}),
	web.APIRoute("quotes", func(rw http.ResponseWriter, r *http.Request) interface{} {
		quotesMu.Lock()
		defer quotesMu.Unlock()
		return quotes
	}),
}

var bday = time.Date(1998, time.August, 4, 4, 0, 0, 0, time.UTC)

func getAge() string {
	age := time.Since(bday).Hours() / 24 / 365
	return fmt.Sprintf("%d", int(age))
}

type Quote struct {
	Body   string `json:"body"`
	Author string `json:"author"`
}

var (
	quotesMu sync.Mutex
	quotes   = []Quote{
		{Body: "Do More", Author: "Casey Neistat"},
		{Body: "Imagination is something you do alone.", Author: "Steve Wazniak"},
		{Body: "I am so clever that sometimes I don't understand a single word of what I am saying.", Author: "Oscar Wilde"},
		{Body: "I did not have sexual relations with that woman.", Author: "Bill Clinton"},
		{Body: "I was never really good at anything except for the ability to learn.", Author: "Kanye West"},
		{Body: "I love sleep; It's my favorite.", Author: "Kanye West"},
		{Body: "I'm gunna be the next hokage!", Author: "Naruto Uzumaki"},
		{Body: "Bush did 911.", Author: "A very intelligent internet user"},
		{Body: "Have you ever had a dream that, that, um, that you had, uh, that you had to, you could, you do, you wit, you wa, you could do so, you do you could, you want, you wanted him to do you so much you could do anything?", Author: "That one kid"},
	}
)

func randomQuote() Quote {
	quotesMu.Lock()
	defer quotesMu.Unlock()
	return quotes[rand.Intn(len(quotes))]
}

func getResume(file string) *resumeContent {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println(err)
		return nil
	}
	c := &resumeContent{}

	if err = json.Unmarshal(b, c); err != nil {
		log.Println(err)
		return nil
	}
	return c
}

type resumeContent struct {
	Experience []resumeItem
	Education  []resumeItem
}

type resumeItem struct {
	Name, Title, Date, Content string
	BulletPoints               []string
}

type info struct {
	Age       float64       `json:"age,omitempty"`
	Uptime    time.Duration `json:"uptime,omitempty"`
	GOVersion string        `json:"goversion,omitempty"`
	Error     string        `json:"error,omitempty"`
}
