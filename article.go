package goinside

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
)

var (
	flDataRe  = regexp.MustCompile(`\('FL_DATA'\).value ?= ?'(.*)'`)
	oflDataRe = regexp.MustCompile(`\('OFL_DATA'\).value ?= ?'(.*)'`)
	urlRe     = regexp.MustCompile(`url="?(.*?)"?>`)
	idRe      = regexp.MustCompile(`id=([^&]*)`)
	numberRe  = regexp.MustCompile(`no=(\d+)`)
)

// Article 구조체는 작성된 글에 대한 정보를 표현합니다. 댓글을 달거나 추천, 비추천 할 때 사용합니다.
type Article struct {
	URL    string
	GallID string
	Number string
}

// ArticleWriter 구조체는 글 작성에 필요한 정보를 전달하기 위한 구조체입니다.
type ArticleWriter struct {
	*Session
	GallID  string
	Subject string
	Content string
	Images  []string
}

// NewArticle 함수는 새로운 NewArticleWriter 객체를 반환합니다.
func (s *Session) NewArticle(gallID, subject, content string, images ...string) *ArticleWriter {
	return &ArticleWriter{
		Session: s,
		GallID:  gallID,
		Subject: subject,
		Content: content,
		Images:  images,
	}
}

// Write 함수는 ArticleWriter의 정보를 가지고 글을 작성합니다.
func (a *ArticleWriter) Write() (*Article, error) {
	// get cookies and block key
	cookies, authKey, err := a.getCookiesAndAuthKey(map[string]string{
		"id":        "programming",
		"w_subject": a.Subject,
		"w_memo":    a.Content,
		"w_filter":  "1",
		"mode":      "write_verify",
	}, optionWrite)
	if err != nil {
		return nil, err
	}

	// upload images and get FL_DATA, OFL_DATA string
	var flData, oflData string
	if len(a.Images) > 0 {
		flData, oflData, err = a.UploadImages(a.GallID, a.Images)
		if err != nil {
			return nil, err
		}
	}

	// wrtie article
	ret := &Article{}
	form, contentType := multipartForm(nil, map[string]string{
		"name":       a.id,
		"password":   a.pw,
		"subject":    a.Subject,
		"memo":       a.Content,
		"mode":       "write",
		"id":         a.GallID,
		"mobile_key": "mobile_nomember",
		"FL_DATA":    flData,
		"OFL_DATA":   oflData,
		"Block_key":  authKey,
		"filter":     "1",
	})
	resp, err := a.post(gWrite, cookies, form, contentType)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	URL := urlRe.FindSubmatch(body)
	gallID := idRe.FindSubmatch(body)
	number := numberRe.FindSubmatch(body)
	if len(URL) != 2 || len(gallID) != 2 || len(number) != 2 {
		return nil, errors.New("Write Article Fail")
	}
	ret.URL, ret.GallID, ret.Number = string(URL[1]), string(gallID[1]), string(number[1])
	return ret, nil
}

// DeleteArticle 함수는 인자로 주어진 글을 삭제합니다.
func (s *Session) DeleteArticle(a *Article) error {
	// get cookies and con key
	m := map[string]string{}
	if s.nomember {
		m["token_verify"] = "nonuser_del"
	} else {
		return errors.New("Need to login")
	}
	cookies, authKey, err := s.getCookiesAndAuthKey(m, accessToken)
	if err != nil {
		return err
	}

	// delete article
	form := form(map[string]string{
		"id":       a.GallID,
		"write_pw": s.pw,
		"no":       a.Number,
		"mode":     "board_del2",
		"con_key":  authKey,
	})
	_, err = s.post(optionWrite, cookies, form, defaultContentType)
	return err
}

// DeleteArticles 함수는 인자로 주어진 여러 개의 글을 동시에 삭제합니다.
func (s *Session) DeleteArticles(as []*Article) error {
	done := make(chan error)
	defer close(done)
	for _, a := range as {
		a := a
		go func() {
			done <- s.DeleteArticle(a)
		}()
	}
	for _ = range as {
		if err := <-done; err != nil {
			return err
		}
	}
	return nil
}

// UploadImages 함수는 인자로 전달받은 이미지 파일들을 디시인사이드 서버에 업로드 한 뒤, 이미지에 대한 FL_DATA, OFL_DATA 값을 반환합니다.
func (s *Session) UploadImages(gall string, images []string) (string, string, error) {
	form, contentType := multipartForm(images, map[string]string{
		"imgId":   gall,
		"mode":    "write",
		"img_num": "11", // ?
	})
	resp, err := s.post(uploadImage, nil, form, contentType)
	if err != nil {
		return "", "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	fldata := flDataRe.FindSubmatch(body)
	ofldata := oflDataRe.FindSubmatch(body)
	if len(fldata) != 2 || len(ofldata) != 2 {
		return "", "", errors.New("Image Upload Fail")
	}
	return string(fldata[1]), string(ofldata[1]), nil
}

func (s *Session) getCookiesAndAuthKey(m map[string]string, URL string) ([]*http.Cookie, string, error) {
	var cookies []*http.Cookie
	var authKey string
	form := form(m)
	resp, err := s.post(URL, nil, form, defaultContentType)
	if err != nil {
		return nil, "", err
	}
	cookies = resp.Cookies()
	authKey, err = parseAuthkKey(resp)
	if err != nil {
		return nil, "", err
	}
	return cookies, authKey, nil
}

func parseAuthkKey(resp *http.Response) (string, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var tempJSON struct {
		Msg  string
		Data string
	}
	json.Unmarshal(body, &tempJSON)
	if tempJSON.Data == "" {
		return "", errors.New("Block Key Parse Fail")
	}
	return tempJSON.Data, nil
}

func multipartForm(images []string, m map[string]string) (io.Reader, string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if images != nil {
		multipartImages(w, images)
	}
	multipartOthers(w, m)
	return &b, w.FormDataContentType()
}

func multipartImages(w *multipart.Writer, images []string) {
	for i, image := range images {
		f, err := os.Open(image)
		if err != nil {
			return
		}
		defer f.Close()
		fw, err := w.CreateFormFile(fmt.Sprintf("upload[%d]", i), image)
		if err != nil {
			return
		}
		if _, err = io.Copy(fw, f); err != nil {
			return
		}
	}
}

func multipartOthers(w *multipart.Writer, m map[string]string) {
	for k, v := range m {
		if fw, err := w.CreateFormField(k); err != nil {
			continue
		} else if _, err := fw.Write([]byte(v)); err != nil {
			continue
		}
	}
}
