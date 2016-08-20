package goinside

import "errors"

var errReportResultFalseWithEmptyCause = errors.New("Report Result False With Empty Cause")

func (a *Article) thumbsUp(s session) error {
	return a.action(s, recommendUpAPI)
}

func (a *Article) thumbsDown(s session) error {
	return a.action(s, recommendDownAPI)
}

func (a *Article) action(s session, api dcinsideAPI) error {
	form, contentType := s.actionForm(a)
	resp, err := api.post(s, form, contentType)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

func (a *Article) report(s session, memo string) error {
	form, contentType := s.reportForm(a.URL, memo)
	resp, err := reportAPI.post(s, form, contentType)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}
