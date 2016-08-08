package gallog

import (
	"io"
	"net/http"
)

// web urls
const (
	desktopLoginURL     = "https://dcid.dcinside.com/join/member_check.php"
	desktopLogoutURL    = "https://dcid.dcinside.com/join/logout.php"
	deleteArticleLogURL = "http://gallog.dcinside.com/inc/_deleteArticle.php"
	deleteCommentLogURL = "http://gallog.dcinside.com/inc/_deleteRepOk.php"
)

// apis
const (
	articleDeleteAPI = "http://m.dcinside.com/api/gall_del.php"
	commentDeleteAPI = "http://m.dcinside.com/api/comment_del.php"
)

// content types
const (
	nonCharsetContentType = "application/x-www-form-urlencoded"
)

var (
	apiRequestHeader = map[string]string{
		"User-Agent": "dcinside.app",
		"Referer":    "http://m.dcinside.com",
		"Host":       "m.dcinside.com",
	}
	gallogRequestHeader = map[string]string{
		"User-Agent": "Mozilla/5.0",
		"Referer":    "http://gallog.dcinside.com",
		"Host":       "gallog.dcinside.com",
	}
	desktopRequestHeader = map[string]string{
		"User-Agent": "Mozilla/5.0",
		"Referer":    "http://www.dcinside.com",
		"Host":       "dcid.dcinside.com",
	}
)

func api(URL string, form io.Reader) (*http.Response, error) {
	return do("POST", URL, nil, form, apiRequestHeader)
}

func do(method, URL string, cookies []*http.Cookie, form io.Reader, requestHeader map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, URL, form)
	if err != nil {
		return nil, err
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	req.Header.Set("Content-Type", nonCharsetContentType)
	for k, v := range requestHeader {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	return client.Do(req)
}