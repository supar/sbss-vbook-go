package main

import (
	"reflect"
	"testing"
)

type vCard_Contact_Test struct {
	Name string `vcard:"fn"`
}

type vCard_Phone_Test struct {
	Type  string `vcard:",separator(=)"`
	Value string `vcard:",omitname"`
}

func Test_EncodeSimpleSingleStruct(t *testing.T) {
	var (
		vc = struct {
			Name   string             `vcard:"fn"`
			Login  string             `vcard:"nickname"`
			Slice  []string           `vcard:"n,inline"`
			Phones []vCard_Phone_Test `vcard:"tel,iteminline(:),separator(;)"`
		}{
			Name:  "Ivan Ivanovich Ivanov",
			Login: "ivanovlogin",
			Slice: []string{"Ivan", "Ivanov", "Ivanovich"},
			Phones: []vCard_Phone_Test{
				vCard_Phone_Test{
					Type:  "cell",
					Value: "+123 00 0000000",
				},
				vCard_Phone_Test{
					Type:  "work",
					Value: "+123 00 1200000",
				},
				vCard_Phone_Test{
					Type:  "home",
					Value: "+123 00 1230000",
				},
			},
		}

		mock = `FN:Ivan Ivanovich Ivanov
NICKNAME:ivanovlogin
N:Ivan;Ivanov;Ivanovich
TEL;TYPE=cell:+123 00 0000000
TEL;TYPE=work:+123 00 1200000
TEL;TYPE=home:+123 00 1230000`
	)

	if d := Encode(vc); string(d) != mock {
		t.Error("Unexpected result")
		t.Logf("%s", d)
		t.Log(mock)
	}
}

func Test_CardsDataArray_CheckNilOptions(t *testing.T) {
	var (
		vc = []vCard_Contact_Test{
			vCard_Contact_Test{
				Name: "Samuel J",
			},
			vCard_Contact_Test{
				Name: "Noname C",
			},
		}

		mock = `FN:Samuel J
FN:Noname C`
	)

	if d := Encode(vc); string(d) != mock {
		t.Error("Unexpected result")
		t.Logf("%s", d)
		t.Log(mock)
	}
}

func Test_AddressBook(t *testing.T) {
	var (
		vc = struct {
			contact []vCard_Contact_Test `vcard:",wrapvcard,version(3.1)"`
			agent   vCard_Contact_Test   `vcard:"agent,wrapvcard,inline"`
		}{
			contact: []vCard_Contact_Test{
				vCard_Contact_Test{
					Name: "Some User A",
				},
				vCard_Contact_Test{
					Name: "Some User B",
				},
			},
			agent: vCard_Contact_Test{
				Name: "Some coworker C",
			},
		}

		mock = `BEGIN:VCARD
VERSION:3.1
FN:Some User A
END:VCARD

BEGIN:VCARD
VERSION:3.1
FN:Some User B
END:VCARD

AGENT:BEGIN:VCARD\nFN:Some coworker C\nEND:VCARD\n`
	)

	if d := Encode(vc); string(d) != mock {
		t.Error("Unexpected result")
		t.Logf("%s", d)
		t.Log(mock)
	}
}

func Test_WrapDataWithVcardTokens(t *testing.T) {
	var (
		data = []byte("FN:Some Name")

		mock = []string{
			`BEGIN:VCARD
FN:Some Name
END:VCARD
`,
			`BEGIN:VCARD
VERSION:3.0
FN:Some Name
END:VCARD
`,
			`BEGIN:VCARD\nFN:Some Name\nEND:VCARD\n`,
			`BEGIN:VCARD\nVERSION:3.0\nFN:Some Name\nEND:VCARD\n`,
		}
	)

	for i, v := range mock {
		opts := &fieldStruct{wrapvcard: true}

		switch i {
		case 1:
			opts.version = true
			opts.versionnum = "3.0"

		case 2:
			opts.inline = true

		case 3:
			opts.inline = true
			opts.version = true
			opts.versionnum = "3.0"
		}

		r := wrapVcard(data, opts)

		if s := string(r); s != v {
			t.Error("Unexpected result")
			t.Log(s)
			t.Log(v)
		}
	}
}

func Test_FieldOptionsValid(t *testing.T) {
	var (
		m fieldStruct

		f = reflect.TypeOf(struct {
			Name      string `vcard:"FN"`
			Nickname  string `vcard:",inline"`
			Skipfield string `vcard:"-"`
			Omitfield string `vcard:"omit,omitempty"`
			Glue      string `vcard:",inline(\n)"`
			NoName    string `vcard:",omitname"`
		}{
			Name:      "Name",
			Nickname:  "Nickname",
			Skipfield: "Skip",
			NoName:    "Skip field label",
		})
	)

	for i := 0; i < f.NumField(); i++ {
		ft := f.Field(i)
		o := fieldOptions(ft)

		switch i {
		case 0:
			m = fieldStruct{
				name:       "FN",
				glue:       ";",
				itemglue:   ";",
				separator:  ":",
				versionnum: "3.0",
			}

		case 1:
			m = fieldStruct{
				name:       "NICKNAME",
				inline:     true,
				iteminline: true,
				glue:       ";",
				itemglue:   ";",
				separator:  ":",
				versionnum: "3.0",
			}

		case 2:
			m = fieldStruct{
				skip: true,
			}

		case 3:
			m = fieldStruct{
				name:       "OMIT",
				omitempty:  true,
				glue:       ";",
				itemglue:   ";",
				separator:  ":",
				versionnum: "3.0",
			}

		case 4:
			m = fieldStruct{
				name:       "GLUE",
				inline:     true,
				iteminline: true,
				glue:       "\n",
				itemglue:   ";",
				separator:  ":",
				versionnum: "3.0",
			}

		case 5:
			m = fieldStruct{
				name:       "NONAME",
				omitname:   true,
				glue:       ";",
				itemglue:   ";",
				separator:  ":",
				versionnum: "3.0",
			}
		}

		if !reflect.DeepEqual(
			m,
			o,
		) {
			t.Errorf("Invalid field options: %v (%v)", o, m)
		}
	}
}
