package common

import (
	"bytes"
	"fmt"
	"github.com/beevik/etree"
	"github.com/fatih/structtag"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	OPTION_FILE          = "file"
	OPTION_HIDDEN        = "hidden"
	OPTION_SELECT        = "select"
	OPTION_DATALIST      = "datalist"
	OPTION_PASSWORD      = "password"
	OPTION_AUTOFOCUS     = "autofocus"
	OPTION_REQUIRED      = "required"
	OPTION_READONLY      = "readonly"
	OPTION_MULTILINE     = "multiline"
	OPTION_MEGALINE      = "megaline"
	OPTION_NOLABEL       = "nolabel"
	OPTION_NOPLACEHOLDER = "noplaceholder"

	INPUT_WIDTH_NORMAL = "pure-input-1-4"
	INPUT_WIDTH_WIDE   = "pure-input-2-4"

	CSS_DIALOG_WIDTH = "css-dialog-width"
	CSS_FULL_WIDTH   = "css-full-width"
	CSS_ERROR_BOX    = "css-error-box"
	CSS_WARNING_BOX  = "css-warning-box"
	CSS_SUCCESS_BOX  = "css-success-box"
	CSS_VERTICAL_DIV = "css-vertical-div"
	CSS_MARGIN_DIV   = "css-margin-div"
	CSS_LOGVIEWER    = "css-logviewer"
	CSS_CONTENT      = "css-content"

	FLASH_WARNING = "warning-flash"
	FLASH_ERROR   = "error-flash"
	FLASH_SUCCESS = "success-flash"

	COOKIE_PASSWORD = "password"
)

type Webpage struct {
	doc *etree.Document

	HtmlRoot          *etree.Element
	HtmlHead          *etree.Element
	HtmlTitle         *etree.Element
	HtmlMenu          *etree.Element
	HtmlBody          *etree.Element
	HtmlScrollContent *etree.Element
	HtmlContent       *etree.Element
}

type FuncDatalist func(string) []string

type FuncFieldIterator func(reflect.StructField, reflect.Value) bool

type ActionItem struct {
	Caption  string
	Icon     string
	Action   string
	File     string
	Enabled  bool
	SubItems []ActionItem
}

func indexOf(options []string, option string) int {
	for i, v := range options {
		if v == option {
			return i
		}
	}

	return -1
}

func NewPage(context echo.Context, contentStyle string, title string, scrollable bool) (*Webpage, error) {
	p := Webpage{doc: etree.NewDocument()}

	p.HtmlRoot = p.doc.CreateElement("html")

	lang, err := GetConfiguration().GetFlag("language")
	if WarnError(err) {
		lang = DEFAULT_LANGUAGE
	}
	p.HtmlRoot.CreateAttr("lang", lang)

	p.HtmlHead = p.HtmlRoot.CreateElement("head")
	p.HtmlTitle = p.HtmlHead.CreateElement("title")
	p.HtmlTitle.SetText(title)

	htmlMeta := p.HtmlHead.CreateElement("meta")
	htmlMeta.CreateAttr("name", "viewport")
	htmlMeta.CreateAttr("content", "width = device-width, initial-scale = 1")

	p.HtmlBody = p.HtmlRoot.CreateElement("body")

	p.HtmlMenu = p.HtmlBody.CreateElement("div")

	p.HtmlScrollContent = p.HtmlBody.CreateElement("div")
	if scrollable {
		p.HtmlScrollContent.CreateAttr("class", CSS_CONTENT)
	}

	p.HtmlContent = p.HtmlScrollContent.CreateElement("div")
	p.HtmlContent.CreateAttr("class", contentStyle)

	msg := PullFlash(context, FLASH_WARNING)
	if msg != "" {
		htmlError := p.HtmlContent.CreateElement("div")
		htmlError.CreateAttr("class", CSS_WARNING_BOX)
		htmlError.SetText(msg)
	}

	msg = PullFlash(context, FLASH_SUCCESS)
	if msg != "" {
		htmlError := p.HtmlContent.CreateElement("div")
		htmlError.CreateAttr("class", CSS_SUCCESS_BOX)
		htmlError.SetText(msg)
	}

	msg = PullFlash(context, FLASH_ERROR)
	if msg != "" {
		htmlError := p.HtmlContent.CreateElement("div")
		htmlError.CreateAttr("class", CSS_ERROR_BOX)
		htmlError.SetText(msg)
	}

	return &p, nil
}

