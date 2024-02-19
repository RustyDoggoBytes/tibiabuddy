package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
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

func runBackground(db *repositoryClient, t *TibiaDataApi, e *emailClient) {
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
				if newStatus == available {
					e.NotifyUserFormerNameIsAvailable(strings.Split(name.NotificationEmail, ","), name.Name)
				}
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
