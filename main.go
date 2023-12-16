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

type FormerNameStatus int

const (
	available FormerNameStatus = iota
	unavailable
	expiring
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

type FormerName struct {
	Name string
	NotificationEmail string
	LastChecked time.Time
	Status FormerNameStatus
}

type CharacterSearch struct{
	Found bool
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
	Information TibiaApiInformation `json:"information"`
}

type TibiaApiInformation struct {
	Status TibiaApiStatus `json:"status"`
}

type TibiaApiStatus struct {
	HttpCode int `json:"http_code"`
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
	
	found := true
	if j.Information.Status.ErrorCode == 20001 {
		found = false
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
		Found: found, 
		NameInput: name,
		Name: j.Character.Character.Name,
		FormerNames: j.Character.Character.FormerNames,
		World: j.Character.Character.World,
		Trackable: trackable,
	}, nil
}


var formerNames = []FormerName {
	{Name: "Mario", NotificationEmail: "rustydoggobytes@gmail.com", LastChecked: time.Now(), Status: expiring},
	{Name: "Djow tattoo", NotificationEmail: "rustydoggobytes@gmail.com", LastChecked: time.Now(), Status: expiring},
	{Name: "Luigi", NotificationEmail: "rustydoggobytes@gmail.com", LastChecked: time.Now(), Status: available },
	{Name: "Peach", NotificationEmail: "rustydoggobytes@gmail.com", LastChecked: time.Now(), Status: unavailable },
}


func runBackground(t *TibiaDataApi) {
			fmt.Println("running background")
	newArray := []FormerName{}
	for _, name := range(formerNames) {
		if name.Status != expiring  {
			newArray = append(newArray, name)
			continue
		}
			fmt.Println("checking name", name.Name)
		char, err := t.SearchCharacter(name.Name)
		if err != nil {
			fmt.Println(err)
		}

		if !char.Found {
			fmt.Println(name.Name, "is available")
			name.Status = available
		}

		if strings.ToLower(char.Name) == strings.ToLower(name.Name) {
			fmt.Println(name.Name, "is taken")
			name.Status = unavailable
		}
		name.LastChecked = time.Now()
		newArray = append(newArray, name)
		time.Sleep(1 * time.Second)
	}
	formerNames = newArray
	time.Sleep(5 * time.Minute)
}

func main() {
	t := TibiaDataApi{
		Url: "https://api.tibiadata.com",
	}

	go runBackground(&t)

	e := echo.New()
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
