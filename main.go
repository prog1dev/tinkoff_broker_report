package main

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strings"
  "time"
)

const (
  token    = "your_token"
  timeout  = time.Second * 3
  from     = "2020-05-05"
  to       = "2020-05-16"
  url      = "https://api-invest.tinkoff.ru/openapi/operations?from=" + from + "T18%3A38%3A33.131642%2B03%3A00&to=" + to + "T18%3A38%3A33.131642%2B03%3A00"
  filepath = "/Users/filenkoivan/Downloads/broker_report.csv"
)

func main() {
  client := &http.Client{
    Timeout: timeout,
  }

  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    log.Fatalf("Can't create operations http request: %s", err)
  }

  req.Header.Add("Authorization", "Bearer "+token)
  resp, err := client.Do(req)
  if err != nil {
    log.Fatalf("Can't send operations request: %s", err)
  }

  defer resp.Body.Close()

  if resp.StatusCode != http.StatusOK {
    log.Fatalf("operations, bad response code '%s' from '%s'", resp.Status, url)
  }

  respBody, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    log.Fatalf("Can't read operations response: %s", err)
  }

  var operationsResp OperationsResponse
  err = json.Unmarshal(respBody, &operationsResp)
  if err != nil {
    log.Fatalf("Can't unmarshal operations response: '%s' \nwith error: %s", string(respBody), err)
  }

  if strings.ToUpper(operationsResp.Status) != "OK" {
    log.Fatalf("operations failed, trackingId: '%s'", operationsResp.TrackingID)
  }

  figis := make(map[string]string)

  for _, v := range operationsResp.Payload.Operations {
    figis[v.Figi] = ""
  }

  figiToCompany := make(map[string]Company)
  for k := range figis {
    if k != "" {
      figiToCompany[k] = getCompany(client, k)
    }
  }

  result := make([]string, 0)
  lines := make([]string, 0)
  count := 0

  result = append(lines, fmt.Sprintf("Date,Time,Symbol,Quantity,Price,Buy/Sell,Commission"))
  for _, v := range operationsResp.Payload.Operations {
    if v.OperationType != "BrokerCommission" && v.OperationType != "PayIn" && v.OperationType != "ServiceCommission" && v.Figi != "BBG0013HGFT4" && v.Status == "Done" {
      date := strings.Split(strings.Split(v.Date, "T")[0], "-")
      day := date[2]
      month := date[1]
      year := date[0]

      dateStr := fmt.Sprintf("%v/%v/%v", month, day, year)
      timeStr := strings.Split(strings.Split(v.Date, "T")[1], ".")[0]

      quantity := int32(0)

      for _, trade := range v.Trades {
        quantity += trade.Quantity
      }

      line := fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v", dateStr, timeStr, figiToCompany[v.Figi].Ticker, quantity, v.Price, v.OperationType, v.Commision.Value*-1)

      lines = append(lines, line)
      count += 1
    }
  }

  revertSlice(lines)
  for _, v := range lines {
    result = append(result, v)
  }
  createFile(result, filepath)
}

func getCompany(client *http.Client, figi string) Company {
  url := "https://www.openfigi.com/search/query?num_rows=20&simpleSearchString=" + figi + "&start=0"
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    log.Fatalf("Can't create openfigi.com http request: %s", err)
  }

  req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36")
  resp, err := client.Do(req)
  if err != nil {
    log.Fatalf("Can't send openfigi.com request: %s", err)
  }

  defer resp.Body.Close()

  if resp.StatusCode != http.StatusOK {
    log.Fatalf("openfigi.com, bad response code '%s' from '%s'", resp.Status, url)
  }

  respBody, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    log.Fatalf("Can't read openfigi.com response: %s", err)
  }

  var openFigiResp OpenFigiResponse
  err = json.Unmarshal(respBody, &openFigiResp)
  if err != nil {
    log.Fatalf("Can't unmarshal openfigi.com response: '%s' \nwith error: %s", string(respBody), err)
  }

  for _, v := range openFigiResp.Result {
    if v.Id == figi {
      return v
    }
  }

  return openFigiResp.Result[0]
}

type OpenFigiResponse struct {
  Result []Company `json:"result"`
}

type Company struct {
  Id     string `json:"id"`
  Name   string `json:"DS002_sd"`
  Ticker string `json:"DS156_sd"`
  Type   string `json:"DS213_sd"`
}

type OperationsResponse struct {
  TrackingID string  `json:"trackingId"`
  Status     string  `json:"status"`
  Payload    Payload `json:"payload"`
}

type Payload struct {
  Operations []Operation `json:"operations"`
}

type Operation struct {
  Id             string    `json:"id"`
  Status         string    `json:"status"`
  OperationType  string    `json:"operationType"`
  Date           string    `json:"date"`
  InstrumentType string    `json:"instrumentType"`
  IsMarginCall   bool      `json:"isMarginCall"`
  Figi           string    `json:"figi"`
  Quantity       int32     `json:"quantity"`
  Price          float64   `json:"price"`
  Payment        float64   `json:"payment"`
  Currency       string    `json:"currency"`
  Commision      Commision `json:"commission"`
  Trades         []Trade   `json:"trades"`
}

type Commision struct {
  Currency string  `json:"currency"`
  Value    float64 `json:"value"`
}

type Trade struct {
  Id       string  `json:"tradeId"`
  Date     string  `json:"date"`
  Quantity int32   `json:"quantity"`
  Price    float64 `json:"price"`
}

func createFile(array []string, filepath string) error {
  f, err := os.Create(filepath)
  if err != nil {
    return fmt.Errorf("error creating %s file: %v", filepath, err)
  }
  defer func() {
    if err := f.Close(); err != nil {
      log.Printf("error closing %s file: %v", filepath, err)
    }
  }()

  result := make([]string, 0)
  for _, v := range array {
    if v != "" {
      result = append(result, strings.TrimSpace(v))
    }
  }

  log.Printf("result len: %v", len(result))
  if _, err := f.WriteString(strings.Join(result, "\n")); err != nil {
    return fmt.Errorf("error writing to %s file: %v", filepath, err)
  }

  log.Printf("the file %s has been written\n", filepath)

  return nil
}

func revertSlice(slice []string) {
  for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
    slice[i], slice[j] = slice[j], slice[i]
  }
}
