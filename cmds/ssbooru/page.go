package main

type BasePage struct {
	Title  string
	Styles []string
}

func NewBasePage() BasePage {
	return BasePage{
		Title:  *btitle,
		Styles: []string{"style.css"},
	}
}
