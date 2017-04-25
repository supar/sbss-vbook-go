package main

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type TestingWrap struct {
	*testing.T
}

type SbssClientTestWrap SbssClient

func (this *TestingWrap) Critical(v ...interface{}) {
	this.compat(true, v...)
}

func (this *TestingWrap) Error(v ...interface{}) {
	this.compat(true, v...)
}

func (this *TestingWrap) Debug(v ...interface{}) {
	this.compat(false, v...)
}

func (this *TestingWrap) Info(v ...interface{}) {
	this.compat(false, v...)
}

func (this *TestingWrap) Notice(v ...interface{}) {
	this.compat(false, v...)
}

func (this *TestingWrap) Warn(v ...interface{}) {
	this.compat(false, v...)
}

func (this *TestingWrap) compat(err bool, v ...interface{}) {
	switch v[0].(type) {
	case string:
		if !err {
			this.T.Logf(v[0].(string), v[1:]...)
		} else {
			this.T.Errorf(v[0].(string), v[1:]...)
		}
	default:
		if !err {
			this.T.Log(v...)
		} else {
			this.T.Error(v...)
		}
	}
}

func (this *SbssClientTestWrap) GetClients(user, pass string, filter *ClientsRequest) (c *ClientsList, err error) {
	return &ClientsList{
		Success: true,
		Users: []*User{
			&User{
				Id:           333,
				Name:         "John Vick",
				Type:         1,
				Organization: "Freindly org",
				FullName:     []string{"John", "Vick"},
				Email: &Email{
					Type:  "internet",
					Value: "john@vick.net",
				},
				Uid:     "uuid-333",
				Updated: time.Now(),
			},
			&User{
				Id:           334,
				Name:         "Jahn Vooz",
				Type:         1,
				Organization: "Freindly org",
				FullName:     []string{"Jahn", "Vooz"},
				Email: &Email{
					Type:  "internet",
					Value: "jahn@vooz.net",
				},
				Uid:     "uuid-334",
				Updated: time.Now(),
			},
		},
	}, nil
}

func (this *SbssClientTestWrap) GetClientsETag(user, pass string) (etag *ClientsETag, err error) {
	return
}

func Test_XMLElelemt_MarshalXML_Valid(t *testing.T) {
	var (
		data []byte
		err  error

		in = &XML_Elements{
			Items: []XML_Element{
				NewXmlElement(
					"getetag",
					[]byte("\"000000\""),
				),
				NewXmlElement(
					"getetag",
					[]byte("<d:foo>any data</d:foo>"),
				),
			},
		}
	)

	for idx, el := range in.Items {
		if data, err = xml.Marshal(el); err != nil {
			t.Error(err)

			continue
		}

		want := ""

		switch idx {
		case 0:
			want = `<getetag>"000000"</getetag>`
		case 1:
			want = `<getetag><d:foo>any data</d:foo></getetag>`
		}

		if want != string(data) {
			t.Errorf("Unexpected result, want %s", want)
		}
	}
}

func Test_PropfindXmlRequest_ValidateStruct(t *testing.T) {
	var (
		err error

		wants = struct {
			XMLName   xml.Name `xml:"multistatus"`
			Responses []*struct {
				Status     string         `xml:"propstat>status"`
				Properties []*XML_Element `xml:"propstat>prop"`
			} `xml:"response"`
		}{}

		req = `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:" xmlns:x0="urn:ietf:params:xml:ns:carddav">
	<D:prop>
		<x0:addressbook-home-set/>
		<D:getetag/>
		<D:getcontenttype/>
		<D:resourcetype/>
		<D:displayname/>
		<D:supported-report-set/>
	</D:prop>
</D:propfind>`

		list = []string{
			"addressbook-home-set",
			"getcontenttype",
			"displayname",
			"supported-report-set",
			"getetag",
		}

		d = XML_Request{XMLName: xml.Name{Local: "propfind"}}
		m = NewMultiStatus()
	)

	if err = xml.Unmarshal([]byte(req), &d); err != nil {
		t.Error(err)
	}

	for _, v := range list {
		if item := d.Get(v); item == nil {
			t.Errorf("Item `%s` expected, but not found", v)
		}
	}

	tt := &TestingWrap{t}
	d.Properties.Each(&m, &ContextAdapter{LogIface: tt})

	if l := len(m.Responses); l != 2 {
		t.Errorf("Expected 2 response blocks, but found %d", l)
	}

	xmldata := []byte(nil)
	if xmldata, err = xml.Marshal(&m); err != nil {
		t.Error(err)
	}
	t.Logf("%s", xmldata)

	if err = xml.Unmarshal(xmldata, &wants); err != nil {
		t.Error(err)
	}

	if l := len(wants.Responses); l != 2 {
		t.Errorf("Expected 2 response blocks after xml decode, but found %d", l)
	}

	for _, item := range wants.Responses {
		if l := len(item.Properties); strings.HasPrefix(item.Status, "HTTP/1.1 200") && l == 0 {
			t.Errorf("Expected properties at %+v", item)
		}
	}
}

