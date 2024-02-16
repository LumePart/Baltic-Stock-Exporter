package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

type CustomCollector struct {
    stockPriceMetric *prometheus.Desc
	stockTradeMetric *prometheus.Desc
	stockVolumeMetric *prometheus.Desc
	stockTurnoverMetric *prometheus.Desc
}

type MarketState struct {}

func (c *CustomCollector) Describe(ch chan<- *prometheus.Desc) {
    ch <- c.stockPriceMetric
	ch <- c.stockTradeMetric
	ch <- c.stockVolumeMetric
	ch <- c.stockTurnoverMetric
}

func (c *CustomCollector) Collect(ch chan<- prometheus.Metric) {

	loc, err := time.LoadLocation("Europe/Tallinn") // Load timezone in the baltics
	if err != nil {
		log.Printf("failed to load location: %s", err)
	}

	checkTime(c, loc)
	
	data := readAllStocks()

	for _, row := range data[1:] {

		if len(row) != 20 { // Ignore companies that have don't have industry and supersector defined
			continue
		}
		
		industry := row[18]
		superSector := row[19]
		price, _ := strconv.ParseFloat(row[11], 64)
		trades, _ := strconv.ParseFloat(row[15], 64)
		volume, _ := strconv.ParseFloat(row[16], 64)
		turnover, _ := strconv.ParseFloat(row[17], 64)

    t := time.Now().In(loc).Add(-15 * time.Minute)
    stockPrice := prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(c.stockPriceMetric, prometheus.GaugeValue, price, row[0], row[1], row[4], industry, superSector, row[5]))
	stockTrades := prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(c.stockTradeMetric, prometheus.CounterValue, trades, row[0], row[1], row[4], industry, superSector, row[5]))
	stockVolume := prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(c.stockVolumeMetric, prometheus.CounterValue, volume, row[0], row[1], row[4], industry, superSector, row[5]))
	stockTurnover := prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(c.stockTurnoverMetric, prometheus.CounterValue, turnover, row[0], row[1], row[4], industry, superSector, row[5]))


    ch <- stockPrice
	ch <- stockTrades
	ch <- stockVolume
	ch <- stockTurnover
	}
}

func regStocks()  *CustomCollector { // Registers stock metrics
	rows := readAllStocks()
	labels := getStockLabels(rows[0])
    collector := &CustomCollector{
        stockPriceMetric: prometheus.NewDesc(
            "stock_price",
            "Holds last price of a stock",
            labels,
            nil,
        ),
		stockTradeMetric: prometheus.NewDesc(
            "stock_trades",
            "Total trades for a stock in a day",
            labels,
            nil,
        ),
		stockVolumeMetric: prometheus.NewDesc(
			"stock_volume",
			"Volume count for a stock in a day",
			labels,
			nil,
		),
		stockTurnoverMetric: prometheus.NewDesc(
			"stock_turnover",
			"Total turnover for a stock in a day",
			labels,
			nil,
		),
    }
	return collector
}

func checkTime(c prometheus.Collector, loc *time.Location) {
	
	currentTime := time.Now().In(loc) // Get time in the Baltics
	yyyy, mm, dd := currentTime.Date()

	targetTime := time.Date(yyyy, mm, dd, 16, 15, 0, 0, currentTime.Location())

	// Compare current time with the target time
	if currentTime.After(targetTime) || currentTime.Weekday() == time.Sunday || currentTime.Weekday() == time.Saturday  { // If the current time in the baltics is later than market close time, exit the program
		prom_reg.Unregister(c)
		fmt.Println("Metrics unregistered.. Exiting..")
		os.Exit(0)
	}

}

func register(metric prometheus.Collector) {
	prometheus.MustRegister(metric)

}