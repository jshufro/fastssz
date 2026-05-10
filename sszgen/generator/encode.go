package generator

func (e *env) encode(name string, v *Value) string {
	tmpl := `// EncodeSSZ encodes the {{.name}} object
	func (:: *{{.name}}) Encode(dst io.Writer, limit int) (int, error) {
		{{.encode}}
    }`

	str := execTmpl(tmpl, map[string]interface{}{
		"name":   name,
		"encode": v.encodeContainer(true),
	})

	return appendObjSignature(str, v)
}

func (v *Value) encodeContainer(start bool) (str string) {
	tmpl := `buf, err := ssz.MarshalSSZ(::)
	if err != nil {
		return 0, err
	}
	return dst.Write(buf)
	`

	str += execTmpl(tmpl, map[string]interface{}{
		"name": v.name,
		"obj":  v,
	})

	return
}