func PullFlash(context echo.Context, flashName string) string {
	cookie, _ := session.Get(Title(), context)
	if cookie != nil {
		flashes := cookie.Flashes(flashName)
		if len(flashes) > 0 {
			flash := flashes[0].(string)

			err := cookie.Save(context.Request(), context.Response())
			if Error(err) {
				return flash
			}

			return flash
		}
	}

	return ""
}

func PushFlash(context echo.Context, flashName string, flash string) error {
	cookie, _ := session.Get(Title(), context)
	if cookie != nil {
		cookie.AddFlash(flash, flashName)
	}

	err := cookie.Save(context.Request(), context.Response())
	if Error(err) {
		return err
	}

	return nil
}

func RefreshCookie(context echo.Context, timeout time.Duration) error {
	cookie, _ := session.Get(Title(), context)
	cookie.Options.Path = "/"
	cookie.Options.MaxAge = int(timeout.Seconds())
	cookie.Options.HttpOnly = true

	err := cookie.Save(context.Request(), context.Response())
	if Error(err) {
		return err
	}

	return nil
}

func AuthenticateCookie(context echo.Context, password string, timeout time.Duration) error {
	cookie, _ := session.Get(Title(), context)

	// Ignore the error (maybe the current cookie was encrypted with an outdated httpServer.store key)

	cookie.Options.Path = "/"
	cookie.Options.MaxAge = int(timeout.Seconds())
	cookie.Options.HttpOnly = true

	cookie.Values[COOKIE_PASSWORD] = password

	err := cookie.Save(context.Request(), context.Response())
	if Error(err) {
		return err
	}

	return nil
}

func IsCookieAuthenticated(context echo.Context, passwords []string, hashFunc func(string) string) bool {
	cookie, err := session.Get(Title(), context)
	if err != nil {
		return false
	}

	sessionPassword, ok := cookie.Values[COOKIE_PASSWORD]
	if !ok {
		return false
	}

	if hashFunc != nil {
		sessionPassword = hashFunc(sessionPassword.(string))
	}

	found := false
	for _, password := range passwords {
		found = password != "" && password == sessionPassword

		if found {
			break
		}
	}

	return found
}

func NewMenu(page *Webpage, menuItems []ActionItem, selectedTitle string) {
	page.HtmlMenu.CreateAttr("class", "pure-menu pure-menu-horizontal")

	newMenuitem(page.HtmlMenu, true, menuItems, selectedTitle)
}

func newMenuitem(parent *etree.Element, mainMenu bool, menuItems []ActionItem, selectedTitle string) {
	htmlUl := parent.CreateElement("ul")
	if mainMenu {
		htmlUl.CreateAttr("class", "pure-menu-list")
	} else {
		htmlUl.CreateAttr("class", "pure-menu-children")
	}

	for _, menu := range menuItems {
		htmlMenu := htmlUl.CreateElement("li")

		classes := []string{"pure-menu-item"}

		if menu.Caption == selectedTitle {
			classes = append(classes, "pure-menu-selected")
		}
		if len(menu.SubItems) > 0 {
			classes = append(classes, "pure-menu-has-children")
			classes = append(classes, "pure-menu-allow-hover")
		}

		htmlMenu.CreateAttr("class", strings.Join(classes, " "))

		htmlAhref := htmlMenu.CreateElement("a")
		if len(menu.SubItems) > 0 {
			htmlAhref.CreateAttr("onClick", "return false;")
		} else {
			if strings.Index(menu.Action, ";") != -1 {
				htmlAhref.CreateAttr("onClick", menu.Action)
			} else {
				htmlAhref.CreateAttr("href", menu.Action)
			}
		}

		htmlAhref.CreateAttr("class", "pure-menu-link")

		if menu.Caption != "" {
			caption := menu.Caption
			if menu.Icon != "" {
				caption = "--%" + menu.Icon + "%--" + caption
			}

			htmlAhref.SetText(caption)
		}

		if len(menu.SubItems) > 0 {
			newMenuitem(htmlMenu, false, menu.SubItems, selectedTitle)
		}

		if menu.File != "" {
			htmlAhref.CreateAttr("download", menu.File)
		}
	}
}

func NewRefreshPage(name string, url string) (*Webpage, error) {
	p := Webpage{doc: etree.NewDocument()}

	p.HtmlRoot = p.doc.CreateElement("html")
	p.HtmlHead = p.HtmlRoot.CreateElement("head")
	p.HtmlTitle = p.HtmlHead.CreateElement("title")
	p.HtmlTitle.SetText(name)
	p.HtmlBody = p.HtmlRoot.CreateElement("body")

	htmlMeta := p.HtmlHead.CreateElement("meta")
	htmlMeta.CreateAttr("http-equiv", "refresh")
	htmlMeta.CreateAttr("content", fmt.Sprintf("0; URL=%s", url))

	return &p, nil
}

