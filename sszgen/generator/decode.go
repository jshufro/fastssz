package generator

import (
	"fmt"
	"strconv"
	"strings"
)

func (e *env) decode(name string, v *Value) string {
	tmpl := `// DecodeSSZ unmarshals the {{.name}} from an io.Reader
	func(:: *{{.name}}) Decode(src io.Reader, limit int) (n int, err error) {
		{{.decode}}
    }`

	str := execTmpl(tmpl, map[string]interface{}{
		"name":   name,
		"decode": v.decodeContainer(true, "limit"),
	})

	return appendObjSignature(str, v)
}

func (v *Value) decode(limit string) string {
	switch obj := v.typ.(type) {
	case *Container, *Reference:
		return v.decodeContainer(false, limit)
	case *Bytes:
		if !obj.IsList && !obj.IsGoDyn {
			return fmt.Sprintf(`read, err = io.ReadFull(src, ::.%s[:])
			n += read
			if err != nil {
			  return
			}`, v.name)
		}

		var tmpl string
		if !v.isFixed() {
			tmpl = `read, ::.{{.name}}, err = ssz.DecodeDynamicBytes(::.{{.name}}, src, {{.limit}}, {{.size}})
			n += read
			if err != nil {
				return
			}`
		} else {
			tmpl = `read, ::.{{.name}}, err = ssz.DecodeBytes(::.{{.name}}, src, {{.size}})
			n += read
			if err != nil {
				return
			}`
		}

		return execTmpl(tmpl, map[string]interface{}{
			"name":  v.name,
			"size":  obj.Size,
			"limit": limit,
		})
	case *BitList:
		tmpl := `read, ::.{{.name}}, err = ssz.DecodeBitList(::.{{.name}}, src, {{.limit}}, {{.max}})
		n += read
		if err != nil {
			return
		}`
		return execTmpl(tmpl, map[string]interface{}{
			"name":  v.name,
			"limit": limit,
			"max":   obj.Size,
		})
	case *Uint:
		intType := uintVToLowerCaseName2(obj)

		var objRef string
		if v.ref != "" {
			objRef = v.objRef()
		} else if v.obj != "" {
			objRef = v.obj
		}

		var tmpl string
		if objRef != "" {
			tmpl = `{
				var val {{.type}}
				read, err = ssz.DecodeValue[{{.type}}](&val, src)
				n += read
				if err != nil {
					return
				}
				::.{{.name}} = {{.objRef}}(val)
			}`
		} else {
			tmpl = `read, err = ssz.DecodeValue[{{.type}}](&::.{{.name}}, src)
			n += read
			if err != nil {
				return
			}`
		}
		return execTmpl(tmpl, map[string]interface{}{
			"name":   v.name,
			"type":   intType,
			"objRef": objRef,
		})
	case *Bool:
		tmpl := `read, err = ssz.DecodeValue[bool](&::.{{.name}}, src)
		n += read
		if err != nil {
			return
		}`
		return execTmpl(tmpl, map[string]interface{}{
			"name": v.name,
		})
	case *Time:
		tmpl := `read, err = ssz.DecodeTime(&::.{{.name}}, src)
		n += read
		if err != nil {
			return
		}`
		return execTmpl(tmpl, map[string]interface{}{
			"name": v.name,
		})
	case *Vector:
		if obj.Elem.isFixed() {
			tmpl := `{{.create}}
			for ii := uint64(0); ii < {{.size}}; ii++ {
				{{.decode}}
			}`
			return execTmpl(tmpl, map[string]interface{}{
				"create": v.createSlice(false),
				"size":   obj.Size,
				"decode": obj.Elem.decode(limit),
			})
		}
		return v.decodeList(limit)
	case *List:
		return v.decodeList(limit)
	default:
		panic(fmt.Errorf("decode not implemented for type %s", v.Type()))
	}
}

