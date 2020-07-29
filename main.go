package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	api1URL = "https://forum.gamer.com.tw/C.php?page=81000&bsn=60076&snA=5037743"
	api2URL = "https://forum.gamer.com.tw/C.php?page=81000&bsn=60076&snA="
)

var (
	// init api 1 req time
	api1LastReqTime = time.Now()
	api1Img         []byte

	// init api 2 req time
	api2LastReqTime = time.Now()
	arBahaUser      = make(map[string]*bahaUser)
)

type bahaUser struct {
	ip     string
	ip2    string
	id     string //帳號
	name   string //名稱
	gp     string
	level  string //等級
	career string //職業
	race   string //種族
}

func main() {

	//建立 server
	http.HandleFunc("/Reply/t02", func(w http.ResponseWriter, r *http.Request) {

		//回傳圖片
		w.Header().Set("Content-Type", "image/svg+xml")

		var ip = getRemoteIP(r)     // 取得IP
		var sn = r.FormValue("snA") //取得get的參數

		now := time.Now() //目前時間

		//如果間隔大於3秒
		if api2LastReqTime.Sub(now) > 3*time.Second {
			api2LastReqTime = now //更新最後請求時間
			fmt.Println("重新請求")

			//請求網址
			urlStr := api2URL + sn
			var rawCookies string = ""

			resp := doGet(urlStr, map[string]string{}, rawCookies)
			dom, _ := goquery.NewDocumentFromReader(strings.NewReader(resp))

			ar := dom.Find(".c-section")
			addBahaUser(ar)

			//如果有上一頁的話，就連同上一頁的內容都抓
			pagebtnA := dom.Find(".BH-pagebtnA").Eq(0).Find("a")

			if pagebtnA.Length() >= 3 {
				href, _ := pagebtnA.Eq(pagebtnA.Length() - 2).Attr("href")
				href = "https://forum.gamer.com.tw/C.php" + href
				fmt.Println("上一頁：" + href)

				//抓取上一頁的內容
				previousPage := href
				rawCookies2 := ""
				resp2 := doGet(previousPage, map[string]string{}, rawCookies2)
				previousDom, _ := goquery.NewDocumentFromReader(strings.NewReader(resp2))
				ar2 := previousDom.Find(".c-section")
				addBahaUser(ar2)
			}

		}

		//取得使用者的IP前3個數字，用來跟巴哈的IP進行比對
		var userIP string
		ipSlice := strings.Split(ip, ".")
		var imgBase64 string
		if len(ipSlice) == 4 {
			userIP = ipSlice[0] + "." + ipSlice[1] + "." + ipSlice[2]
		}

		var txt = "你根本沒回文吧"
		if _, ok := arBahaUser[userIP]; ok {
			txt = "你是" + arBahaUser[userIP].race + arBahaUser[userIP].career + "沒錯吧"
			imgBase64 = img01
		} else {
			imgBase64 = img02
		}

		io.WriteString(
			w,
			`<?xml version="1.0" encoding="utf-8"?>
			<svg version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px"
				 viewBox="0 0 400 275" style="enable-background:new 0 0 400 275;" xml:space="preserve">
			<style type="text/css">
				.st0{fill:#FFFFFF;}
				.st1{font-family:'AdobeMingStd-Light-B5pc-H';}
				.st2{font-size:28px;}
			</style>
			<g id="圖層_2">
				<rect width="400" height="275"/>
			</g>
			<g id="圖層_3">
				<text transform="matrix(1 0 0 1 74 261)" class="st0 st1 st2">`+txt+`</text>
			</g>
			<g id="圖層_1">				
					<image style="overflow:visible;" width="300" height="169" xlink:href="`+imgBase64+`" transform="matrix(1.3333 0 0 1.3333 0 0)">
				</image>
			</g>
			</svg>
			`,
		)

		/*var sum = ""
		for key, value := range arBahaUser {
			sum += key + " " + value.ip + "<br>" +
				"id:" + value.id + "<br>" +
				"career:" + value.career + "<br>" +
				"level:" + value.level + "<br>" +
				"race:" + value.race + "<br>" +
				"gp:" + value.gp + "<hr>"
		}*/
		/*io.WriteString(
			w,
			`<doctype html>
			<html>
				<head>
					<title>2</title>
					<meta charset="utf-8" />
				</head>
				<body>
					<img src= "`+img01+`">
					<h3>`+txt+" "+ip+sum+`</h3>
				</body>
			</html>`,
		)*/

	})

	//建立 server
	http.HandleFunc("/Reply/t01.png", func(w http.ResponseWriter, r *http.Request) {

		now := time.Now() //目前時間

		//如果間隔低於3秒
		if api1LastReqTime.Sub(now) < 3*time.Second {
			if api1Img != nil {
				//使用上次的圖片進行回傳
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Content-Length", strconv.Itoa(len(api1Img)))
				if _, err := w.Write(api1Img); err != nil {
					log.Println("無法寫圖像")
				}
				return
			}
		}

		//請求網址
		var rawCookies string
		resp := doGet(api1URL, map[string]string{}, rawCookies)
		dom, _ := goquery.NewDocumentFromReader(strings.NewReader(resp))

		//取得最後一個回文的帳號
		users := dom.Find(".userid")
		userID := users.Eq(users.Length() - 1).Text()
		userAvatar := getAvatarURL(userID)

		//下載圖片
		api1Img = downloadImg(userAvatar).Bytes()

		//回傳圖片
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(api1Img)))
		if _, err := w.Write(api1Img); err != nil {
			log.Println("無法寫圖像")
		}

		fmt.Println("重新請求")
		api1LastReqTime = time.Now() //更新最後請求時間

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(
			"Content-Type",
			"text/html",
		)

		io.WriteString(
			w,
			`<doctype html>
			<html>
				<head>
					<title>你在期待什麼啦</title>
					<meta charset="utf-8" />
				</head>
				<body>
					<h1>你在期待什麼啦？</h1>
				</body>
			</html>`,
		)
	})

	var port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	//在指定的port上面進行啟用server
	http.ListenAndServe(":"+port, nil)
}