func NewForm(parent *etree.Element, name string, data interface{}, method string, encType string, formAction string, actions []ActionItem, funcDatalist FuncDatalist, funcFieldIterator FuncFieldIterator) (*etree.Element, error) {
	htmlForm := parent.CreateElement("form")
	htmlForm.CreateAttr("method", method)
	if encType != "" {
		htmlForm.CreateAttr("enctype", encType)
	}
	htmlForm.CreateAttr("class", "pure-form pure-form-aligned")

	htmlForm.CreateAttr("action", formAction)
	htmlForm.CreateAttr("method", method)

	htmlFieldset := htmlForm.CreateElement("fieldset")

	err := newFieldset(0, htmlFieldset, name, data, funcDatalist, funcFieldIterator)
	if Error(err) {
		return nil, err
	}

	htmlGroup := htmlForm.CreateElement("div")
	htmlGroup.CreateAttr("class", "pure-controls")

	for i, action := range actions {
		NewButton(htmlGroup, i == 0, action)
	}

	htmlFooter := parent.CreateElement("div")
	htmlFooter.CreateAttr("class", "css-margin-div")

	return htmlForm, nil
}

func BindForm(context echo.Context, data interface{}) error {
	err := IterateStruct(data, func(typ reflect.StructField, val reflect.Value) error {
		if val.Kind() == reflect.Struct {
			err := context.Bind(val.Addr().Interface())
			if Error(err) {
				return err
			}
		}

		fieldTags, err := structtag.Parse(string(typ.Tag))
		if err != nil {
			return err
		}

		if val.Kind() == reflect.Bool {
			tag, err := fieldTags.Get("form")
			if err != nil {
				return nil
			}

			val.SetBool(context.FormValue(tag.Name) != "")
		}

		if val.Kind() == reflect.Slice {
			tagForm, err := fieldTags.Get("form")
			if err != nil {
				return err
			}

			if tagForm.Name == "file" {
				file, err := context.FormFile(tagForm.Name)
				if err != nil {
					return err
				}

				src, err := file.Open()
				if err != nil {
					return err
				}
				defer func() {
					Error(src.Close())
				}()

				var buf bytes.Buffer

				_, err = io.Copy(&buf, src)
				if err != nil {
					return err
				}

				val.SetBytes(buf.Bytes())
			}
		}

		return nil
	})

	return err
}

