package main

import (
	"bytes"
	"encoding/json"
	"github.com/supar/gosbss"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ClientsList struct {
	// В случае, если запрос будет отклонен по причине
	// авторизации, то вернется заполненный challenge
	Challenge int `json:"challenge"`
	// Текст ошибки
	Error string `json:"error"`
	// Список файлов
	Users []*User `json:"results"`
	// Статус ответа: успех/ошибка
	Success bool `json:"success"`
}

type ClientsETag struct {
	// В случае, если запрос будет отклонен по причине
	// авторизации, то вернется заполненный challenge
	Challenge int `json:"challenge"`
	// Текст ошибки
	Error string `json:"error"`
	// тэг
	ETag string `json:"etag"`
	// Статус ответа: успех/ошибка
	Success bool `json:"success"`
}

type User struct {
	Id           int       `json:"id" vcard:"-"`
	Name         string    `json:"name" vcard:"fn"`
	Type         int       `json:"type" vcard:"-"`
	Organization string    `json:"-" vcard:"org"`
	FullName     []string  `json:"-" vcard:"n,inline"`
	Email        *Email    `json:"email" vcard:"email,separator(;),inline,iteminline(:)"`
	Uid          string    `json:"-" vcard:"uid"`
	Classname    string    `json:"classname" vcard:"categories"`
	Updated      time.Time `json:"update" vcard:"-"`
}

type Email struct {
	Type  string `vcard:",separator(=)"`
	Value string `vcard:",omitname"`
}

type ClientsRequest struct {
	Async    int    `url:"async"`
	Inc      string `url:"inc"`
	Cmd      string `url:"cmd"`
	Contacts int    `url:"contacts"`
	Classid  int    `url:"classid"`
	Uid      int    `url:"uid"`
}

type SbssIface interface {
	GetClients(string, string, *ClientsRequest) (*ClientsList, error)
	GetClientsETag(string, string) (*ClientsETag, error)
}

// Расширение http клиента из пакета gosbss
// Дополнен возможностью хранить URL для
// исходящих запросов к SBSS серверу
type SbssClient struct {
	*gosbss.Client
	// URL SBSS сервера
	server string
}

// Создай http клиент для работы с SBSS сервером
func NewSbssClient(url string) *SbssClient {
	return &SbssClient{
		Client: gosbss.NewClient(),
		server: url,
	}
}

func (this *SbssClient) GetClients(user, pass string, filter *ClientsRequest) (clients *ClientsList, err error) {
	var (
		auth *gosbss.AuthRequest
		form *bytes.Buffer
		res  *http.Response
	)

	// Аутентифицируй
	auth = gosbss.NewAuthRequest(user, pass)
	if err = this.Client.Login(this.server, auth); err != nil {
		return
	}

	if filter == nil {
		filter = &ClientsRequest{}
	}

	filter.Async = 1
	filter.Inc = "clients"
	filter.Cmd = "get"

	if form, err = gosbss.EncodeForm(&filter); err != nil {
		return
	}

	if res, err = this.post(form); err != nil {
		return
	}

	// Read response
	clients = &ClientsList{}
	if err = gosbss.ReadResponse(res, &clients); err != nil {
		return
	}

	return
}

func (this *SbssClient) GetClientsETag(user, pass string) (etag *ClientsETag, err error) {
	var (
		auth *gosbss.AuthRequest
		form *bytes.Buffer
		res  *http.Response
	)

	// Аутентифицируй
	auth = gosbss.NewAuthRequest(user, pass)
	if err = this.Client.Login(this.server, auth); err != nil {
		return
	}

	if form, err = gosbss.EncodeForm(&ClientsRequest{
		Async: 1,
		Inc:   "clients",
		Cmd:   "getclientsetag",
	}); err != nil {
		return
	}

	if res, err = this.post(form); err != nil {
		return
	}

	// Read response
	etag = &ClientsETag{}
	if err = gosbss.ReadResponse(res, &etag); err != nil {
		return
	}

	return
}

// Посылай POST запрос к SBSS серверу
// Подразумевается, что ApiKey объект определен
func (this *SbssClient) post(data *bytes.Buffer) (res *http.Response, err error) {
	var (
		req *http.Request
	)

	if req, err = this.NewRequest("POST", this.server, data); err != nil {
		return
	}

	return this.Do(req)
}

func (this *User) UnmarshalJSON(b []byte) (err error) {
	var t = make(map[string]interface{})

	if err = json.Unmarshal(b, &t); err != nil {
		return
	}

	this.Id, _ = strconv.Atoi(t["id"].(string))
	this.Name, _ = t["name"].(string)
	this.Type, _ = strconv.Atoi(t["type"].(string))
	this.Uid = "uuid-" + strconv.Itoa(this.Id)
	this.Classname, _ = t["classname"].(string)

	if tt, ok := t["updated"].(string); ok && tt != "" {
		this.Updated, _ = time.Parse("2006-01-02 15:04:05", tt)
	}

	if this.Type == 1 {
		this.Organization = this.Name
	} else {
		this.FullName = strings.Split(this.Name, " ")
	}

	if v, ok := t["email"]; ok && v.(string) != "" {
		this.Email = &Email{
			Type:  "internet,pref",
			Value: v.(string),
		}
	}

	return
}