//-------------------

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
)

// getRemoteIP 返回遠程客戶端的 IP，如 192.168.1.1
func getRemoteIP(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get(XRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get(XForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}

	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}

	return remoteAddr
}

//
// 解析網頁，儲存到 arBahaUser 裡面
//
func addBahaUser(ar *goquery.Selection) {

	ar.Each(func(i int, selection *goquery.Selection) {
		if selection.Find(".edittime").Length() > 0 {

			userIP, _ := selection.Find(".edittime").Eq(0).Attr("data-hideip") //記錄於巴哈的IP
			userID := selection.Find(".userid").Eq(0).Text()                   //帳號
			userGP, _ := selection.Find(".usergp").Eq(0).Attr("title")         //gp
			userLevel := selection.Find(".userlevel").Eq(0).Text()             //等級
			userCareer, _ := selection.Find(".usercareer").Eq(0).Attr("title") //職業
			userRace, _ := selection.Find(".userrace").Eq(0).Attr("title")     //種族

			newBahaUser := &bahaUser{
				ip:     strings.Replace(userIP, ".xxx", "", -1),
				id:     userID,
				gp:     userGP,
				level:  userLevel,
				career: userCareer,
				race:   userRace,
			}

			arBahaUser[userIP] = newBahaUser
		}
	})
}

//
//
//
func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

//
// 請求網頁，並且回傳已解析的html物件
//
func doGet(urlStr string, queryData map[string]string, rawCookies string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", urlStr, nil)
	checkErr(err)
	q := req.URL.Query()
	for k, v := range queryData {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	header := http.Header{}
	header.Add("Cookie", rawCookies)
	header.Add("Content-Type", "application/x-www-form-urlencoded; param=value")
	req.Header = header

	resp, err := client.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	checkErr(err)
	return string(b)
}

//
// 取得勇照的網址
//
func getAvatarURL(userID string) string {
	//https://avatar2.bahamut.com.tw/avataruserpic/j/e/jeff60316377/jeff60316377.png
	userID = strings.ToLower(userID)
	var t1 string = string(userID[0])
	var t2 string = string(userID[1])
	url := fmt.Sprintf("https://avatar2.bahamut.com.tw/avataruserpic/%s/%s/%s/%s.png", t1, t2, userID, userID)
	return url
}

//
// 下載圖片
//
func downloadImg(url string) *bytes.Buffer {

	//通過http請求獲取圖片的流文件
	var resp, _ = http.Get(url)
	var body, _ = ioutil.ReadAll(resp.Body)
	var buffer *bytes.Buffer = new(bytes.Buffer)

	io.Copy(buffer, bytes.NewReader(body))
	return buffer
}