func newFieldset(level int, parent *etree.Element, name string, data interface{}, funcDatalist FuncDatalist, funcFieldIterator FuncFieldIterator) error {
	if reflect.TypeOf(data).Kind() == reflect.Ptr {
		data = reflect.ValueOf(data).Elem()
	}

	htmlLegend := parent.CreateElement("legend")
	htmlLegend.SetText(Translate(name))

	structValue := reflect.ValueOf(data)

	for i := 0; i < structValue.NumField(); i++ {
		fieldType := structValue.Type().Field(i)
		fieldValue := structValue.Field(i)

		if funcFieldIterator != nil && !funcFieldIterator(fieldType, fieldValue) {
			continue
		}

		Debug("%+v", fieldType)

		fieldTags, err := structtag.Parse(string(fieldType.Tag))
		if Error(err) {
			return err
		}

		tagForm, err := fieldTags.Get("form")
		if err != nil {
			continue
		}

		tagHtml, err := fieldTags.Get("html")
		if err != nil {
			continue
		}

		if fieldType.Type.Kind() == reflect.Struct {
			if i == 0 {
				parent.RemoveChildAt(0)
			}

			if funcFieldIterator != nil && !funcFieldIterator(fieldType, fieldValue) {
				continue
			}

			err = newFieldset(level+1, parent, tagHtml.Name, fieldValue.Interface(), funcDatalist, funcFieldIterator)
			if Error(err) {
				return err
			}

			continue
		}

		htmlDiv := parent.CreateElement("div")
		htmlDiv.CreateAttr("class", "pure-control-group")
		if indexOf(tagHtml.Options, OPTION_HIDDEN) != -1 {
			htmlDiv.CreateAttr("style", "display: none;")
		}

		htmlLabel := htmlDiv.CreateElement("label")
		htmlLabel.CreateAttr("for", tagForm.Name)

		if indexOf(tagHtml.Options, OPTION_NOLABEL) == -1 {
			htmlLabel.SetText(Translate(tagHtml.Name))
		}

		var values []string

		if funcDatalist != nil {
			values = funcDatalist(tagForm.Name)
		}

		if len(values) == 0 {
			datalist, err := fieldTags.Get("html_list")
			if err == nil {
				values = strings.Split(datalist.Value(), ",")
			}
		}

		var htmlInput *etree.Element

		switch fieldType.Type.Kind() {
		case reflect.Bool:
			htmlInput = htmlDiv.CreateElement("input")
			htmlInput.CreateAttr("type", "checkbox")
			htmlInput.CreateAttr("value", "true")
			if fieldValue.Bool() {
				htmlInput.CreateAttr("checked", "")
			}
		default:
			if indexOf(tagHtml.Options, OPTION_MULTILINE) != -1 {
				htmlInput = htmlDiv.CreateElement("textarea")
				htmlInput.CreateAttr("class", INPUT_WIDTH_WIDE)
				htmlInput.CreateAttr("cols", "65")
				htmlInput.CreateAttr("rows", "5")
				if fieldValue.String() != "" {
					htmlInput.SetText(fieldValue.String())
				}

				break
			}

			if indexOf(tagHtml.Options, OPTION_MEGALINE) != -1 {
				htmlInput = htmlDiv.CreateElement("textarea")
				htmlInput.CreateAttr("class", INPUT_WIDTH_WIDE)
				htmlInput.CreateAttr("cols", "65")
				htmlInput.CreateAttr("rows", "20")
				if fieldValue.String() != "" {
					htmlInput.SetText(fieldValue.String())
				}

				break
			}

			if indexOf(tagHtml.Options, OPTION_SELECT) != -1 {
				htmlInput = htmlDiv.CreateElement("select")
				htmlInput.CreateAttr("class", INPUT_WIDTH_NORMAL)

				preselectValue := fieldValue.String()
				if reflect.TypeOf(fieldValue.Interface()).Kind() == reflect.Int {
					preselectValue = strconv.Itoa(int(fieldValue.Int()))
				}

				if len(values) > 0 {
					for _, value := range values {
						htmlOption := htmlInput.CreateElement("option")
						htmlOption.CreateAttr("value", value)
						htmlOption.SetText(value)

						if value == preselectValue {
							htmlOption.CreateAttr("selected", "")
						}
					}
				}

				break
			}

			htmlInput = htmlDiv.CreateElement("input")
			htmlInput.CreateAttr("class", INPUT_WIDTH_NORMAL)

			if fieldType.Type.Kind() == reflect.Int {
				htmlInput.CreateAttr("type", "number")
				htmlInput.CreateAttr("value", fmt.Sprintf("%d", fieldValue.Int()))
				option, err := fieldTags.Get("html_min")
				if err == nil {
					htmlInput.CreateAttr("min", fmt.Sprintf("%s", option.Value()))
				}
				option, err = fieldTags.Get("html_max")
				if err == nil {
					htmlInput.CreateAttr("max", fmt.Sprintf("%s", option.Value()))
				}
				htmlInput.CreateAttr("onchange", fmt.Sprintf("document.getElementById(--$%s.range$--).value = this.value;", tagForm.Name))

				htmlRange := htmlDiv.CreateElement("input")
				htmlRange.CreateAttr("id", fmt.Sprintf("%s.range", tagForm.Name))
				htmlRange.CreateAttr("class", INPUT_WIDTH_NORMAL)

				htmlRange.CreateAttr("type", "range")
				htmlRange.CreateAttr("tabIndex", "-1")
				htmlRange.CreateAttr("value", fmt.Sprintf("%d", fieldValue.Int()))
				option, err = fieldTags.Get("html_min")
				if err == nil {
					htmlRange.CreateAttr("min", fmt.Sprintf("%s", option.Value()))
				}
				option, err = fieldTags.Get("html_max")
				if err == nil {
					htmlRange.CreateAttr("max", fmt.Sprintf("%s", option.Value()))
				}

				htmlRange.CreateAttr("oninput", fmt.Sprintf("document.getElementById(--$%s$--).value = this.value;", tagForm.Name))
			} else {
				if indexOf(tagHtml.Options, OPTION_FILE) != -1 {
					htmlInput.CreateAttr("type", "file")

					tagAccept, err := fieldTags.Get("accept")
					if err == nil {
						htmlInput.CreateAttr("accept", tagAccept.Name)
					}
				} else {
					if indexOf(tagHtml.Options, OPTION_PASSWORD) != -1 {
						htmlInput.CreateAttr("type", "password")
					} else {
						htmlInput.CreateAttr("type", "text")
					}
					htmlInput.CreateAttr("value", fmt.Sprintf("%s", fieldValue.String()))
				}
			}

			if indexOf(tagHtml.Options, OPTION_DATALIST) != -1 {
				htmlInput.CreateAttr("list", tagForm.Name+"_list")

				htmlDatalist := htmlDiv.CreateElement("datalist")
				htmlDatalist.CreateAttr("id", tagForm.Name+"_list")

				if len(values) > 0 {
					for _, value := range values {
						htmlOption := htmlDatalist.CreateElement("option")
						htmlOption.CreateAttr("value", value)
						htmlOption.SetText(value)
					}
				}
			}
		}

		htmlInput.CreateAttr("name", tagForm.Name)
		htmlInput.CreateAttr("id", tagForm.Name)
		htmlInput.CreateAttr("spellcheck", "false")

		if indexOf(tagHtml.Options, OPTION_NOPLACEHOLDER) == -1 {
			htmlInput.CreateAttr("placeholder", Translate(tagHtml.Name))
		}

		if indexOf(tagHtml.Options, OPTION_AUTOFOCUS) != -1 {
			htmlInput.CreateAttr("autofocus", "")
		}

		if indexOf(tagHtml.Options, OPTION_REQUIRED) != -1 {
			htmlInput.CreateAttr("required", "")
		}

		if indexOf(tagHtml.Options, OPTION_READONLY) != -1 {
			htmlInput.CreateAttr("readonly", "")
		}

		tagPattern, err := fieldTags.Get("html_pattern")
		if err == nil {
			pattern := tagPattern.Name

			pattern = strings.ReplaceAll(pattern, ";", ",")

			htmlInput.CreateAttr("pattern", pattern)
			htmlInput.CreateAttr("title", Translate("Invalid characters used in the input. Valid %v", pattern))
		}
	}

	return nil
}
func NewButton(parent *etree.Element, primary bool, actionItem ActionItem) {
	button := parent.CreateElement("input")

	button.CreateAttr("value", actionItem.Caption)

	if actionItem.Action != "submit" && actionItem.Action != "reset" {
		button.CreateAttr("type", "button")
		button.CreateAttr("onclick", "location.href=--$"+actionItem.Action+"$--")
	} else {
		button.CreateAttr("type", actionItem.Action)
	}

	if primary {
		button.CreateAttr("class", "pure-button pure-button-primary")
	} else {
		button.CreateAttr("class", "pure-button")
	}
}

