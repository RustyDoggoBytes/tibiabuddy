package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type FormerName struct {
	Name string
	NotificationEmail string
	LastChecked time.Time
	IsAvailable bool
}

type CharacterSearch struct{
	FormerNames []string
	NameInput string
	Name string
	World string
	Trackable bool
	Error error 
}

type TibiaDataApi struct {
	Url string
}

type TibiaApiResponse struct {
	Information interface{} `json:"information"`
}

type CharacterResponse struct {
	TibiaApiResponse
	Character CharacterWrapper `json:"character"`
}
type CharacterWrapper struct {
	Character Character `json:"character"`
}

type Character struct {
	Name string `json:"name"`
	World string `json:"world"`
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

	trackable := false
	formerNames := j.Character.Character.FormerNames
	for _, formerName := range(formerNames) {
		if strings.ToLower(formerName) == strings.ToLower(name) {
			trackable = true
			break
		}
	}

	return &CharacterSearch{
		NameInput: name,
		Name: j.Character.Character.Name,
		FormerNames: j.Character.Character.FormerNames,
		World: j.Character.Character.World,
		Trackable: trackable,
	}, nil
}


var formerNames = []FormerName {
	{Name: "Mario", NotificationEmail: "rustydoggobytes@gmail.com", LastChecked: time.Now(), IsAvailable: true},
	{Name: "Luigi", NotificationEmail: "rustydoggobytes@gmail.com", LastChecked: time.Now() },
}


func main() {
	t := TibiaDataApi{
		Url: "https://api.tibiadata.com",
	}
	e := echo.New()
	e.POST("/former-name/search", func(c echo.Context) error {
		formerName := c.FormValue("former-name")
		searchCharacter, err := t.SearchCharacter(formerName)

		if err != nil {
			e.Logger.Error("failed to search for character", err)
			errorMsg := errors.New("Fail to search. Try again")
			searchCharacter = &CharacterSearch{Error: errorMsg}
		}

		component := index(formerNames, searchCharacter, nil)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.GET("/", func(c echo.Context) error {
		component := index(formerNames, nil, nil)
		return component.Render(c.Request().Context(), c.Response())
	})

	e.DELETE("/former-names/:name", func (c echo.Context) error {
		formerName := c.Param("name")

		removeIdx := -1
		for idx, name := range(formerNames) {
			if name.Name == formerName {
				removeIdx = idx
				println(formerName, removeIdx)
				break 
			}
		}


		var err error
		if removeIdx == -1 {
			err = errors.New(fmt.Sprintf("Former Name %s not found", formerName))
		}

		formerNames = append(formerNames[:removeIdx], formerNames[removeIdx+1:]...)
		component := index(formerNames, nil, err)
		return component.Render(c.Request().Context(), c.Response())
		})

	e.POST("/former-names", func (c echo.Context) error {
		formerName := c.FormValue("former-name")
		notificationEmail := c.FormValue("notification-email")


		formerNames = append(formerNames, FormerName{Name: formerName, NotificationEmail: notificationEmail, LastChecked: time.Now()})
		component := index(formerNames, nil, nil)
		return component.Render(c.Request().Context(), c.Response())
		})

	e.Logger.Fatal(e.Start("127.0.0.1:1323"))
}
