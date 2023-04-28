package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

var apikey string

func main() {
	var (
		env string
		to  = "harrybrown98@gmail.com"
	)
	flag.StringVar(&env, "env", ".env", ".env file to look for SENDGRID_API_KEY")
	flag.StringVar(&to, "to", to, "email address to send an email to")
	flag.Parse()

	err := godotenv.Load(env)
	if err != nil {
		log.Fatal(err)
	}

	apikey = os.Getenv("SENDGRID_API_KEY")
	if len(apikey) == 0 {
		log.Fatal("could not find api key")
	}
	err = run(to)
	if err != nil {
		log.Fatal(err)
	}
}

func run(to string) error {
	from := mail.NewEmail("harrybrwn", "noreply@hrry.io")
	toEmail := mail.NewEmail(emailName(to), to)
	subject := "Testing Out Our System"
	plainTextContent := "This is a testing email, please disregard its contents."
	// plainTextContent = ""
	//htmlContent := `<div>Hi ` + toEmail.Name +
	//	`,<br><br>This is a testing email, please disregard its contents.` +
	//	`<br><br>Thank you for you patience.<br><br>Harry</div>`
	htmlContent := ""
	message := mail.NewSingleEmail(from, subject, toEmail, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(apikey)
	fmt.Printf("%#v\n", message)
	fmt.Printf("body: %s\n", mail.GetRequestBody(message))

	response, err := client.SendWithContext(context.Background(), message)
	if err != nil {
		return err
	} else {
		fmt.Println(response.StatusCode, http.StatusText(response.StatusCode))
		fmt.Println(response.Body)
		for k, v := range response.Headers {
			fmt.Printf("%s: %v\n", k, v)
		}
	}
	return nil
}

func emailName(addr string) string {
	parts := strings.Split(addr, "@")
	if len(parts) == 2 {
		return parts[0]
	}
	return addr
}
