package main

import (
	"embed"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

//go:embed static/*
var embeddedFiles embed.FS

var contentHandler = echo.WrapHandler(http.FileServer(http.FS(embeddedFiles)))
var contentRewrite = middleware.Rewrite(map[string]string{"/*": "/static/$1"})

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Error("Error loading .env file")
	}

	db, err := RepositoryClient("data/tibiabuddy.db")
	if err != nil {
		panic(err)
	}

	emailClient := EmailClient(os.Getenv("RESEND_API_TOKEN"), os.Getenv("EMAIL"))
	authService := NewAuthService(db.Db)
	cookieStore := sessions.NewCookieStore([]byte(os.Getenv("SESSION_STORE_SECRET")))

	t := TibiaDataApi{
		Url: "https://tibiadata.rustydoggobytes.com",
	}
	go runBackground(db, &t, &emailClient)

	e := echo.New()
	e.Static("/static", "static")
	e.Use(session.Middleware(cookieStore))
	e.Use(authService.AuthMiddleware)

	e.GET("/*", contentHandler, contentRewrite)

	e.POST("/former-name/search", func(c echo.Context) error {
		formerName := c.FormValue("former-name")
		searchCharacter, err := t.SearchCharacter(formerName)

		if err != nil {
			errorMsg := "Search failed. Try again."
			searchCharacter = &CharacterSearch{Error: errors.New(errorMsg)}
		}
		if !searchCharacter.Found {
			searchCharacter = &CharacterSearch{Error: errors.New(fmt.Sprintf("Character Not Found - %s", formerName))}
		}

		formerNames, _ := db.GetFormerNames()
		component := layout(index(formerNames, searchCharacter, nil), true)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.GET("/", func(c echo.Context) error {
		formerNames, _ := db.GetFormerNames()
		component := layout(index(formerNames, nil, nil), true)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.DELETE("/former-names/:name", func(c echo.Context) error {
		formerName := c.Param("name")
		err := db.DeleteFormerName(formerName)

		if err != nil {
			if err.Error() == "not found" {
				err = errors.New(fmt.Sprintf("Former Name %s not found", formerName))
			}
		}

		formerNames, _ := db.GetFormerNames()
		component := layout(index(formerNames, nil, nil), true)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.POST("/former-names", func(c echo.Context) error {
		formerName := c.FormValue("former-name")
		notificationEmail := c.FormValue("notification-email")
		var status FormerNameStatus
		status = status.FromString(c.FormValue("status"))

		if err := db.SaveFormerName(FormerName{Name: formerName, NotificationEmail: notificationEmail, LastChecked: time.Now(), Status: status}); err != nil {
			e.Logger.Fatal(err)
		}
		formerNames, _ := db.GetFormerNames()
		component := layout(index(formerNames, nil, nil), true)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.POST("/send-email", func(c echo.Context) error {
		emails := strings.Split(c.FormValue("emails"), ",")
		formerName := c.FormValue("name")

		emailClient.NotifyUserFormerNameIsAvailable(emails, formerName)

		formerNames, _ := db.GetFormerNames()
		component := layout(index(formerNames, nil, nil), true)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.GET("/signup", SignUpPage)
	e.GET("/signin", SignInPage)

	e.POST("/signup", authService.SignUp)
	e.POST("/signin", authService.SignIn)
	e.GET("/signout", authService.SignOut)

	e.Logger.Fatal(e.Start("0.0.0.0:8080"))
}

func SignUpPage(c echo.Context) error {
	component := layout(signUp(nil), false)
	return component.Render(c.Request().Context(), c.Response())
}

func (a *AuthService) SignUp(c echo.Context) error {
	email := c.FormValue("email")
	password1 := c.FormValue("password1")
	password2 := c.FormValue("password2")

	var errorMsg string
	if password1 != password2 {
		errorMsg = "password do not match"
	} else {
		_, err := a.signUp(email, password1)
		if err != nil {
			errorMsg = err.Error()
		}
	}

	if errorMsg != "" {
		component := layout(signUp(&errorMsg), false)
		return component.Render(c.Request().Context(), c.Response())
	}

	return c.Redirect(http.StatusFound, "/signin")

}

func SignInPage(c echo.Context) error {
	component := layout(signIn(nil), false)
	return component.Render(c.Request().Context(), c.Response())
}

func (a *AuthService) SignIn(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	user, err := a.signIn(email, password)
	if err != nil {
		var errorMsg = err.Error()
		component := layout(signIn(&errorMsg), false)
		return component.Render(c.Request().Context(), c.Response())
	}

	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	sess.Values["user_id"] = user.ID

	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusFound, "/")
}

func (a *AuthService) SignOut(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusFound, "/signin")
}

func (a *AuthService) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().URL.Path == "/signin" || c.Request().URL.Path == "/signup" {
			return next(c)
		}

		sess, _ := session.Get("session", c)
		if sess.Values["user_id"] == nil {
			fmt.Println("no id. redirecting")
			return c.Redirect(http.StatusFound, "/signin")
		}

		return next(c)
	}
}