func (v *Value) decodeList(limit string) string {
	var size Size
	if obj, ok := v.typ.(*List); ok {
		size = obj.MaxSize
	} else if obj, ok := v.typ.(*Vector); ok {
		size = obj.Size
	} else {
		panic(fmt.Errorf("decodeList not implemented for type %s", v.Type()))
	}

	inner := getElem(v.typ)
	if inner.isFixed() {
		var tmpl string
		var innerSize string

		if inner.isContainer() && !inner.noPtr {
			tmpl = `read, err = ssz.DecodeSliceSSZ(&::.{{.name}}, src, {{.limit}}, {{.max}})
			n += read
			if err != nil {
				return
			}`
		} else {
			switch obj := inner.typ.(type) {
			case *Uint:
				innerSize = fmt.Sprintf("%d", obj.Size)
			case *Bytes:
				innerSize = obj.Size.MarshalTemplate()
			case *Container:
				innerSize = inner.fixedSizeForContainer()
			default:
				panic(fmt.Errorf("decodeList not implemented for type %s", inner.Type()))
			}
			tmpl = `read, err = ssz.DecodeSliceWithIndexCallback(&::.{{.name}}, src, {{.limit}}, {{.size}}, {{.max}}, func(ii uint64, src io.Reader, elementLimit int) (n int, err error) {
			var read int
			{{.decode}}
			return
			})
			n += read
			if err != nil {
				return
			}`
		}
		return execTmpl(tmpl, map[string]interface{}{
			"name":   v.name,
			"size":   innerSize,
			"max":    size,
			"limit":  limit,
			"decode": inner.decode("elementLimit"),
		})
	}

	var tmpl string
	if inner.isContainer() && !inner.noPtr {
		tmpl = `read, err = ssz.DecodeDynamicSliceSSZ(&::.{{.name}}, src, {{.limit}}, {{.max}})
		n += read
		if err != nil {
			return
		}`
	} else {
		tmpl = `read, err = ssz.DecodeDynamicSliceWithCallback(&::.{{.name}}, src, {{.limit}}, {{.max}}, func(indx uint64, src io.Reader, elementLimit int) (n int, err error) {
			{{.decode}}
			return
		})
		n += read
		if err != nil {
			return
		}`
	}

	inner.name = v.name + "[indx]"
	return execTmpl(tmpl, map[string]interface{}{
		"max":    size,
		"name":   v.name,
		"limit":  limit,
		"decode": inner.decode("elementLimit"),
	})
}

func isInDecodeOffset(limit string) bool {
	return !strings.Contains(limit, "SizeSSZ")
}

func (v *Value) decodeContainer(start bool, limit string) (str string) {
	if !start {
		var tmpl string
		if isInDecodeOffset(limit) {
			tmpl = `{{if .ptr}}read, err = ssz.DecodeField(&::.{{.name}}, io.LimitReader(src, int64({{.limit}})), {{.limit}})
			n += read
			if err != nil {
				return
			}{{else}}read, err = ::.{{.name}}.Decode(io.LimitReader(src, int64({{.limit}})), {{.limit}})
			n += read
			if err != nil {
				return
			}{{end}}
			`
		} else {
			tmpl = `{{if .ptr}}read, err = ssz.DecodeField(&::.{{.name}}, src, ::.{{.name}}.SizeSSZ())
			n += read
			if err != nil {
				return
			}{{else}}read, err = ::.{{.name}}.Decode(src, ::.{{.name}}.SizeSSZ())
			n += read
			if err != nil {
				return
			}{{end}}
			`
		}
		return execTmpl(tmpl, map[string]interface{}{
			"name":  v.name,
			"ptr":   !v.noPtr,
			"limit": limit,
		})

	}
	offsets, offsetsMatch := v.getOffsets()

	str += `fixedSize := ::.fixedSize()
	if limit < fixedSize {
		return 0, ssz.ErrSize
	}
	`

	if len(v.getObjs()) > 0 {
		str += "var read int\n"
	}

	tmpl := `{{if .offsets}}var {{.offsets}} uint64
		marker := ssz.NewOffsetMarker(uint64(limit), uint64(fixedSize))
	{{end}}
	`

	str += execTmpl(tmpl, map[string]interface{}{
		"offsets": strings.Join(offsets, ", "),
	})

	outs := []string{}
	for indx, i := range v.getObjs() {
		var res string
		if i.isFixed() {
			res = fmt.Sprintf("// Field (%d) '%s'\n%s\n\n", indx, i.name, i.decode("SizeSSZ"))
		} else {
			// decode the offset
			offset := "o" + strconv.Itoa(indx)
			data := map[string]interface{}{
				"indx":   indx,
				"name":   i.name,
				"offset": offset,
			}
			if prev, ok := offsetsMatch[offset]; ok {
				data["more"] = fmt.Sprintf(" || %s > %s", prev, offset)
			} else {
				data["more"] = ""
			}

			data["isLastOffset"] = indx == len(v.getObjs())-1

			tmpl := `// Offset ({{.indx}}) '{{.name}}'
			{{.offset}}, read, err = marker.DecodeOffset(src)
			n += read
			if err != nil {
				return
			}
			`
			res = execTmpl(tmpl, data)
		}
		outs = append(outs, res)
	}

	// Decode the dynamic parts
	c := 0
	for indx, i := range v.getObjs() {
		if !i.isFixed() {
			from := offsets[c]
			var to string
			if c == len(offsets)-1 {
				to = "uint64(limit)"
			} else {
				to = offsets[c+1]
			}
			fieldLimit := fmt.Sprintf("int(%s - %s)", to, from)
			tmpl := `// Field ({{.indx}}) '{{.name}}'
			{{.decode}}
			`

			res := execTmpl(tmpl, map[string]interface{}{
				"indx":   indx,
				"name":   i.name,
				"from":   from,
				"to":     to,
				"decode": i.decode(fieldLimit),
			})
			outs = append(outs, res)
			c++
		}
	}

	tmpl = `if n != {{.limit}} {
		return n, ssz.ErrSize
	}`
	outs = append(outs, execTmpl(tmpl, map[string]interface{}{
		"limit": limit,
	}))

	str += strings.Join(outs, "\n\n")

	str += "\nreturn"

	return
}
