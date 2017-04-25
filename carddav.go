package main

import (
	"encoding/xml"
	"strconv"
)

type XML_Request struct {
	XMLName    xml.Name
	Properties XML_Properties `xml:"prop"`
}

type XML_Properties []XMLPropertyIface

type XMLPropertyIface interface {
	Name() string
	Prefix() string
	Status() int
	XmlElem(*ContextAdapter) XML_Element
}

type XML_Property struct {
	XmlInner    []byte
	XmlName     string
	XmlNsPrefix string
	XmlStatus   int
}

type Multistatus_XML_Response struct {
	XMLName   xml.Name `xml:"d:multistatus"`
	XmlnsDAV  string   `xml:"xmlns:d,attr"`
	XmlnsCard string   `xml:"xmlns:card,attr"`
	XmlnsCS   string   `xml:"xmlns:cs,attr"`
	Token     string   `xml:"sync-token,omitempty"`
	Responses []*XML_Response
}

type XML_Response struct {
	XMLName    xml.Name            `xml:"d:response"`
	Href       string              `xml:"d:href,omitempty"`
	Properties []*XML_Elements     `xml:"d:propstat>d:prop"`
	Status     Status_XML_Response `xml:"d:propstat>d:status"`
}

type XML_Elements struct {
	Items []XML_Element
}

type Status_XML_Response struct {
	value int
}

type CTag_XML_Property XML_Property

type XML_Element struct {
	XMLName         xml.Name
	Data            []byte `xml:",innerxml"`
	Error           string `xml:"d:error,omitempty"`
	Status          int    `xml:"-"`
	UnmarshaledName string `xml:"-"`
}

type XmlResponseIface interface {
	Do(*ContextAdapter) (int, []byte)
}

var Known_Properties map[string]func(xml.StartElement) XMLPropertyIface

func init() {
	Known_Properties = map[string]func(xml.StartElement) XMLPropertyIface{
		// Support CTag property
		"getctag": func(start xml.StartElement) XMLPropertyIface {
			return &CTag_XML_Property{
				XmlName:     start.Name.Local,
				XmlNsPrefix: "cs",
				XmlStatus:   200,
			}
		},

		"addressbook-home-set": func(start xml.StartElement) XMLPropertyIface {
			return &XML_Property{
				XmlName:     start.Name.Local,
				XmlNsPrefix: "card",
				XmlStatus:   200,
				XmlInner:    []byte("<d:href>/carddav/contacts</d:href>"),
			}
		},

		"supported-report-set": func(start xml.StartElement) XMLPropertyIface {
			return &XML_Property{
				XmlName:     start.Name.Local,
				XmlNsPrefix: "d",
				XmlStatus:   200,
				XmlInner:    []byte("<d:supported-report><d:report><d:sync-collection/></d:report></d:supported-report>"),
			}
		},

		"displayname": func(start xml.StartElement) XMLPropertyIface {
			return &XML_Property{
				XmlName:     start.Name.Local,
				XmlNsPrefix: "d",
				XmlStatus:   200,
				XmlInner:    []byte("SBSS contacts"),
			}
		},

		"resourcetype": func(start xml.StartElement) XMLPropertyIface {
			return &XML_Property{
				XmlName:     start.Name.Local,
				XmlNsPrefix: "d",
				XmlStatus:   200,
				XmlInner:    []byte("<d:collection/><card:addressbook/>"),
			}
		},

		"getcontenttype": func(start xml.StartElement) XMLPropertyIface {
			return &XML_Property{
				XmlName:     start.Name.Local,
				XmlNsPrefix: "d",
				XmlStatus:   200,
				XmlInner:    []byte("text/x-vcard; charset=utf-8"),
			}
		},

		"current-user-principal": func(start xml.StartElement) XMLPropertyIface {
			return &XML_Property{
				XmlName:     start.Name.Local,
				XmlNsPrefix: "d",
				XmlStatus:   200,
				XmlInner:    []byte("<d:href>/carddav</d:href>"),
			}
		},

		"getetag": func(start xml.StartElement) XMLPropertyIface {
			return &XML_Property{
				XmlName:     start.Name.Local,
				XmlNsPrefix: "d",
				XmlStatus:   200,
				XmlInner:    []byte(nil),
			}
		},
	}
}

