package main

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func HandlePropfind(w http.ResponseWriter, r *http.Request, ctx *ContextAdapter) {
	var (
		body      []byte
		err       error
		sentBytes int

		req   = XML_Request{}
		multi = NewMultiStatus()
	)

	defer r.Body.Close()

	if ctx.Href == "" {
		ctx.Href = r.URL.RequestURI()

		ctx.Debug("Empty context Href property overwrited to: %s", ctx.Href)
	}

	if body, err = ioutil.ReadAll(r.Body); err != nil {
		ctx.Error(err)
		return
	}

	if len(body) == 0 {
		ctx.Warn("Empty request")

		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err = xml.Unmarshal(body, &req); err != nil {
		ctx.Error(err)

		return
	}

	if v := req.Get("getetag"); v != nil {
		ctx.Href = "/carddav/contacts"
	}

	req.Properties.Each(&multi, ctx)

	// Re-use body
	body = make([]byte, 0)

	if body, err = xml.Marshal(multi); err != nil {
		ctx.Error(err)
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)

	sentBytes, _ = w.Write(body)
	ctx.Notice("Sent %d bytes", sentBytes)
}

func HandleReport(w http.ResponseWriter, r *http.Request, ctx *ContextAdapter) {
	var (
		body      []byte
		err       error
		sentBytes int

		req = XML_Request{}

		multi = Multistatus_XML_Response{
			XmlnsDAV:  "DAV:",
			XmlnsCard: "urn:ietf:params:xml:ns:carddav",
			XmlnsCS:   "http://calendarserver.org/ns/",
		}
	)

	if body, err = ioutil.ReadAll(r.Body); err != nil {
		ctx.Error(err)
		return
	}

	if len(body) == 0 {
		ctx.Warn("Empty request")

		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err = xml.Unmarshal(body, &req); err != nil {
		ctx.Warn("Can't identify request: %s", err.Error())

		return
	}

	switch req.XMLName.Local {
	case "sync-collection":
		SyncCollection(&req, &multi, ctx)

	case "addressbook-multiget":
		MultiGet(&req, &multi, ctx)
	}

	// Re-use body
	body = make([]byte, 0)

	if body, err = xml.Marshal(multi); err != nil {
		ctx.Error(err)
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)

	sentBytes, _ = w.Write(body)
	ctx.Notice("Sent %d bytes", sentBytes)
}

func SyncCollection(req *XML_Request, multi *Multistatus_XML_Response, ctx *ContextAdapter) {
	var (
		contenttype bool
		clients     *ClientsList
		elem        *XML_Elements
		err         error
	)

	if v := req.Get("getcontenttype"); v != nil {
		contenttype = true
	}

	if clients, err = ctx.GetClients(ctx.User, ctx.Password, nil); err != nil {
		ctx.Error(err)

		return
	}

	for _, item := range clients.Users {
		elem = &XML_Elements{}

		elem.Add(NewXmlElement(
			"d:getetag",
			[]byte(`"`+strconv.FormatInt(item.Updated.Unix(), 10)+`"`),
		))

		if contenttype {
			elem.Add(NewXmlElement(
				"d:getcontenttype",
				[]byte("text/vcard"),
			))
		}

		resp := multi.AddResponse(200, elem)
		resp.Href = "/carddav/" + item.Uid + ".vcf"
	}

	multi.Token = RandStringId(16)
}

func MultiGet(req *XML_Request, multi *Multistatus_XML_Response, ctx *ContextAdapter) {
	var (
		clients *ClientsList
		buff    *bytes.Buffer
		elem    *XML_Elements
		err     error
	)

	if clients, err = ctx.GetClients(ctx.User, ctx.Password, nil); err != nil {
		ctx.Error(err)

		return
	}

	for _, item := range clients.Users {
		elem = &XML_Elements{}
		buff = bytes.NewBuffer(nil)

		elem.Add(NewXmlElement(
			"d:getetag",
			[]byte(`"`+strconv.FormatInt(item.Updated.Unix(), 10)+`"`),
		))

		if err = xml.EscapeText(buff, EncodeWrap(item)); err != nil {
			ctx.Error("Can't encode item: %+v", item)
			continue
		}

		elem.Add(NewXmlElement(
			"card:address-data",
			buff.Bytes(),
		))

		resp := multi.AddResponse(200, elem)
		resp.Href = "/carddav/contacts/" + item.Uid + ".vcf"
	}
}

func HandleGetContact(w http.ResponseWriter, r *http.Request, ctx *ContextAdapter) {
	var (
		uid       int
		filter    *ClientsRequest
		clients   *ClientsList
		sentBytes int
		err       error
	)

	w.Header().Set("Contact-type", "text/x-vcard; charset=utf-8")

	uid, _ = strconv.Atoi(
		strings.TrimSuffix(
			strings.TrimPrefix(ctx.Params.ByName("contact"), "uuid-"),
			".vcf",
		),
	)

	if uid == 0 {
		ctx.Error("Can't get user id from url %s", r.URL.RequestURI())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	filter = &ClientsRequest{
		Contacts: 1,
		Uid:      uid,
	}

	if clients, err = ctx.GetClients(ctx.User, ctx.Password, filter); err != nil {
		ctx.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !clients.Success && clients.Error != "" {
		ctx.Error(clients.Error)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if len(clients.Users) == 0 {
		ctx.Error("Can't get user by id %d", uid)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	sentBytes, _ = w.Write(EncodeWrap(clients.Users[0]))
	ctx.Notice("Sent %d bytes", sentBytes)
}
