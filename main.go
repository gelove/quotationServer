package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	socketio "github.com/googollee/go-socket.io"
)

/**
			flutter		reacgt-native
荣耀v10			1ms			15ms
三星S6			1ms			25ms
红米2A			3ms			60ms
*/

// INTERVAL 数据推送时间间隔(毫秒)
const INTERVAL = 10

// NUMBER 每次更新的行情数量
const NUMBER = 10

// PORT 服务端口号
const PORT = ":12345"

var sockets map[string]socketio.Socket

var initData []quotation

var tickData = make([]quotation, NUMBER)

type quotation struct {
	ContractNo      string  `json:"contractNo"`
	ContractName    string  `json:"contractName"`
	ChangeRate      float64 `json:"changeRate,string"`
	ChangeValue     float64 `json:"changeValue,string"`
	LastPrice       float64 `json:"lastPrice,string"`
	PreClosingPrice float64 `json:"preClosingPrice,string"`
}

func init() {
	list, err := readFile("data.json")
	if err != nil {
		fmt.Println("init data failed: ", err.Error())
	}
	initData = list
}

func readFile(filename string) ([]quotation, error) {
	var list []quotation
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("ReadFile failed: ", err.Error())
		return nil, err
	}
	if err := json.Unmarshal(bytes, &list); err != nil {
		fmt.Println("Unmarshal failed: ", err.Error())
		return nil, err
	}
	return list, nil
}

func shuffle(vals []quotation) []quotation {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	n := len(vals)
	ret := make([]quotation, n)
	perm := r.Perm(n)
	for i, randIndex := range perm {
		ret[i] = vals[randIndex]
	}
	return ret
}

func main() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	dataTicker := time.NewTicker(time.Millisecond * INTERVAL)
	go func() {
		for {
			select {
			case <-dataTicker.C:
				rs := shuffle(initData)[:NUMBER]
				data := make([]quotation, NUMBER)
				for i, item := range rs {
					x := rand.Intn(100)
					isNegative := x/2 == 1
					rate := float64(x) / 1000
					if isNegative {
						rate = rate * -1
					}
					item.ChangeRate = rate
					item.ChangeValue = item.PreClosingPrice * rate
					LastPrice := item.PreClosingPrice * (1 + rate)
					item.LastPrice, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", LastPrice), 64)
					data[i] = item
				}
				tickData = data
			}
		}
	}()

	senderTicker := time.NewTicker(time.Millisecond * INTERVAL)
	go func() {
		for {
			select {
			case <-senderTicker.C:
				fmt.Printf("ticked at %v %v %v\n", time.Now(), tickData[0].ContractName, tickData[0].LastPrice)
				server.BroadcastTo("quotation", "mdDataEvent", tickData)
			}
		}
	}()

	server.On("connection", func(soc socketio.Socket) {
		log.Println("on connection", soc.Id())
		// sockets[soc.Id()] = soc

		soc.On("registerEvent", func(msg string) {
			log.Println("on registerEvent", msg)
			soc.Join("quotation")
			// 单个
			soc.Emit("mdListEvent", initData)
		})

		soc.On("disconnection", func() {
			log.Println("on disconnect")
			soc.Leave("quotation")
		})
	})

	server.On("error", func(soc socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	log.Println("Serving at localhost", PORT)
	log.Fatal(http.ListenAndServe(PORT, nil))
}
