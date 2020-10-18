package freenom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"freenom-ddns/internal/checkprofile"
	"github.com/tidwall/gjson"
	"golang.org/x/net/publicsuffix"
	"github.com/PuerkitoBio/goquery"
)

const (
	version         = "v0.0.5"
	timeout         = 34
	baseURL         = "https://my.freenom.com"
	refererURL      = "https://my.freenom.com/clientarea.php"
	loginURL        = "https://my.freenom.com/dologin.php"
	domainStatusURL = "https://my.freenom.com/domains.php?a=renewals"
	renewDomainURL  = "https://my.freenom.com/domains.php?submitrenewals=true"
	authKey         = "WHMC#tG5deHTGhfWtg"
	manageUrl       = "https://my.freenom.com/clientarea.php?managedns=ufly.ml&domainid=1097692681"
)

var (
	tokenREGEX       = regexp.MustCompile(`name="token"\svalue="(?P<token>[^"]+)"`)
	domainInfoREGEX  = regexp.MustCompile(`<tr><td>(?P<domain>[^<]+)<\/td><td>[^<]+<\/td><td>[^<]+<span class="[^"]+">(?P<days>\d+)[^&]+&domain=(?P<id>\d+)"`)
	loginStatusREGEX = regexp.MustCompile(`<li.*?Logout.*?<\/li>`)
	checkRenew       = regexp.MustCompile(`(?i)Order Confirmation`)
)

//Domain struct
type Domain struct {
	DomainName string
	Days       int
	ID         string
	RenewState int
}

//User data
type User struct {
	UserName   string
	PassWord   string
	CheckTimes int
	Domains    map[int]*Domain
}

// Freenom for opterate FreenomAPI
type Freenom struct {
	cookiejar *cookiejar.Jar
	client    *http.Client
	Users     map[int]*User
	token     string
	list map[int]map[string]string
}

var instance *Freenom
var once sync.Once

var (
	renewNo  int = 0
	renewYes int = 1
	renewErr int = 3
)
var (
	FreenomType_A 		string = "A"
	FreenomType_AAAA 	string = "AAAA"
	FreenomType_CNAME 	string = "CNAME"
	FreenomType_LOC 	string = "LOC"
	FreenomType_MX 		string = "MX"
	FreenomType_NAPTR 	string = "NAPTR"
	FreenomType_PR 		string = "PR"
	FreenomType_TXT 	string = "TXT"
)


// GetInstance is get  instance
func GetInstance() *Freenom {
	once.Do(func() {
		instance = &Freenom{}
		instance.Users = make(map[int]*User)
		instance.list = make(map[int]map[string]string)
		instance.cookiejar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		instance.client = &http.Client{Timeout: timeout * time.Second, Jar: instance.cookiejar}
	})
	return instance
}

// InputAccount input user data
func (f *Freenom) InputAccount(config *checkprofile.Config) *Freenom {
	for i, a := range config.Accounts {
		f.Users[i] = &User{
			UserName:   a.Username,
			PassWord:   a.Password,
			CheckTimes: 0,
		}
	}
	return f
}

// Login on Freenom
func (f *Freenom) Login(uid int) *Freenom {
	if f.checkLogin() {
		return f
	}

	_,status := sendRequest(
		"POST",
		loginURL,
		`{"headers":{
			"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.130 Safari/537.36",
			"Content-Type": "application/x-www-form-urlencoded",
			"Referer": "`+refererURL+`",
		},}`,
		url.Values{
			"username": {f.Users[uid].UserName},
			"password": {f.Users[uid].PassWord},
		}.Encode(),
	)
	log.Println("Post Login Status:",status)

	u, _ := url.Parse(baseURL)
	for _, authcook := range f.cookiejar.Cookies(u) {
		if authKey == authcook.Name && authcook.Value == "" {
			log.Println("AUTH error")
		}
		log.Println("log: cookie_id: ", authcook.Value)
	}
	return f
}

func (f *Freenom) checkLogin() bool {
	body,status := sendRequest(
		"GET",
		domainStatusURL,
		`{"headers":{
			"Referer": "`+refererURL+`"
		},}`,
		"",
	)
	log.Println("Get RenewDomains Status:",status)

	f.token = getParams(tokenREGEX, string(body))[0]["token"]
	if !loginStatusREGEX.Match(body) {
		log.Println("login state error no login")
		return false
	}
	f.list = getParams(domainInfoREGEX, string(body))
	return true
}


