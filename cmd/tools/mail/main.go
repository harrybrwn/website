package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

var apikey string

func main() {
	var env string
	flag.StringVar(&env, "env", ".env", ".env file to look for SENDGRID_API_KEY")
	flag.Parse()

	err := godotenv.Load(env)
	if err != nil {
		log.Fatal(err)
	}

	apikey = os.Getenv("SENDGRID_API_KEY")
	if len(apikey) == 0 {
		log.Fatal("could not find api key")
	}
	err = run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	from := mail.NewEmail("harrybrwn", "noreply@harrybrwn.com")
	to := mail.NewEmail("harry", "harrybrown98@gmail.com")

	subject := "Testing Out Our System"
	// plainTextContent := "and easy to do anywhere, even with Go."
	// plainTextContent := "this is a test message"
	// htmlContent := `<p>this is a test message</p>`
	// htmlContent := "<p>It is easy to send from <em>anywhere</em>, even with Go</p>"
	plainTextContent := "This is a testing email, please disregard its contents."
	htmlContent := `<div>Hi ` + to.Name +
		`,<br><br>This is a testing email, please disregard its contents.` +
		`<br><br>Thank you for you patience.<br><br>Harry</div>`
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(apikey)
	response, err := client.SendWithContext(context.Background(), message)
	if err != nil {
		return err
	} else {
		fmt.Println(response.StatusCode, http.StatusText(response.StatusCode))
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
	return nil
}
