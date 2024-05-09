package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/robfig/cron/v3"
)

const at7AmEvery30Min = "TZ=Asia/Bangkok */30 7 * * *"

func main() {
	conn, err := amqp091.Dial(
		"amqp://" +
			os.Getenv("BROKER_USER") +
			":" + os.Getenv("BROKER_PASSWORD") +
			"@" + os.Getenv("BROKER_HOST") +
			":" + os.Getenv("BROKER_PORT") + "/")
	if err != nil {
		log.Fatal(err)
	}

	ch, err := conn.Channel()

	if err != nil {
		log.Fatal(err)
	}

	queue, err := ch.QueueDeclare("MAIL", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	coord, err := getLongLat()
	if err != nil {
		log.Fatal(err)
	}
	long := fmt.Sprintf("%f", coord["longitude"].(float64))
	lat := fmt.Sprintf("%f", coord["latitude"].(float64))

	c := cron.New()

	c.AddFunc(at7AmEvery30Min, func() {
		// Receive weather detail
		result, err := getWeatherDetail(lat, long)
		if err != nil {
			retry := 0
			for {
				time.Sleep(5 * time.Minute)
				retry++
				result, err = getWeatherDetail(lat, long)
				if err == nil {
					break
				}
				log.Print(err)
				if retry >= 3 {
					return
				}
			}
		}

		msg := "HCM city temprature at " + result["time"].(string) + " is " +
			fmt.Sprintf("%f", result["temperature_2m"].(float64)+10) + "°C"
		fmt.Println(msg)
		err = send(ch, queue.Name, msg)
		if err != nil {
			fmt.Println(err)
			retry := 0
			for {
				time.Sleep(5 * time.Minute)
				retry++
				err = send(ch, queue.Name, msg)
				if err == nil {
					break
				}
				log.Print(err)
				if retry >= 3 {
					return
				}
			}
		}

	})
	c.Start()
	var forever chan struct{}
	<-forever
}

func send(channel *amqp091.Channel, queue, msg string) error {
	context, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	return channel.PublishWithContext(context, "", queue, true, false, amqp091.Publishing{
		DeliveryMode: amqp091.Persistent,
		ContentType:  "text/plain",
		Body:         []byte(msg),
	})
}

func getWeatherDetail(lat, lang string) (map[string]interface{}, error) {
	url := "https://api.open-meteo.com/v1/cma?latitude=" + lat + "&longitude=" + lang +
		"&current=temperature_2m,apparent_temperature&timezone=Asia/Ho_Chi_Minh"
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var resBody map[string]interface{}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &resBody)
	if err != nil {
		return nil, err
	}
	current := resBody["current"].(map[string]interface{})
	return current, err
}

func getLongLat() (map[string]interface{}, error) {
	response, err := http.Get("https://geocoding-api.open-meteo.com/v1/search?name=saigon&count=10&language=en&format=json")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var resBody map[string]interface{}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &resBody)
	if err != nil {
		return nil, err
	}
	resArr := resBody["results"].([]interface{})
	for _, r := range resArr {
		if r.(map[string]interface{})["country"] == "Vietnam" {
			return r.(map[string]interface{}), nil
		}
	}
	return nil, fmt.Errorf("cannot get saigon")
}
