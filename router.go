package main

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

// Контекс функции обрабатыващей запрос
type ContextAdapter struct {
	// UID запроса
	Id string
	// Интерфейс для ведения логов
	LogIface
	// Интерфейс для обращения к SBSS серверу
	SbssIface
	// Параметры маршрута
	Params   httprouter.Params
	Href     string
	User     string
	Password string
}

// Расширение функционала маршрутизатора запросов
type Router struct {
	// Интерфейс для ведения логов
	LogIface
	// Наследуемый маршрутизатор
	*httprouter.Router
	// Адрес SBSS
	Server string
	// Cache
	Cache []*Session
}

type Session struct {
	sync.Mutex
	id      string
	created time.Time
	client  SbssIface
}

// Шаблон функции для обработки запросов
// указанного маршрута
type ContextHandlerFunc func(http.ResponseWriter, *http.Request, *ContextAdapter)

// Создай объект маршрутизатора
func NewRouter(url string, log LogIface) *Router {
	return &Router{
		LogIface: log,
		Router:   httprouter.New(),
		Server:   url,
	}
}

// Переопредели стандартный метод Handle в маршрутизаторе,
// чтобы расширить фукнционал и реализовать контекст для
// входящих запросов
func (this *Router) Handle(method, path string, handle ContextHandlerFunc) {
	this.Router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		var (
			cookie *http.Cookie
			err    error
			id     string
			ses    *Session
		)

		id = RandStringId(12)

		this.Notice("%s: %s, %s: %s %s", id, r.RemoteAddr, r.UserAgent(), method, r.URL.RequestURI())

		if cookie, err = r.Cookie(NAME + "-sid"); err != nil || cookie == nil {
			this.Debug("%s: Unknown session cookie, will be create", id)

			cookie = &http.Cookie{
				Name:  NAME + "-sid",
				Value: RandStringId(32),
			}
		}

		if ses = this.getSession(cookie.Value); ses == nil {
			ses = this.setSession(cookie.Value)
		}

		ses.Lock()
		defer ses.Unlock()

		this.Debug("%s: Session cookie: %s", id, cookie.Value)
		http.SetCookie(w, cookie)

		if data, err := httputil.DumpRequest(r, true); err != nil {
			this.Error("%s: %s", id, err.Error())
		} else {
			this.Debug("%s: %s", id, data)
		}

		handle(w, r, &ContextAdapter{
			Id:        id,
			LogIface:  this.LogIface,
			SbssIface: ses.client,
			Params:    p,
		})
	})
}

func (this *Router) getSession(sid string) *Session {
	for _, item := range this.Cache {
		if item.id == sid {
			return item
		}
	}
	return nil
}

func (this *Router) setSession(sid string) *Session {
	var s = &Session{
		id:      sid,
		created: time.Now(),
		client:  NewSbssClient(this.Server),
	}

	this.Cache = append(this.Cache, s)

	return s
}

// Remove from the session slice item by index
func (this *Router) deleteCacheItem(i int) {
	var l = len(this.Cache)

	if i > -1 && i < l {
		copy(this.Cache[i:], this.Cache[i+1:])
		this.Cache[len(this.Cache)-1] = nil
		this.Cache = this.Cache[:l-1]
	}
}

func (this *Router) garbage() {
	var now = time.Now().Unix()

	this.Debug("gc: Records in the cache %d", len(this.Cache))

	for idx, item := range this.Cache {
		if (now - item.created.Unix()) > 86400 {
			this.Debug("gc: Session %s is expired", item.id)

			this.deleteCacheItem(idx)
		}
	}
}

func (this *Router) watchGarbage() {
	var gc = time.Duration(1) * time.Hour

	this.Notice("Start garbage collector")

	this.garbage()
	time.AfterFunc(gc, func() { this.watchGarbage() })
}

// Переопредели функцию отладки и добавь UID запроса
func (this *ContextAdapter) Debug(v ...interface{}) {
	this.addId(&v)
	this.LogIface.Debug(v...)
}

// Переопредели функцию ошибки и добавь UID запроса
func (this *ContextAdapter) Error(v ...interface{}) {
	this.addId(&v)
	this.LogIface.Error(v...)
}

// Переопредели функцию информирования и добавь UID запроса
func (this *ContextAdapter) Notice(v ...interface{}) {
	this.addId(&v)
	this.LogIface.Notice(v...)
}

// Переопредели функцию предупреждения и добавь UID запроса
func (this *ContextAdapter) Warn(v ...interface{}) {
	this.addId(&v)
	this.LogIface.Warn(v...)
}

// Добавь к сообщению вначале уникальный ключ
// обрабатываемого запроса
func (this *ContextAdapter) addId(v *[]interface{}) {
	var (
		ln int
	)

	ln = len(*v)

	if ln == 0 {
		return
	}

	switch (*v)[0].(type) {
	case string:
		(*v)[0] = this.Id + ": " + (*v)[0].(string)

	case error:
		(*v)[0] = errors.New(this.Id + ": " + (*v)[0].(error).Error())
	}

}

func HandleAuthorize(fn ContextHandlerFunc) ContextHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *ContextAdapter) {
		var (
			hasAuth bool
		)

		if ctx.User, ctx.Password, hasAuth = r.BasicAuth(); !hasAuth {
			ctx.Warn("Unauthorized request")

			w.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

			return
		}

		ctx.Notice("Authenticate user: %s", ctx.User)

		fn(w, r, ctx)
	}
}
