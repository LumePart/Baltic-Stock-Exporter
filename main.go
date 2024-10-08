package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"flag"

	"github.com/xuri/excelize/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)


var promReg *prometheus.Registry
var daemon bool

func downloadFile(url string, localFilePath string) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("failed getting response from website: %s", err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatalf("failed to download file: %s", response.Status)
	}

	file, err := os.Create(localFilePath)
	if err != nil {
		log.Fatalf("failed to create file: %s", err.Error())
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Fatalf("failed to copy response to file: %s", err.Error())
	}
}

func readAllStocks() [][]string { // Reads an Excel file named and returns a 2D string slice representing the "Shares" sheet data.
	stock_url := "https://nasdaqbaltic.com/statistics/en/shares?download=1"
	file_path := "/tmp/shares.xlsx"
	downloadFile(stock_url,file_path)


	f, err := excelize.OpenFile(file_path)
    if err != nil {
        fmt.Println(err)
    }
    defer func() {
        if err := f.Close(); err != nil {
            fmt.Println(err)
        }
    }()

    rows, err := f.GetRows("Shares")
    if err != nil {
        fmt.Println(err)
    }
    return rows
}

func getStockLabels(row []string) []string { // Extracts specific fields from a row of stock data to create and return a list of metric labels.
	tags := make([]string, 0, 6)
	for _, val := range []int{0, 1, 4, 18, 19} {
		tags = append(tags, strings.ToLower(row[val]))
	}
	tags = append(tags, strings.Split(row[5], "/")[1])
	return tags
}

func main() {

	flag.Bool("d", true, "run exporter as a service, scrapes data 24/7")
	flag.Parse()

	daemon = flag.NFlag() == 1
	if daemon {
		fmt.Println("running exporter as service")
	}

	promReg = prometheus.NewRegistry()
	stocks := regStocks()

	prometheus.DefaultRegisterer = promReg
	prometheus.DefaultGatherer = promReg


	collect := &CustomCollector{
		stockPriceMetric: stocks.stockPriceMetric,
		stockTradeMetric: stocks.stockTradeMetric,
		stockVolumeMetric: stocks.stockVolumeMetric,
		stockTurnoverMetric: stocks.stockTurnoverMetric,
	}
	register(collect)

	http.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))

	err := http.ListenAndServe(":33171", nil)
	if err != nil {
		log.Fatalf("failed to start http server: %s", err.Error())
	}
}
