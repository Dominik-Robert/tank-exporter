package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

type TankerKoenig struct {
	Ok       bool   `json:"ok"`
	License  string `json:"license"`
	Data     string `json:"data"`
	Status   string `json:"status"`
	Stations []struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Brand       string  `json:"brand"`
		Street      string  `json:"street"`
		Place       string  `json:"place"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		Dist        float64 `json:"dist"`
		Diesel      float64 `json:"diesel"`
		E5          float64 `json:"e5"`
		E10         float64 `json:"e10"`
		IsOpen      bool    `json:"isOpen"`
		HouseNumber string  `json:"houseNumber"`
		PostCode    int     `json:"postCode"`
	} `json:"stations"`
}

var (
	data map[string]prometheus.Gauge
)

func main() {
	viper.SetDefault("latitude", 51.575710)
	viper.SetDefault("longitude", 7.209179)
	viper.SetDefault("radius", 2)
	viper.SetDefault("apikey", "")

	viper.AutomaticEnv()

	data = make(map[string]prometheus.Gauge)
	initialize()

	fmt.Println("latitude:", viper.GetFloat64("latitude"))
	fmt.Println("longitude:", viper.GetFloat64("longitude"))
	fmt.Println("radius:", viper.GetFloat64("radius"))
	fmt.Println("apikey:", viper.GetString("apikey"))
	fmt.Println("URL:", fmt.Sprintf("https://creativecommons.tankerkoenig.de/json/list.php?lat=%f&lng=%f&apikey=%s&type=all&rad=%d", viper.GetFloat64("latitude"), viper.GetFloat64("longitude"), viper.GetString("apiKey"), viper.GetInt("radius")))

	router := gin.New()
	router.Use(Middleware())

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.Run(":2112")
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		refresh()
	}
}

func refresh() {
	url := fmt.Sprintf("https://creativecommons.tankerkoenig.de/json/list.php?lat=%f&lng=%f&apikey=%s&type=all&rad=%f", viper.GetFloat64("latitude"), viper.GetFloat64("longitude"), viper.GetString("apikey"), viper.GetFloat64("radius"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Cannot create request", err)
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("cannot make request", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Println("cannot read data from body", err)
	}
	var tankerkoenig TankerKoenig
	err = json.Unmarshal(body, &tankerkoenig)

	if err != nil {
		log.Println("cannot unmarshal data", err)
	}

	for _, value := range tankerkoenig.Stations {
		data[value.ID+"_Diesel"].Set(value.Diesel)
		data[value.ID+"_E10"].Set(value.E10)
		data[value.ID+"_E5"].Set(value.E5)
	}
}

func initialize() {
	url := fmt.Sprintf("https://creativecommons.tankerkoenig.de/json/list.php?lat=%f&lng=%f&apikey=%s&type=all&rad=%f", viper.GetFloat64("latitude"), viper.GetFloat64("longitude"), viper.GetString("apiKey"), viper.GetFloat64("radius"))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Println(err)
	}
	var tankerkoenig TankerKoenig
	err = json.Unmarshal(body, &tankerkoenig)

	if err != nil {
		log.Println(err)
	}

	for _, value := range tankerkoenig.Stations {
		data[value.ID+"_E5"] = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "GasStation",
				Help: "The gas station values",
				ConstLabels: prometheus.Labels{
					"brand":  value.Brand,
					"name":   value.Name,
					"type":   "E5",
					"street": value.Street,
					"number": value.HouseNumber,
					"place":  value.Place,
				},
			},
		)

		data[value.ID+"_E10"] = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "GasStation",
				Help: "The gas station values",
				ConstLabels: prometheus.Labels{
					"brand":  value.Brand,
					"name":   value.Name,
					"type":   "E10",
					"street": value.Street,
					"number": value.HouseNumber,
					"place":  value.Place,
				},
			},
		)

		data[value.ID+"_Diesel"] = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "GasStation",
				Help: "The gas station values",
				ConstLabels: prometheus.Labels{
					"brand":  value.Brand,
					"name":   value.Name,
					"type":   "Diesel",
					"street": value.Street,
					"number": value.HouseNumber,
					"place":  value.Place,
				},
			},
		)

		data[value.ID+"_E5"].Set(float64(time.Now().Unix()))
		data[value.ID+"_E10"].Set(value.E10)
		data[value.ID+"_Diesel"].Set(value.Diesel)

		prometheus.MustRegister(data[value.ID+"_E5"])
		prometheus.MustRegister(data[value.ID+"_E10"])
		prometheus.MustRegister(data[value.ID+"_Diesel"])
	}
}