func Test_HandlerPropfind_Request_Valid(t *testing.T) {
	var mock = struct {
		XMLName  xml.Name `xml:"multistatus"`
		Response []struct {
			Properties []XML_Element `xml:"propstat>prop"`
			Status     string        `xml:"propstat>status"`
		} `xml:"response"`
	}{}

	// Wrap testing object to keep context logger
	tt := &TestingWrap{t}
	router := NewRouter("", tt)

	body := strings.NewReader(`<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:">
	<D:prop xmlns:x0="urn:ietf:params:xml:ns:carddav">
		<D:current-user-principal/>
		<D:resourcetype/>
		<x0:addressbook-home-set/>
		<D:displayname/>
	</D:prop>
</D:propfind>`)

	req, err := http.NewRequest("PROPFIND", "/", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Basic dXNlcmZvbzpwYXNzd29yZGJhcg==")

	rr := httptest.NewRecorder()
	router.Handle("PROPFIND", "/", HandleAuthorize(HandlePropfind))
	router.ServeHTTP(rr, req)

	w := rr.Result()
	data, err := ioutil.ReadAll(w.Body)

	if err != nil {
		t.Error(err)
	}

	if len(data) == 0 {
		t.Error("Response expected")
	}

	if err = xml.Unmarshal(data, &mock); err != nil {
		t.Error(err)
	}

	statusOk := 0
	statusNone := 0

	for _, item := range mock.Response {
		if strings.HasPrefix(item.Status, "HTTP/1.1 404") {
			statusNone++
		}

		if strings.HasPrefix(item.Status, "HTTP/1.1 200") {
			statusOk++
		}
	}

	if statusOk == 0 {
		t.Error("Unexpected valid properties count")
	}

	if statusNone == 0 {
		t.Error("Unexpected invalid properties count")
	}
}

// Response Example
//<D:multistatus xmlns:D="DAV:"
//               xmlns:C="urn:ietf:params:xml:ns:carddav">
//  <D:response>
//  <D:href>/addressbooks/` + ctx.User + `/contacts/v102.vcf</D:href>
//    <D:propstat>
//      <D:prop>
//        <D:getetag>"23ba4d-ff11fb"</D:getetag>
//		<D:getcontenttype>text/vcard</D:getcontenttype>
//      </D:prop>
//      <D:status>HTTP/1.1 200 OK</D:status>
//    </D:propstat>
//  </D:response>
//  <D:response>
//    <D:href>/addressbooks/` + ctx.User + `/contacts/v104.vcf</D:href>
//    <D:propstat>
//      <D:prop>
//        <D:getetag>"23ba4d-ff11fb"</D:getetag>
//		<D:getcontenttype>text/vcard</D:getcontenttype>
//      </D:prop>
//      <D:status>HTTP/1.1 200 OK</D:status>
//    </D:propstat>
//  </D:response>
//  <D:sync-token>` + RandStringId(16) + `</D:sync-token>
//</D:multistatus>
func Test_RepoertSyncCollection_Request_Valid(t *testing.T) {

	// Wrap testing object to keep context logger
	tt := &TestingWrap{t}
	router := NewRouter("", tt)

	body := strings.NewReader(`<?xml version="1.0"?>
<sync-collection xmlns="DAV:">
	<sync-token>jGSqiSFZZJPTgQyr</sync-token>
	<prop>
	<getetag/>
	<getcontenttype/>
	</prop>
</sync-collection>`)

	req, err := http.NewRequest("REPORT", "/", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Basic dXNlcmZvbzpwYXNzd29yZGJhcg==")

	rr := httptest.NewRecorder()
	router.Handle("REPORT", "/", HandleAuthorize(func(w http.ResponseWriter, r *http.Request, ctx *ContextAdapter) {
		ctx.SbssIface = &SbssClientTestWrap{}
		HandleReport(w, r, ctx)
	}))
	router.ServeHTTP(rr, req)

	w := rr.Result()
	data, err := ioutil.ReadAll(w.Body)

	if err != nil {
		t.Error(err)
	}

	t.Logf("%s", data)
}

func Test_ReportMultiget_Request_Valid(t *testing.T) {
	// Wrap testing object to keep context logger
	tt := &TestingWrap{t}
	router := NewRouter("", tt)

	body := strings.NewReader(`<?xml version="1.0" encoding="utf-8"?>
<C:addressbook-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
	<D:prop>
		<D:getetag/>
		<C:address-data content-type='text/vcard'/>
	</D:prop>
	<D:href>/</D:href>
</C:addressbook-multiget>`)

	req, err := http.NewRequest("REPORT", "/", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Basic dXNlcmZvbzpwYXNzd29yZGJhcg==")

	rr := httptest.NewRecorder()
	router.Handle("REPORT", "/", HandleAuthorize(func(w http.ResponseWriter, r *http.Request, ctx *ContextAdapter) {
		ctx.SbssIface = &SbssClientTestWrap{}
		HandleReport(w, r, ctx)
	}))
	router.ServeHTTP(rr, req)

	w := rr.Result()
	data, err := ioutil.ReadAll(w.Body)

	if err != nil {
		t.Error(err)
	}

	t.Logf("%s", data)
}