//RenewDomains is renew domain name
func (f *Freenom) RenewDomains(uid int) *Freenom {
	if !f.checkLogin() {
		return f
	}
	f.Users[uid].CheckTimes++
	f.Users[uid].Domains = make(map[int]*Domain)
	for i, d := range f.list {
		tmp, _ := d["days"]
		f.Users[uid].Domains[i] = &Domain{}
		f.Users[uid].Domains[i].Days, _ = strconv.Atoi(tmp)
		f.Users[uid].Domains[i].ID, _ = d["id"]
		f.Users[uid].Domains[i].DomainName, _ = d["domain"]
		if f.Users[uid].Domains[i].Days <= 14 {
			body,status := sendRequest(
				"POST",
				renewDomainURL,
				`{"headers":{
					"Referer": "https://my.freenom.com/domains.php?a=renewdomain&domain=`+f.Users[uid].Domains[i].ID+`",
					"Content-Type": "application/x-www-form-urlencoded",
				},}`,
				url.Values{
					"token":     {f.token},
					"renewalid": {f.Users[uid].Domains[i].ID},
					"renewalperiod[" + f.Users[uid].Domains[i].ID + "]": {"12M"},
					"paymentmethod": {"credit"},
				}.Encode(),
			)
			log.Println("Post RenewDomains Status:",status)

			if checkRenew.Match(body) {
				f.Users[uid].Domains[i].RenewState = renewYes
			} else {
				log.Fatalln("renew error")
				f.Users[uid].Domains[i].RenewState = renewErr
			}
		} else {
			f.Users[uid].Domains[i].RenewState = renewNo
		}
	}
	return f
}


//add record
func (f *Freenom) AddRecord(uid int,managedns string,name string,ip string) *Freenom {
	if !f.checkLogin() {
		return f
	}

	f.Users[uid].Domains = make(map[int]*Domain)
	for _, d := range f.list {
		if d["domain"] == managedns {
			requstUrl := refererURL + fmt.Sprintf("?managedns=%s&domainid=%s",managedns,d["id"])
			_,_ = sendRequest(
				"POST",
				requstUrl,
				`{"headers":{` +
					`"Referer": ` + requstUrl +`,
					"Content-Type": "application/x-www-form-urlencoded",
				},}`,
				url.Values{
					"token":     {f.token},
					"dnsaction": {"add"},
					"addrecord[0][name]":{name},
					"addrecord[0][type]":{FreenomType_A},
					"addrecord[0][ttl]":{"3600"},
					"addrecord[0][value]":{ip},//45.76.105.88
					"addrecord[0][priority]":{""},
					"addrecord[0][port]":{""},
					"addrecord[0][weight]":{""},
					"addrecord[0][forward_type]":{"1"},
					//"addrecord[1][name]":{name},
					//"addrecord[1][type]":{FreenomType_A},
					//"addrecord[1][ttl]":{"3600"},
					//"addrecord[1][value]":{ip},//45.76.105.88
					//"addrecord[1][priority]":{""},
					//"addrecord[1][port]":{""},
					//"addrecord[1][weight]":{""},
					//"addrecord[1][forward_type]":{"1"},

				}.Encode(),
			)
			log.Println("add success")
			return f
		}
	}
	log.Println("add error")
	return f
}

//delete record
func (f *Freenom) DeleteRecord(uid int,managedns string,name string,ip string) *Freenom {
	if !f.checkLogin() {
		return f
	}
	f.Users[uid].Domains = make(map[int]*Domain)
	for _, d := range f.list {
		if d["domain"] == managedns {
			requstUrl := refererURL + fmt.Sprintf("?managedns=%s&domainid=%s",managedns,d["id"])
			_,_ = sendRequest(
				"GET",
				requstUrl + fmt.Sprintf("&page=&records=A&dnsaction=delete&name=%s&value=%s&line=&ttl=3600&priority=&weight=&port=",name,ip),
				`{"headers":{` +
					`"Referer": ` + requstUrl + `,
					"Content-Type": "application/x-www-form-urlencoded",
				},}`,
				"",
			)
			log.Println("delete success")
			return f
		}
	}
	log.Println("delete error")
	return f
}

