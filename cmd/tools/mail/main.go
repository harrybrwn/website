package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gopkg.in/gomail.v2"
)

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

	err = runAhasend("bsky@mail.hrry.me", to)
	if err != nil {
		log.Fatal(err)
	}
}

func runAhasend(from, to string) error {
	email := gomail.NewMessage()
	email.SetAddressHeader("From", from, "Bluesky PDS Admin")
	email.SetAddressHeader("To", to, "PDS User")
	email.SetHeader("Subject", "Sample email test")
	plainTextContent := "This is a testing email, please disregard its contents."
	htmlContent := `<div>Hi ` + "PDS User" +
		`,<br><br>This is a testing email, please disregard its contents.` +
		`<br><br>Thank you for you patience.<br><br>Harry</div>`
	email.SetBody("text/plain", plainTextContent)
	email.SetBody("text/html", htmlContent)
	u, err := url.Parse(os.Getenv("SMTP_URL"))
	if err != nil {
		return err
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return err
	}
	pw, _ := u.User.Password()
	d := gomail.NewDialer(host, int(port), u.User.Username(), pw)
	return d.DialAndSend(email)
}

func runSendgrid(to, apikey string) error {
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
