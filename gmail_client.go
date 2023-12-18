package main

import (
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unkown fromServer")
		}
	}
	return nil, nil
}

type emailClient struct {
	fromEmail string
	auth      smtp.Auth
}

func EmailClient(fromEmail, password string) emailClient {
	return emailClient{fromEmail, LoginAuth(fromEmail, password)}
}

func (c *emailClient) NotifyUserFormerNameIsAvailable(toEmails []string, name string) {
	msg := fmt.Sprintf("To: %s\r\n"+
		"Subject: Tibia Buddy - %s is now Available!\r\n"+
		"\r\n"+
		"%s is now available. Go login to tibia in order to catch it!\r\n", strings.Join(toEmails, ";"), name, name)
	err := smtp.SendMail("smtp.gmail.com:587", c.auth, c.fromEmail, toEmails, []byte(msg))
	if err != nil {
		log.Fatal(err)
	}
}
