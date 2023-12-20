package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

type FormerNameStatus int

const (
	available FormerNameStatus = iota
	expiring
	unavailable
	unknown
)

func (e FormerNameStatus) String() string {
	switch e {
	case available:
		return "available"
	case unavailable:
		return "unavailable"
	case expiring:
		return "expiring"
	default:
		return "unknown"
	}
}

func (e FormerNameStatus) FromString(s string) FormerNameStatus {
	switch s {
	case "available":
		return available
	case "unavailable":
		return unavailable
	case "expiring":
		return expiring
	default:
		return unknown
	}
}

type FormerName struct {
	Name              string
	NotificationEmail string
	LastChecked       time.Time
	LastUpdatedStatus *time.Time
	Status            FormerNameStatus
}

type CharacterSearch struct {
	Found       bool
	FormerNames []string
	NameInput   string
	Name        string
	World       string
	Trackable   bool
	Error       error
}

type TibiaDataApi struct {
	Url string
}

type TibiaApiResponse struct {
	Information TibiaApiInformation `json:"information"`
}

type TibiaApiInformation struct {
	Status TibiaApiStatus `json:"status"`
}

type TibiaApiStatus struct {
	HttpCode  int `json:"http_code"`
	ErrorCode int `json:"error"`
}

type CharacterResponse struct {
	TibiaApiResponse
	Character CharacterWrapper `json:"character"`
}
type CharacterWrapper struct {
	Character Character `json:"character"`
}

type Character struct {
	Name        string   `json:"name"`
	World       string   `json:"world"`
	FormerNames []string `json:"former_names"`
}

func (t *TibiaDataApi) SearchCharacter(name string) (*CharacterSearch, error) {
	resp, err := http.Get(t.Url + "/v4/character/" + name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var j CharacterResponse
	err = json.NewDecoder(resp.Body).Decode(&j)

	if err != nil {
		return nil, err
	}

	found := true
	if j.Information.Status.ErrorCode == 20001 {
		found = false
	}

	trackable := false
	formerNames := j.Character.Character.FormerNames
	for _, formerName := range formerNames {
		if strings.ToLower(formerName) == strings.ToLower(name) {
			trackable = true
			break
		}
	}

	return &CharacterSearch{
		Found:       found,
		NameInput:   name,
		Name:        j.Character.Character.Name,
		FormerNames: j.Character.Character.FormerNames,
		World:       j.Character.Character.World,
		Trackable:   trackable,
	}, nil
}

func getNewStatus(name string, c *CharacterSearch) FormerNameStatus {
	if !c.Found {
		return available
	}

	if strings.ToLower(c.Name) == strings.ToLower(name) {
		return unavailable
	}

	for _, charFormerName := range c.FormerNames {
		if strings.ToLower(charFormerName) == strings.ToLower(name) {
			return expiring
		}
	}
	fmt.Println("not sure what is happening", c)

	return unknown
}

func runBackground(db *repositoryClient, t *TibiaDataApi) {
	fmt.Println("running background")
	for {
		fmt.Println("started...")
		formerNames, err := db.GetFormerNames()
		if err != nil {
			panic(err)

		}
		for _, name := range formerNames {
			fmt.Println("checking name", name.Name)
			char, err := t.SearchCharacter(name.Name)
			if err != nil {
				fmt.Println(err)
				time.Sleep(1 * time.Second)
				continue
			}

			oldStatus := name.Status
			newStatus := getNewStatus(name.Name, char)
			fmt.Printf("checked name %s old_status=%s new_status=%s\n", name.Name, oldStatus, newStatus)
			if oldStatus != newStatus {
				now := time.Now()
				name.LastUpdatedStatus = &now
			}

			name.Status = newStatus
			name.LastChecked = time.Now()

			db.SaveFormerName(name)
			time.Sleep(1 * time.Second)
		}

		time.Sleep(5 * time.Minute)
	}
}

func main() {
	ssl := ""

	t := TibiaDataApi{
		Url: "https://api.tibiadata.com",
	}

	emailClient := EmailClient("rustydoggobytes@gmail.com", "fkqa dugm wjgs brpa")
	db, err := RepositoryClient("tibiabuddy.db")

	if err != nil {
		panic(err)
	}

	go runBackground(db, &t)

	e := echo.New()
	e.AutoTLSManager.HostPolicy = autocert.HostWhitelist("tibiabuddy.rustydoggobytes.com")
	e.AutoTLSManager.Cache = autocert.DirCache("/var/www/.cache")
	e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		// Be careful to use constant time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(username), []byte("rusty")) == 1 &&
			subtle.ConstantTimeCompare([]byte(password), []byte("$umm3R")) == 1 {
			return true, nil
		}
		return false, nil
	}))

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
		component := index(formerNames, searchCharacter, nil)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.GET("/", func(c echo.Context) error {
		formerNames, _ := db.GetFormerNames()
		component := index(formerNames, nil, nil)
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
		component := index(formerNames, nil, err)
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
		component := index(formerNames, nil, nil)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.POST("/send-email", func(c echo.Context) error {
		emails := strings.Split(c.FormValue("emails"), ",")
		formerName := c.FormValue("name")

		emailClient.NotifyUserFormerNameIsAvailable(emails, formerName)

		formerNames, _ := db.GetFormerNames()
		component := index(formerNames, nil, nil)
		return component.Render(c.Request().Context(), c.Response())
	})

	if ssl == "ssl" {
		e.Logger.Fatal(e.StartAutoTLS("127.0.0.1:1324"))
	} else {
		e.Logger.Fatal(e.Start("127.0.0.1:1324"))
	}
}