//update record
func (f *Freenom) UpdateRecord(uid int,managedns string,name string,ip string) *Freenom {
	if !f.checkLogin() {
		return f
	}
	f.Users[uid].Domains = make(map[int]*Domain)
	for _, d := range f.list {
		if d["domain"] == managedns {
			requstUrl := refererURL + fmt.Sprintf("?managedns=%s&domainid=%s",managedns,d["id"])
			body,status := sendRequest(
				"GET",
				requstUrl,
				`{"headers":{` +
					`"Referer": ` + requstUrl +`,
					"Content-Type": "application/x-www-form-urlencoded",
				},}`,
				"",
			)


			log.Println("Get UpdateRecord Status:",status)
			// Load the HTML document
			doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(body))
			if err != nil {
				log.Fatal(err)
			}

			urlParam := url.Values{}
			// Find the review items
			doc.Find("#recordslistform table tbody tr").Each(func(i int, s *goquery.Selection) {
				// For each item found, get the band and title
				lineElem  := s.Find(fmt.Sprintf("td input[name='records[%d][line]']",i))//.Attr("value")
				typeElem  := s.Find(fmt.Sprintf("td input[name='records[%d][type]']",i))//.Attr("value")
				nameElem  := s.Find(fmt.Sprintf("td input[name='records[%d][name]']",i))//.Attr("value")
				ttlElem   := s.Find(fmt.Sprintf("td input[name='records[%d][ttl]']",i))//.Attr("value")
				valueElem := s.Find(fmt.Sprintf("td input[name='records[%d][value]']",i))//.Attr("value")

				k1,_ := lineElem.Attr("name")
				v1,_ := lineElem.Attr("value")

				k2,_ := typeElem.Attr("name")
				v2,_ := typeElem.Attr("value")

				k3,_ := nameElem.Attr("name")
				v3,_ := nameElem.Attr("value")

				k4,_ := ttlElem.Attr("name")
				v4,_ := ttlElem.Attr("value")

				k5,_ := valueElem.Attr("name")
				v5,_ := valueElem.Attr("value")

				if len(name) > 0 {
					if name == v3 {
						v5 = ip
					}
				}


				urlParam[k1] = []string{v1}
				urlParam[k2] = []string{v2}
				urlParam[k3] = []string{v3}
				urlParam[k4] = []string{v4}
				urlParam[k5] = []string{v5}
			})

			urlParam["token"] = []string{f.token}
			urlParam["dnsaction"] = []string{"modify"}
			_,status = sendRequest(
				"POST",
				requstUrl,
				`{"headers":{` +
					`"Referer": ` + requstUrl +`,
					"Content-Type": "application/x-www-form-urlencoded",
				},}`,
				urlParam.Encode(),
			)
			log.Println("Get UpdateRecord3 Status:",status)
			log.Println("update success")
			return f
		}
	}
	log.Println("update error")
	return f
}


func (f *Freenom) GetIp() *Freenom {
	body,_ := sendRequest(
		"GET",
		"http://v4v6.ipv6-test.com/api/myip.php?json",
		"",
		"",
	)

	v := make(map[string]string)
	json.Unmarshal(body,&v)
	log.Println("ip:",v["address"]," ",v["proto"])

	return f
}

func (f *Freenom) GetV4Ip() *Freenom {
	body,_ := sendRequest(
		"GET",
		"http://v4.ipv6-test.com/api/myip.php?json",
		"",
		"",
	)
	v := make(map[string]string)
	json.Unmarshal(body,&v)
	log.Println("ipv4:",v["address"]," ",v["proto"])
	return f
}

func (f *Freenom) GetV6Ip() *Freenom {
	body,_ := sendRequest(
		"GET",
		"http://v6.ipv6-test.com/api/myip.php?json",
		"",
		"",
	)
	v := make(map[string]string)
	json.Unmarshal(body,&v)
	log.Println("ipv6:",v["address"]," ",v["proto"])
	return f
}

func (f *Freenom) GetIp2() *Freenom {
	body,_ := sendRequest(
		"GET",
		"http://ipv4.icanhazip.com/",
		"",
		"",
	)
	log.Println("ipv2:",body)
	return f
}
//api.ipify.org

/**
 * Parses url with the given regular expression and returns the
 * group values defined in the expression.
 *
 */
func getParams(regEx *regexp.Regexp, url string) (paramsMaps map[int]map[string]string) {
	match := regEx.FindAllStringSubmatch(url, -1)
	paramsMaps = map[int]map[string]string{}

	for j := 0; j < len(match); j++ {
		paramsMaps[j] = make(map[string]string)
		for i, name := range regEx.SubexpNames() {
			if i > 0 && i <= len(match[j]) {
				paramsMaps[j][name] = match[j][i]
			}
		}
	}
	return
}

/**
 * sendRequest just all in one
 */
func sendRequest(method, furl, headers, datas string) ([]byte,string) {
RETRY:
	req, err := http.NewRequest(method, furl, strings.NewReader(datas))
	if err != nil {
		log.Fatal("Create http request error", err)
	}
	f := GetInstance()
	if headers != "" {
		headerObj := gjson.Get(headers, "headers")
		headerObj.ForEach(func(key, value gjson.Result) bool {
			req.Header.Add(key.String(), value.String())
			return true
		})
	}
	resp, err := f.client.Do(req)
	if err != nil {
		log.Println("http response error: ", err)
		time.Sleep(3 * time.Second)
		goto RETRY
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return body,resp.Status
}
