package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		if err := godotenv.Load("/root/apps/.tibiabuddy.env"); err != nil {
			panic("Error loading .env file")
		}
	}

	t := TibiaDataApi{
		Url: "https://api.tibiadata.com",
	}

	emailClient := EmailClient(os.Getenv("email"), os.Getenv("email_password"))
	db, err := RepositoryClient("tibiabuddy.db")
	authService := NewAuthService(db.Db)
	cookieStore := sessions.NewCookieStore([]byte(os.Getenv("session_store_secret")))

	if err != nil {
		panic(err)
	}

	go runBackground(db, &t, &emailClient)

	e := echo.New()

	e.Use(session.Middleware(cookieStore))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fmt.Println("middleware")
			if c.Request().URL.Path != "/signin" && c.Request().URL.Path == "/signup" {
				sess, _ := session.Get("session", c)
				if sess.Values["user_id"] == nil {
					fmt.Println("no id. redirecting")
					return c.Redirect(http.StatusFound, "/signin")
				}
			}

			return next(c)
		}
	})

	e.AutoTLSManager.HostPolicy = autocert.HostWhitelist("tibiabuddy.rustydoggobytes.com")
	e.AutoTLSManager.Cache = autocert.DirCache("/var/www/.cache")

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
		component := layout(index(formerNames, nil, nil))
		return component.Render(c.Request().Context(), c.Response())
	})

	e.GET("/", func(c echo.Context) error {
		fmt.Println("index")
		formerNames, _ := db.GetFormerNames()
		component := layout(index(formerNames, nil, nil))
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
		component := layout(index(formerNames, nil, nil))
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
		component := layout(index(formerNames, nil, nil))
		return component.Render(c.Request().Context(), c.Response())
	})

	e.POST("/send-email", func(c echo.Context) error {
		emails := strings.Split(c.FormValue("emails"), ",")
		formerName := c.FormValue("name")

		emailClient.NotifyUserFormerNameIsAvailable(emails, formerName)

		formerNames, _ := db.GetFormerNames()
		component := layout(index(formerNames, nil, nil))
		return component.Render(c.Request().Context(), c.Response())
	})

	e.GET("/signup", SignUpPage)
	e.POST("/signup", authService.SignUp)

	e.GET("/signin", SignInPage)
	e.POST("/signin", authService.SignIn)

	e.GET("/signout", authService.SignOut)

	e.Logger.Fatal(e.Start("127.0.0.1:1324"))
}

func SignUpPage(c echo.Context) error {
	component := layout(signUp(nil))
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
		component := layout(signUp(&errorMsg))
		return component.Render(c.Request().Context(), c.Response())
	}

	return c.Redirect(http.StatusFound, "/signin")

}

func SignInPage(c echo.Context) error {
	component := layout(signIn(nil))
	return component.Render(c.Request().Context(), c.Response())
}

func (a *AuthService) SignIn(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	user, err := a.signIn(email, password)
	if err != nil {
		var errorMsg = err.Error()
		component := layout(signIn(&errorMsg))
		return component.Render(c.Request().Context(), c.Response())
	}

	fmt.Println(user.ID)

	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	sess.Values["user_id"] = user.ID
	sess.Save(c.Request(), c.Response())
	fmt.Println(sess)

	return c.Redirect(http.StatusFound, "/")
}

func (a *AuthService) SignOut(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusFound, "/signin")
}
