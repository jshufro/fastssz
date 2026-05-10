package generator

func (e *env) decode(name string, v *Value) string {
	tmpl := `// DecodeSSZ unmarshals the {{.name}} from an io.Reader
	func(:: *{{.name}}) Decode(src io.Reader, limit int) (int, error) {
		{{.decode}}
    }`

	str := execTmpl(tmpl, map[string]interface{}{
		"name":   name,
		"decode": v.decodeContainer(true, "src", "limit"),
	})

	return appendObjSignature(str, v)
}

func (v *Value) decodeContainer(start bool, src string, limit string) (str string) {
	tmpl := `fixedSize := ::.fixedSize()
	if {{.limit}} < fixedSize {
		return 0, ssz.ErrSize
	}
	buf, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}
	_, err = ::.UnmarshalSSZTail(buf)
	if err != nil {
		return 0, err
	}
	return len(buf), nil
	`

	str += execTmpl(tmpl, map[string]interface{}{
		"name":  v.name,
		"obj":   v,
		"src":   src,
		"limit": limit,
	})

	return
}
