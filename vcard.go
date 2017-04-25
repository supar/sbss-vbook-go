package main

import (
	"reflect"
	"strconv"
	"strings"
)

const (
	VCardTagName = "vcard"
)

// Stuct tags
// `vcard:"fieldNamme,inlineitems(;),inline(;),omitname,separator(=),wrapvcard,version(3.0)"`
// Exportaed field name is required, You may live empty to take struct original field name
// Field name wil be upper case always
// inlineitem - for the none primitive fields will  join data in line string
// inline - join all field data inline string
// omitaname - do not append field name and write value only
// separator - change default label separator
//
//     struct{
//         Tel struct {
//             Type string    `vcard:",separator(=)"`
//             Value string   `vcard:",omitname"`
//         }                  `vcard:",iteminline(:),separator(;)"`
//     }
type fieldStruct struct {
	// Field name
	name string
	// Join all struct data in line string
	inline bool
	glue   string
	// If struct field is slice, then
	// slice items can be joined in line string
	iteminline bool
	itemglue   string
	// Label separator
	separator string
	// Skip fields with empty data
	omitempty bool
	// Skip field label, just write value
	omitname bool
	// Skip field
	skip bool
	// Write vcard virsion if wrap option is on
	version    bool
	versionnum string
	// Wrap data with VCARD tokens
	wrapvcard bool
}

type vcardWrap struct {
	Item interface{} `vcard:",omitname,wrapvcard,version(3.0)"`
}

func Encode(v interface{}) []byte {
	return element(reflect.ValueOf(v), nil)
}

func EncodeWrap(v interface{}) []byte {
	w := vcardWrap{v}
	return Encode(w)
}

func element(v reflect.Value, opts *fieldStruct) (data []byte) {
	if opts != nil {
		if opts.skip {
			return
		}
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			data = element(v.Elem(), opts)
		}

	case reflect.Slice, reflect.Array:
		data = walkSlice(v, opts)

	case reflect.Struct:
		data = walkStruct(v, opts)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:

		data = primitive(v, opts)
	}

	return
}

func walkSlice(v reflect.Value, opts *fieldStruct) (data []byte) {
	var (
		length = v.Len()
	)

	for i := 0; i < length; i++ {
		item := element(v.Index(i), opts)

		if len(item) > 0 {
			if opts != nil {
				if !opts.inline && opts.iteminline && !opts.omitname {
					item = append([]byte(opts.name+opts.separator), item...)
				}
			}

			data = append(data, item...)

			if i < (length - 1) {
				if opts != nil && opts.inline {
					data = append(data, []byte(opts.glue)...)
				} else {
					data = append(data, []byte("\n")...)
				}
			}
		}
	}

	return
}

func walkStruct(v reflect.Value, opts *fieldStruct) (data []byte) {
	var (
		field     reflect.Value
		fieldOpts fieldStruct

		itemType = v.Type()
		length   = itemType.NumField()
	)

	for i := 0; i < length; i++ {
		field = v.Field(i)

		fieldOpts = fieldOptions(itemType.Field(i))

		item := element(field, &fieldOpts)

		if !fieldOpts.omitname {
			switch field.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64,
				reflect.String:

				item = append([]byte(fieldOpts.name+fieldOpts.separator), item...)

			default:
				if fieldOpts.inline {
					item = append([]byte(fieldOpts.name+fieldOpts.separator), item...)
				}
			}
		}

		if len(item) > 0 {
			data = append(data, item...)

			if i < (length - 1) {
				if opts != nil && opts.iteminline {
					data = append(data, []byte(opts.itemglue)...)
				} else {
					data = append(data, []byte("\n")...)
				}
			}
		}
	}

	return wrapVcard(data, opts)
}

func wrapVcard(data []byte, opts *fieldStruct) (vr []byte) {
	if opts == nil || !opts.wrapvcard || len(data) == 0 {
		return data
	}

	vc := [][]byte{
		[]byte("BEGIN:VCARD"),
		[]byte("VERSION:" + opts.versionnum),
		data,
		[]byte("END:VCARD"),
	}

	for idx, val := range vc {
		if idx == 1 && !opts.version {
			continue
		}

		vr = append(vr, val...)

		if opts.inline {
			vr = append(vr, []byte("\\n")...)
		} else {
			vr = append(vr, []byte("\n")...)
		}
	}

	return

}

func primitive(v reflect.Value, opts *fieldStruct) (data []byte) {
	if opts == nil {
		panic("field options required")
	}

	if opts.skip {
		return
	}

	switch v.Kind() {
	case reflect.String:
		if v.String() == "" && opts.omitempty {
			return
		}

		return []byte(v.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(v.Uint(), 10))

	case reflect.Float32:
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 32))

	case reflect.Float64:
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 64))
	}

	return
}

// Get field properties
func fieldOptions(f reflect.StructField) (fs fieldStruct) {
	var (
		parts []string

		tag = f.Tag.Get(VCardTagName)
	)

	if tag == "-" {
		fs.skip = true
		return
	}

	parts = strings.Split(tag, ",")
	fs.name = parts[0]

	if fs.name == "" {
		fs.name = f.Name
	}

	fs.name = strings.ToUpper(fs.name)
	fs.glue = ";"
	fs.itemglue = ";"
	fs.separator = ":"
	fs.versionnum = "3.0"

	for _, s := range parts[1:] {
		if s == "omitempty" {
			fs.omitempty = true

			continue
		}

		if s == "wrapvcard" {
			fs.wrapvcard = true

			continue
		}

		if s == "omitname" {
			fs.omitname = true

			continue
		}

		if strings.HasPrefix(s, "inline") {
			s = strings.Trim(
				strings.TrimPrefix(s, "inline"),
				" ()",
			)

			fs.inline = true

			if s != "" {
				fs.glue = s
			}

			continue
		}

		if strings.HasPrefix(s, "iteminline") {
			s = strings.Trim(
				strings.TrimPrefix(s, "iteminline"),
				" ()",
			)

			fs.iteminline = true

			if s != "" {
				fs.itemglue = s
			}

			continue
		}

		if strings.HasPrefix(s, "separator") {
			s = strings.Trim(
				strings.TrimPrefix(s, "separator"),
				" ()",
			)

			if s != "" {
				fs.separator = s
			}

			continue
		}

		if strings.HasPrefix(s, "version") {
			s = strings.Trim(
				strings.TrimPrefix(s, "version"),
				" ()",
			)

			fs.version = true

			if s != "" {
				fs.versionnum = s
			}

			continue
		}
	}

	if fs.inline && !fs.iteminline {
		fs.iteminline = true
	}

	return
}
