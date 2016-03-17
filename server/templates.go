package server

import (
	"bytes"
	"html/template"
	"net/http"
	"time"
)

var htmlTemplates = template.Must(template.New("").Parse(rawHTML))

type Page struct { }

func RenderHTML(w http.ResponseWriter, r *http.Request, name string, page *Page, cacheCtrl string) {
	var buf bytes.Buffer
	Must(htmlTemplates.ExecuteTemplate(&buf, name, page))

	w.Header().Set(ContentType, MediaTypeHTML)
	w.Header().Set(CacheControl, cacheCtrl)
	w.Header().Set(ETag, ETagFor(buf.Bytes()))
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(buf.Bytes()))
}

const rawHTML = `
{{define "home"}}
<!DOCTYPE html>
<html lang="en-US">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Cloud9</title>
	<link rel="stylesheet" type="text/css" href="/css/style.css">
	<link rel="icon" type="image/x-icon" href="/favicon.ico">
	<script src="/js/main.js"></script>
</head>
<body>
	<h1>Cloud9</h1>
	<p>Hello, anonymous user!</p>
</body>
</html>
{{end}}
`