func NewTable(parent *etree.Element, cells [][]string) {
	htmlTable := parent.CreateElement("table")
	htmlTable.CreateAttr("class", "pure-table pure-table.bordered")

	var htmlHeader, htmlRow *etree.Element
	var tagName string

	for rowIndex, row := range cells {
		if rowIndex == 0 {
			htmlHeader = htmlTable.CreateElement("thead")
			htmlRow = htmlHeader.CreateElement("tr")
			tagName = "th"
		} else {
			htmlHeader = htmlTable.CreateElement("tbody")
			htmlRow = htmlHeader.CreateElement("tr")
			tagName = "td"
		}

		if rowIndex%2 == 1 {
			htmlRow.CreateAttr("class", "pure-table-odd")
		}

		for _, cell := range row {
			htmlCell := htmlRow.CreateElement(tagName)
			htmlCell.SetText(cell)
		}
	}
}

func (this *Webpage) HTML() (string, error) {
	this.doc.Indent(4)
	this.doc.WriteSettings = etree.WriteSettings{
		CanonicalEndTags: true,
		CanonicalText:    true,
		CanonicalAttrVal: false,
		UseCRLF:          false,
	}

	html, err := this.doc.WriteToString()
	if Error(err) {
		return "", err
	}

	// icons
	html = strings.ReplaceAll(html, "--%", "<i class=\"")
	html = strings.ReplaceAll(html, "%--", "\" style=\"width:22px;text-align: center;\"></i>\n")

	// suppress masking
	html = strings.ReplaceAll(html, "--[", "<")
	html = strings.ReplaceAll(html, "]--", ">")
	html = strings.ReplaceAll(html, "--$", "'")
	html = strings.ReplaceAll(html, "$--", "'")
	html = strings.ReplaceAll(html, "=\"\"", "")

	// preserve empty href attribute"

	html = strings.ReplaceAll(html, "<a href ", "<a href=\"\" ")

	strayEnds := []string{"></link>", "></meta>", "></input>"}
	for _, strayEnd := range strayEnds {
		html = strings.ReplaceAll(html, strayEnd, "/>")
	}

	html = fmt.Sprintf("<!DOCTYPE html>\n%s", html)

	return html, nil
}