func NewMultiStatus() Multistatus_XML_Response {
	return Multistatus_XML_Response{
		XmlnsDAV:  "DAV:",
		XmlnsCard: "urn:ietf:params:xml:ns:carddav",
		XmlnsCS:   "http://calendarserver.org/ns/",
	}
}

func NewXmlElement(name string, data []byte) XML_Element {
	return XML_Element{
		XMLName: xml.Name{Local: name},
		Data:    data,
	}
}

func (this *XML_Properties) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var (
		elem XMLPropertyIface
		name string
	)

	for {
		token, _ := d.Token()

		if token == nil {
			break
		}

		switch startElement := token.(type) {
		case xml.StartElement:
			name = startElement.Name.Local

			if item, ok := Known_Properties[name]; ok {
				elem = item(startElement)
			} else {
				elem = XMLPropertyIface(&XML_Property{
					XmlName:   name,
					XmlStatus: 404,
				})
			}

			(*this) = append((*this), elem)
		}
	}

	return nil
}

// Encode Status struct to the XML Element
func (this Status_XML_Response) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	s := "HTTP/1.1 " + strconv.Itoa(this.value) + " "

	switch this.value {
	case 200:
		s += "Ok"

	case 404:
		s += "Not Found"
	}

	e.EncodeElement(s, start)
	return nil
}

func (this *Multistatus_XML_Response) AddResponse(status int, items *XML_Elements) (resp *XML_Response) {
	resp = &XML_Response{
		Properties: []*XML_Elements{items},
		Status:     Status_XML_Response{status},
	}

	this.Responses = append(this.Responses, resp)

	return
}

func (this *XML_Request) Get(name string) XMLPropertyIface {
	for _, item := range this.Properties {
		if item.Name() == name {
			return item
		}
	}
	return nil
}

func (this *XML_Elements) Add(el XML_Element) {
	this.Items = append(this.Items, el)
}

func (this *XML_Properties) Each(multi *Multistatus_XML_Response, ctx *ContextAdapter) {
	var (
		resp = make(map[int]*XML_Elements)
	)

	for _, item := range *this {
		ctx.Debug("Processing property: %s status(%d)", item.Name(), item.Status())

		if _, ok := resp[item.Status()]; !ok {
			resp[item.Status()] = &XML_Elements{}
		}

		resp[item.Status()].Add(item.XmlElem(ctx))
	}

	for s, items := range resp {
		r := multi.AddResponse(s, items)

		if r.Status.value == 200 {
			r.Href = ctx.Href
		}
	}
}

func (this *XML_Property) Name() string {
	return this.XmlName
}

func (this *XML_Property) Prefix() string {
	if this.XmlNsPrefix != "" {
		return this.XmlNsPrefix + ":"
	}

	return this.XmlNsPrefix
}

func (this *XML_Property) Status() int {
	return this.XmlStatus
}

func (this *XML_Property) XmlElem(ctx *ContextAdapter) XML_Element {
	return XML_Element{
		XMLName: xml.Name{
			Local: this.Prefix() + this.Name(),
		},
		Status: 200,
		Data:   this.XmlInner,
	}
}

func (this *CTag_XML_Property) Name() string {
	return this.XmlName
}

func (this *CTag_XML_Property) Prefix() string {
	if this.XmlNsPrefix != "" {
		return this.XmlNsPrefix + ":"
	}

	return this.XmlNsPrefix
}

func (this *CTag_XML_Property) Status() int {
	return this.XmlStatus
}

func (this *CTag_XML_Property) XmlElem(ctx *ContextAdapter) XML_Element {
	var (
		err  error
		etag *ClientsETag

		elem = XML_Element{
			XMLName: xml.Name{
				Local: this.Prefix() + this.Name(),
			},
			Status: 200,
		}
	)

	if etag, err = ctx.GetClientsETag(ctx.User, ctx.Password); err != nil {
		ctx.Error(err)

		elem.Error = "Internal server error"
		elem.Status = 500
		return elem
	}

	elem.Data = []byte(`"` + etag.ETag + `"`)

	return elem
}
