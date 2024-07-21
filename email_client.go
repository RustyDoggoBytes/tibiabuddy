package main

import (
	"fmt"
	"github.com/resend/resend-go/v2"
	"log"
)

type emailClient struct {
	Client    *resend.Client
	FromEmail string
}

func EmailClient(resendApiKey, fromEmail string) emailClient {
	client := resend.NewClient(resendApiKey)
	return emailClient{client, fromEmail}
}

func (c *emailClient) NotifyUserFormerNameIsAvailable(toEmails []string, name string) {
	params := &resend.SendEmailRequest{
		To:      toEmails,
		From:    c.FromEmail,
		Text:    fmt.Sprintf("%s is now available. Log in to Tibia  to claim it!", name),
		Subject: fmt.Sprintf("Tibia Buddy - %s is now available!", name),
	}

	_, err := c.Client.Emails.Send(params)
	if err != nil {
		log.Fatal(err)
	}
}
