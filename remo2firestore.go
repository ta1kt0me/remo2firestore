package main

import (
	"context"
	"encoding/json"
	firebase "firebase.google.com/go"
	"fmt"
	"google.golang.org/api/option"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type RemoDevice struct {
	Name              string    `json:"name"`
	ID                string    `json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	FirmwareVersion   string    `json:"firmware_version"`
	TemperatureOffset int       `json:"temperature_offset"`
	HumidityOffset    int       `json:"humidity_offset"`
	Users             []struct {
		ID        string `json:"id"`
		Nickname  string `json:"nickname"`
		Superuser bool   `json:"superuser"`
	} `json:"users"`
	NewestEvents struct {
		Hu struct {
			Val       int       `json:"val"`
			CreatedAt time.Time `json:"created_at"`
		} `json:"hu"`
		Il struct {
			Val       float64   `json:"val"`
			CreatedAt time.Time `json:"created_at"`
		} `json:"il"`
		Te struct {
			Val       float64   `json:"val"`
			CreatedAt time.Time `json:"created_at"`
		} `json:"te"`
	} `json:"newest_events"`
}

func main() {
	req, err := http.NewRequest("GET", "https://api.nature.global/1/devices", nil)
	if err != nil {
		return
	}

	token := fmt.Sprintf("Bearer %s", os.Getenv("REMO_TOKEN"))
	req.Header.Add("Authorization", token)

	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	var decoded []RemoDevice
	json.Unmarshal([]byte(body), &decoded)

	sa := option.WithCredentialsFile("./serviceAccount.json")
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	firebaseClient, err := app.Firestore(ctx)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	defer firebaseClient.Close()

	for _, data := range decoded {
		event := data.NewestEvents
		Humidity := event.Hu
		Illuminance := event.Il
		Temperature := event.Te

		device := map[string]interface{}{
			"id":   data.ID,
			"name": data.Name,
		}

		_, _, fbErr := firebaseClient.Collection("roomConditions").Add(ctx, map[string]interface{}{
			"temperature":          Temperature.Val,
			"temperatureCreatedAt": Temperature.CreatedAt,
			"illuminance":          Illuminance.Val,
			"illuminanceCreatedAt": Illuminance.CreatedAt,
			"humidity":             Humidity.Val,
			"humidityCreatedAt":    Humidity.CreatedAt,
			"measuredAt":           time.Now(),
			"device":               device,
		})

		if fbErr != nil {
			fmt.Printf("%v\n", fbErr)
			return
		}
	}

	fmt.Printf("%s\n", "Completed!")
}
