package common

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/beevik/etree"
	"github.com/fatih/structtag"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/quasoft/memstore"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	OPTION_FILE        = "file"
	OPTION_HIDDEN      = "hidden"
	OPTION_SELECT      = "select"
	OPTION_MULTISELECT = "multiselect"
	OPTION_DATALIST    = "datalist"
	OPTION_PASSWORD    = "password"
	OPTION_AUTOFOCUS   = "autofocus"
	OPTION_REQUIRED    = "required"
	OPTION_READONLY    = "readonly"
	OPTION_MULTILINE   = "multiline"
	OPTION_WIDE        = "wide"
	OPTION_MEGALINE    = "megaline"
	OPTION_NOLABEL     = "nolabel"
	OPTION_EXPERTVIEW  = "expertview"

	INPUT_WIDTH_NORMAL = "pure-input-1-4"
	INPUT_WIDTH_WIDE   = "pure-input-1-2"

	CSS_DIALOG_CONTENT     = "css-dialog-content pure-controls"
	CSS_DEFAULT_CONTENT    = "css-default-content pure-controls"
	CSS_ERROR_BOX          = "css-error-box"
	CSS_WARNING_BOX        = "css-warning-box blink"
	CSS_SUCCESS_BOX        = "css-success-box"
	CSS_VERTICAL_DIV       = "css-vertical-div"
	CSS_LOGVIEWER          = "css-logviewer"
	CSS_CONTENT            = "css-content"
	CSS_CHECKLIST          = "css-checklist"
	CSS_CHECKLIST_CHECKBOX = "css-checklist-checkbox"
	CSS_CHECKBOX           = "css-checkbox"
	CSS_BUTTON_GROUP       = "css-button-group"
	CSS_BORDER             = "css-border"

	CSS_COLOR_OFF = "#CFCFCF"
	CSS_COLOR_ON  = "MediumSeaGreen"

	MAX_INPUT_LENGTH = "10240"

	FLASH_WARNING = "warning-flash"
	FLASH_ERROR   = "error-flash"
	FLASH_SUCCESS = "success-flash"

	COOKIE_PASSWORD = "password"
	COOKIE_EXPIRE   = "expire"

	FLASH_TIMEOUT   = time.Second
	REFRESH_TIMEOUT = time.Second * 30
)

var (
	SessionStore *memstore.MemStore
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
	redirectTimeout   time.Duration
	redirectUrl       string
	onLoad            string
}

type FuncFieldIterator func(string, reflect.StructField, reflect.Value, *structtag.Tag) (bool, []string)

type ActionItem struct {
	Caption  string
	Icon     string
	Action   string
	Download string
	Message  string
	Enabled  bool
	SubItems []ActionItem
}

func init() {
	storeSecret, err := RndBytes(32)
	Panic(err)

	SessionStore = memstore.NewMemStore(storeSecret)
	SessionStore.Options.Secure = true
	SessionStore.Options.SameSite = http.SameSiteStrictMode
	SessionStore.Options.HttpOnly = true
	SessionStore.Options.MaxAge = 0
}

func NewPage(context echo.Context, contentStyle string, title string) (*Webpage, error) {
	p := Webpage{doc: etree.NewDocument()}

	p.HtmlRoot = p.doc.CreateElement("html")

	lang := *FlagLanguage
	if lang == "" {
		lang = DEFAULT_LANGUAGE
	}
	p.HtmlRoot.CreateAttr("lang", lang)

	p.HtmlHead = p.HtmlRoot.CreateElement("head")
	p.HtmlTitle = p.HtmlHead.CreateElement("title")
	p.HtmlTitle.SetText(title)

	htmlMeta := p.HtmlHead.CreateElement("meta")
	htmlMeta.CreateAttr("name", "viewport")
	htmlMeta.CreateAttr("content", "width = device-width, initial-scale = 1")

	htmlMeta = p.HtmlHead.CreateElement("meta")
	htmlMeta.CreateAttr("http-equiv", "X-UA-Compatible")
	htmlMeta.CreateAttr("content", "IE=Edge")

	htmlMeta = p.HtmlHead.CreateElement("meta")
	htmlMeta.CreateAttr("charset", "UTF-8")

	htmlMeta = p.HtmlHead.CreateElement("meta")
	htmlMeta.CreateAttr("http-equiv", "cache-control")
	htmlMeta.CreateAttr("content", "no-cache, no-store, must-revalidate")

	p.HtmlBody = p.HtmlRoot.CreateElement("body")

	p.HtmlMenu = p.HtmlBody.CreateElement("div")

	p.HtmlScrollContent = p.HtmlBody.CreateElement("div")
	p.HtmlScrollContent.CreateAttr("class", CSS_CONTENT)

	p.HtmlContent = p.HtmlScrollContent.CreateElement("div")

	if contentStyle != "" {
		p.HtmlContent.CreateAttr("class", contentStyle)
	}

	if context != nil {
		msgs := PullFlash(context, FLASH_WARNING)
		if msgs != nil {
			htmlError := p.HtmlContent.CreateElement("div")
			htmlError.CreateAttr("class", CSS_WARNING_BOX)
			htmlError.SetText(strings.Join(msgs, "??br??br"))
		}

		msgs = PullFlash(context, FLASH_SUCCESS)
		if msgs != nil {
			htmlError := p.HtmlContent.CreateElement("div")
			htmlError.CreateAttr("class", CSS_SUCCESS_BOX)
			htmlError.SetText(strings.Join(msgs, "??br??br"))
		}

		msgs = PullFlash(context, FLASH_ERROR)
		if msgs != nil {
			htmlError := p.HtmlContent.CreateElement("div")
			htmlError.CreateAttr("class", CSS_ERROR_BOX)
			htmlError.SetText(strings.Join(msgs, "??br??br"))
		}
	}

	return &p, nil
}

func PullFlash(context echo.Context, flashName string) []string {
	cookie, err := GetCookie(context)
	if Error(err) {
		return nil
	}
	if cookie != nil {
		flashes := cookie.Flashes(flashName)
		if len(flashes) > 0 {
			flash := strings.Split(flashes[0].(string), "??br")

			err := SetCookie(context, cookie, REFRESH_TIMEOUT)
			if Error(err) {
				return nil
			}

			return flash
		}
	}

	return nil
}

func PushFlash(context echo.Context, flashName string, flash string) error {
	list := PullFlash(context, flashName)

	list = append(list, flash)

	cookie, err := GetCookie(context)
	if Error(err) {
		return nil
	}
	if cookie != nil {
		cookie.AddFlash(strings.Join(list, "??br"), flashName)
	}

	err = SetCookie(context, cookie, REFRESH_TIMEOUT)
	if Error(err) {
		return err
	}

	return nil
}

func GetCookie(context echo.Context) (*sessions.Session, error) {
	cookie, _ := session.Get(Title(), context)

	// Ignore the error (maybe the current cookie was encrypted with an outdated httpServer.store key)

	if cookie.IsNew {
		cookie.Options.Path = "/"
		cookie.Options.MaxAge = 0
		cookie.Options.Secure = SessionStore.Options.Secure
		cookie.Options.SameSite = SessionStore.Options.SameSite
		cookie.Options.HttpOnly = SessionStore.Options.HttpOnly
	}

	err := SetCookie(context, cookie, REFRESH_TIMEOUT)
	if Error(err) {
		return nil, err
	}

	return cookie, nil
}

func DisableCookie(context echo.Context) error {
	cookie, err := GetCookie(context)
	if Error(err) {
		return nil
	}

	cookie.Options.MaxAge = -1
	delete(cookie.Values, COOKIE_PASSWORD)
	delete(cookie.Values, COOKIE_EXPIRE)

	err = SetCookie(context, cookie, REFRESH_TIMEOUT)
	if Error(err) {
		return err
	}

	return nil
}

func SetCookie(context echo.Context, cookie *sessions.Session, timeout time.Duration) error {
	cookie.Values[COOKIE_EXPIRE] = fmt.Sprintf("%s", time.Now().Add(timeout).Format(DateTimeMask))

	err := SessionStore.Save(context.Request(), context.Response(), cookie)
	if Error(err) {
		return err
	}

	return nil
}

func AuthenticateCookie(context echo.Context, password string) error {
	cookie, err := GetCookie(context)
	if Error(err) {
		return nil
	}

	// Ignore the error (maybe the current cookie was encrypted with an outdated httpServer.store key)

	cookie.Values[COOKIE_PASSWORD] = password

	err = SetCookie(context, cookie, REFRESH_TIMEOUT)
	if Error(err) {
		return err
	}

	return nil
}

func CheckCookieAuthenticated(context echo.Context, password string) error {
	cookie, err := GetCookie(context)
	if Error(err) {
		return nil
	}

	expire, ok := cookie.Values[COOKIE_EXPIRE]
	if !ok {
		return fmt.Errorf("cookie %s attribute %s not available", Title(), COOKIE_EXPIRE)
	}

	expireTime, err := ParseDateTime(DateTimeMask, expire.(string))
	if Error(err) {
		return fmt.Errorf("cookie %s expire date attribute can not be parsed: %s", Title(), expire.(string))
	}

	now := time.Now()
	if now.After(expireTime) {
		return fmt.Errorf("cookie %s expire date reached. expire %s, now %s", Title(), expireTime.Format(DateTimeMask), now.Format(DateTimeMask))
	}

	cookiePassword, ok := cookie.Values[COOKIE_PASSWORD]
	if !ok {
		return fmt.Errorf("cookie %s attribute %s not available", Title(), COOKIE_PASSWORD)
	}

	if password == "" {
		return fmt.Errorf("login password not defined")
	}

	if password != cookiePassword {
		return fmt.Errorf("password does not equal defined password")
	}

	return nil
}

func IsCookieAuthenticated(context echo.Context, password string) bool {
	err := CheckCookieAuthenticated(context, password)

	if err != nil {
		DebugFunc(err.Error())

		return false
	}

	return true
}

func (page *Webpage) SetRedirectTimeout(timeout time.Duration, url string) {
	page.redirectTimeout = timeout
	page.redirectUrl = url
}

func NewMenu(page *Webpage, menuItems []ActionItem, selectedTitle string, disableMenues bool) {
	page.HtmlMenu.CreateAttr("class", "pure-menu pure-menu-horizontal")

	newMenuitem(page.HtmlMenu, true, menuItems, selectedTitle, disableMenues)
}

func newMenuitem(parent *etree.Element, mainMenu bool, menuItems []ActionItem, selectedTitle string, disableMenues bool) {
	htmlUl := parent.CreateElement("ul")
	if mainMenu {
		htmlUl.CreateAttr("class", "pure-menu-list")
	} else {
		htmlUl.CreateAttr("class", "pure-menu-children")
	}

	for _, menu := range menuItems {
		classes := []string{"pure-menu-item"}

		isMenuDisabled := !menu.Enabled || disableMenues
		if isMenuDisabled {
			classes = append(classes, "pure-menu-disabled")
		} else {
			if menu.Caption == selectedTitle {
				classes = append(classes, "pure-menu-selected")
			}

			if len(menu.SubItems) > 0 {
				classes = append(classes, "pure-menu-has-children")
				classes = append(classes, "pure-menu-allow-hover")
			}
		}

		htmlMenu := htmlUl.CreateElement("li")
		htmlMenu.CreateAttr("class", strings.Join(classes, " "))

		htmlAhref := htmlMenu.CreateElement("a")
		htmlAhref.CreateAttr("class", "pure-menu-link")
		textAlign := "left"
		if mainMenu {
			textAlign = "center"
		}
		htmlAhref.CreateAttr("style", fmt.Sprintf("cursor: pointer;text-align: %s;", textAlign))

		if menu.Caption != "" {
			caption := menu.Caption
			if menu.Icon != "" {
				if strings.HasPrefix(menu.Icon, "data:") {
					htmlImg := htmlAhref.CreateElement("img")
					htmlImg.CreateAttr("src", menu.Icon)
					htmlImg.CreateAttr("alt", "")
					htmlImg.CreateAttr("style", "float:left;display: inline-block;width: 14px;height: 14px;margin-right: 4px;text-align: center;")
				} else {
					caption = "--%" + menu.Icon + "%--" + caption
				}
			}

			htmlAhref.SetText(caption)
		}

		if !isMenuDisabled {
			if len(menu.SubItems) > 0 {
				htmlAhref.CreateAttr("onClick", "return false;")
			} else {
				if strings.Index(menu.Action, ";") != -1 {
					if menu.Message != "" {
						htmlAhref.CreateAttr("onClick", fmt.Sprintf("if(confirm(--$%s$--)) { %s }", menu.Message, menu.Action))
					} else {
						htmlAhref.CreateAttr("onClick", menu.Action)
					}
				} else {
					if menu.Message != "" {
						htmlAhref.CreateAttr("onClick", fmt.Sprintf("if(confirm(--$%s$--)) { window.location.replace(--$%s$--); }", menu.Message, menu.Action))
					} else {
						htmlAhref.CreateAttr("href", menu.Action)
					}
				}
			}

			if len(menu.SubItems) > 0 {
				newMenuitem(htmlMenu, false, menu.SubItems, selectedTitle, disableMenues)
			}

			if menu.Download != "" {
				htmlAhref.CreateAttr("onClick", fmt.Sprintf("{ window.open(--$%s$--,--$_blank$--); window.location.replace(--$/$--); }", menu.Download))
			}
		}
	}
}

func NewForm(parent *etree.Element, caption string, data interface{}, defaultData interface{}, method string, formAction string, actions []ActionItem, readOnly bool, isExpertViewAvailable bool, funcFieldIterator FuncFieldIterator) (*etree.Element, *etree.Element, error) {
	htmlForm := parent.CreateElement("form")
	htmlForm.CreateAttr("method", method)
	htmlForm.CreateAttr("enctype", echo.MIMEMultipartForm)
	htmlForm.CreateAttr("class", "pure-form pure-form-aligned")

	htmlForm.CreateAttr("action", formAction)
	htmlForm.CreateAttr("method", method)

	htmlGroup := htmlForm.CreateElement("div")
	htmlGroup.CreateAttr("class", CSS_BUTTON_GROUP)
	htmlGroup.CreateAttr("style", "display:flex;")

	htmlGroupH1 := htmlGroup.CreateElement("h1")
	htmlGroupH1.SetText(caption)
	htmlGroupH1.CreateAttr("style", "margin:auto;")

	htmlGroupCenter := htmlGroup.CreateElement("div")

	isFieldExpertView, err := newFieldset(true, htmlForm, caption, data, defaultData, "", readOnly, isExpertViewAvailable, funcFieldIterator)
	if Error(err) {
		return nil, nil, err
	}

	for i, action := range actions {
		NewButton(htmlGroupCenter, i == 0, action)
	}

	if isFieldExpertView {
		expertViewCheckbox := newCheckbox(htmlGroupCenter, isExpertViewAvailable)
		expertViewCheckbox.SetText(Translate("Expert view"))
		expertViewCheckbox.CreateAttr("onClick", fmt.Sprintf("setExpertViewVisible(--$fieldset$--);"))
		expertViewCheckbox.CreateAttr("style", "display:flex;")
	} else {
		if len(actions) == 0 {
			htmlForm.RemoveChild(htmlGroup)
		}
	}

	return htmlForm, htmlGroup, nil
}

func BindForm(context echo.Context, data interface{}, bodyLimit int) error {
	if strings.Index(context.Request().Header.Get("Content-Type"), "multipart/form-data") == -1 {
		return fmt.Errorf("no multipart/form-data request: %+v", context.Request())
	}

	err := context.Request().ParseMultipartForm(int64(bodyLimit))
	if Error(err) {
		return err
	}

	err = IterateStruct(data, func(fieldPath string, fieldType reflect.StructField, fieldValue reflect.Value) error {
		_, ok := context.Request().Form[fieldPath]
		if !ok {
			_, ok = context.Request().MultipartForm.File[fieldPath]
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			break
		case reflect.Bool:
			if ok {
				fieldValue.SetBool(true)
			} else {
				fieldValue.SetBool(false)
			}
		case reflect.Int:
			if ok {
				formValue := context.Request().FormValue(fieldPath)
				if formValue == "" {
					formValue = "0"
				}

				i, err := strconv.Atoi(formValue)
				if Error(err) {
					break
				}
				fieldValue.SetInt(int64(i))
			}
		case reflect.String:
			if ok {
				values := context.Request().Form[fieldPath]

				if len(values) > 0 {
					fieldValue.SetString(strings.Join(values, ";"))
				}
			}
		case reflect.Slice:
			if ok {
				switch reflect.TypeOf(fieldValue.Interface()).Elem().Kind() {
				case reflect.String:
					values := context.Request().Form[fieldPath]
					scanner := bufio.NewScanner(strings.NewReader(values[0]))
					lines := make([]string, 0)
					for scanner.Scan() {
						line := strings.TrimSpace(scanner.Text())
						if len(line) > 0 {
							if !strings.Contains(line, "=") {
								line = line + "="
							}

							lines = append(lines, line)
						}
					}
					sort.Strings(lines)

					if fieldValue.Type().AssignableTo(reflect.TypeOf(KeyValues{})) {
						kvs := NewKeyValues(lines)
						fieldValue.Set(reflect.ValueOf(*kvs))
					} else {
						fieldValue.Set(reflect.ValueOf(lines))
					}
				default:
					_, file, err := context.Request().FormFile(fieldPath)
					if file == nil {
						return nil
					}
					if Error(err) {
						return err
					}

					src, err := file.Open()
					if Error(err) {
						return err
					}
					defer func() {
						Error(src.Close())
					}()

					var buf bytes.Buffer

					_, err = io.Copy(&buf, src)
					if Error(err) {
						return err
					}

					fieldValue.SetBytes([]byte(base64.StdEncoding.EncodeToString(buf.Bytes())))
				}
			}
		default:
			return fmt.Errorf("unsupported field: %s", fieldPath)
		}

		return nil
	})

	return err
}

func newCheckbox(parent *etree.Element, value bool) *etree.Element {
	var htmlInput *etree.Element

	if parent == nil {
		htmlInput = etree.NewElement("input")
	} else {
		htmlInput = parent.CreateElement("input")
	}

	htmlInput.CreateAttr("type", "checkbox")
	htmlInput.CreateAttr("class", CSS_CHECKBOX)
	htmlInput.CreateAttr("value", "true")

	if value {
		htmlInput.CreateAttr("checked", "")
	}

	return htmlInput
}

func newFieldset(isFirstFieldset bool, parent *etree.Element, caption string, data interface{}, dataDefault interface{}, path string, readOnly bool, isExpertViewActive bool, funcFieldIterator FuncFieldIterator) (bool, error) {
	parent = parent.CreateElement("fieldset")

	htmlLegend := parent.CreateElement("legend")
	htmlLegend.SetText(Translate(caption))

	// -----------------------------------------------------......................

	if reflect.TypeOf(data).Kind() == reflect.Ptr {
		data = reflect.ValueOf(data).Elem()
	}

	useDefaultValue := dataDefault != nil

	if useDefaultValue && reflect.TypeOf(dataDefault).Kind() == reflect.Ptr {
		dataDefault = reflect.ValueOf(dataDefault).Elem()
	}

	structValue := reflect.ValueOf(data)

	var structValueDefault reflect.Value
	if useDefaultValue {
		structValueDefault = reflect.ValueOf(dataDefault)
	}

	// -----------------------------------------------------......................

	useDefaultValueBackup := useDefaultValue
	expertViewFieldExists := false
	allFieldsAreExpertView := true
	hasMultipleFieldsets := false

	for i := 0; i < structValue.NumField(); i++ {
		useDefaultValue = useDefaultValueBackup

		field := structValue.Type().Field(i)
		fieldPath := field.Name
		if path != "" {
			fieldPath = strings.Join([]string{path, fieldPath}, "_")
		}

		fieldValue := structValue.Field(i)
		fieldValueOptions := []string{}

		var fieldValueDefault reflect.Value
		if useDefaultValue {
			fieldValueDefault = structValueDefault.Field(i)
		}

		fieldTags, err := structtag.Parse(string(field.Tag))
		if Error(err) {
			return false, err
		}

		tagHtml, err := fieldTags.Get("html")
		if err != nil {
			continue
		}

		if funcFieldIterator != nil {
			var fieldVisible bool

			fieldVisible, fieldValueOptions = funcFieldIterator(fieldPath, field, fieldValue, tagHtml)

			if !fieldVisible {
				continue
			}
		}

		Debug("%+v", field)

		if field.Type.Kind() == reflect.Struct {
			if i == 0 {
				parent.RemoveChildAt(0)
			}

			var ev bool

			hasMultipleFieldsets = true

			subStruct := fieldValue.Interface()
			var subStructDefault interface{}
			if useDefaultValue {
				subStructDefault = fieldValueDefault.Interface()
			}

			ev, err = newFieldset(false, parent, tagHtml.Name, subStruct, subStructDefault, fieldPath, readOnly, isExpertViewActive, funcFieldIterator)
			if Error(err) {
				return false, err
			}

			expertViewFieldExists = expertViewFieldExists || ev

			continue
		}

		htmlDiv := parent.CreateElement("div")

		classes := []string{"pure-control-group"}

		isFieldExpertView := IndexOf(tagHtml.Options, OPTION_EXPERTVIEW) != -1
		isFieldHidden := (IndexOf(tagHtml.Options, OPTION_HIDDEN) != -1) || (isFieldExpertView && !isExpertViewActive)
		isFieldReadOnly := (IndexOf(tagHtml.Options, OPTION_READONLY) != -1)
		isFieldPassword := (IndexOf(tagHtml.Options, OPTION_PASSWORD) != -1)

		useDefaultValue = useDefaultValue && !isFieldReadOnly && !isFieldPassword

		expertViewFieldExists = expertViewFieldExists || isFieldExpertView

		if !isFieldExpertView {
			allFieldsAreExpertView = false
		}

		if isFieldExpertView {
			classes = append(classes, OPTION_EXPERTVIEW)
		}

		htmlDiv.CreateAttr("class", strings.Join(classes, " "))

		if isFieldHidden {
			htmlDiv.CreateAttr("style", "display: none;")
		}

		htmlLabel := htmlDiv.CreateElement("label")
		htmlLabel.CreateAttr("for", fieldPath)

		if IndexOf(tagHtml.Options, OPTION_NOLABEL) == -1 {
			htmlLabel.SetText(Translate(tagHtml.Name))
		}

		var htmlInput *etree.Element

		switch field.Type.Kind() {
		case reflect.Bool:
			htmlInput = newCheckbox(htmlDiv, fieldValue.Bool())
		default:
			if IndexOf(tagHtml.Options, OPTION_MULTILINE) != -1 || IndexOf(tagHtml.Options, OPTION_MEGALINE) != -1 {
				htmlInput = htmlDiv.CreateElement("textarea")
				htmlInput.CreateAttr("class", INPUT_WIDTH_WIDE)
				htmlInput.CreateAttr("maxlength", MAX_INPUT_LENGTH)
				htmlInput.CreateAttr("cols", "65")

				if IndexOf(tagHtml.Options, OPTION_MULTILINE) != -1 {
					htmlInput.CreateAttr("rows", "5")
				} else {
					htmlInput.CreateAttr("rows", "20")
				}

				if reflect.TypeOf(fieldValue.Interface()).Kind() == reflect.Slice && reflect.TypeOf(fieldValue.Interface()).Elem().Kind() == reflect.String {
					lines := make([]string, 0)
					for i := 0; i < fieldValue.Len(); i++ {
						lines = append(lines, fieldValue.Index(i).String())
					}
					sort.Strings(lines)
					htmlInput.SetText(strings.Join(lines, "\n"))
				} else {
					if !fieldValue.IsZero() {
						htmlInput.SetText(fieldValue.String())
					}
				}

				break
			}

			if IndexOf(tagHtml.Options, OPTION_MULTISELECT) != -1 {
				htmlSpan := htmlDiv.CreateElement("span")
				htmlSpan.CreateAttr("id", fieldPath+".span")
				htmlSpan.CreateAttr("class", CSS_CHECKLIST)

				preselectValue := fieldValue.String()
				if reflect.TypeOf(fieldValue.Interface()).Kind() == reflect.Int {
					preselectValue = strconv.Itoa(int(fieldValue.Int()))
				}

				preselectedValues := make(map[string]bool)

				list := strings.Split(preselectValue, ";")
				for _, v := range list {
					preselectedValues[v] = true
				}

				for _, value := range fieldValueOptions {
					htmlItem := htmlSpan.CreateElement("input")
					htmlItem.CreateAttr("type", "checkbox")
					htmlItem.CreateAttr("value", value)
					htmlItem.CreateAttr("name", fieldPath)
					htmlItem.CreateAttr("id", fieldPath+"."+value)
					htmlItem.CreateAttr("class", CSS_CHECKLIST_CHECKBOX)
					htmlItem.CreateAttr("onkeypress", "multiCheck(event);")
					htmlItem.CreateAttr("data-default-value", strconv.FormatBool(preselectedValues[value]))
					htmlItem.CreateAttr("onclick", fmt.Sprintf("checkDefaultValue(--$%s$--);", fieldPath+"."+value))
					htmlItem.SetText(value)

					if preselectedValues[value] {
						htmlItem.CreateAttr("checked", "")
					}

					if IndexOf(tagHtml.Options, OPTION_AUTOFOCUS) != -1 {
						htmlItem.CreateAttr("autofocus", "")
					}

					if IndexOf(tagHtml.Options, OPTION_READONLY) != -1 {
						htmlItem.CreateAttr("readonly", "")
					}

					htmlSpan.CreateElement("br")
				}

				continue
			}

			if IndexOf(tagHtml.Options, OPTION_SELECT) != -1 {
				htmlInput = htmlDiv.CreateElement("select")
				htmlInput.CreateAttr("class", INPUT_WIDTH_NORMAL)

				preselectValue := fieldValue.String()
				if reflect.TypeOf(fieldValue.Interface()).Kind() == reflect.Int {
					preselectValue = strconv.Itoa(int(fieldValue.Int()))
				}
				htmlInput.CreateAttr("data-default-value", preselectValue)

				for _, value := range fieldValueOptions {
					htmlOption := htmlInput.CreateElement("option")
					htmlOption.SetText(value)

					if value == preselectValue {
						htmlOption.CreateAttr("selected", "")
					}
				}

				break
			}

			htmlInput = htmlDiv.CreateElement("input")
			if IndexOf(tagHtml.Options, OPTION_WIDE) != -1 {
				htmlInput.CreateAttr("class", INPUT_WIDTH_WIDE)
			} else {
				htmlInput.CreateAttr("class", INPUT_WIDTH_NORMAL)
			}
			htmlInput.CreateAttr("onclick", "this.select();")

			if field.Type.Kind() == reflect.Int {
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

				if !readOnly && !isFieldReadOnly {
					htmlInput.CreateAttr("onchange", fmt.Sprintf("document.getElementById(--$%s.range$--).value = this.value;", fieldPath))

					htmlRange := htmlDiv.CreateElement("input")
					htmlRange.CreateAttr("id", fmt.Sprintf("%s.range", fieldPath))
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

					htmlRange.CreateAttr("oninput", fmt.Sprintf("document.getElementById(--$%s$--).value = this.value;checkDefaultValue(--$%s$--);", fieldPath, fieldPath))  // does NOT work with IE10
					htmlRange.CreateAttr("onchange", fmt.Sprintf("document.getElementById(--$%s$--).value = this.value;checkDefaultValue(--$%s$--);", fieldPath, fieldPath)) // IE 10 only
				}
			} else {
				if IndexOf(tagHtml.Options, OPTION_FILE) != -1 {
					htmlInput.CreateAttr("type", "file")

					useDefaultValue = false

					tagAccept, err := fieldTags.Get("accept")
					if err == nil {
						htmlInput.CreateAttr("accept", tagAccept.Name)
					}
					htmlInput.CreateAttr("style", "width: 250px;")

					if !readOnly && !isFieldReadOnly {
						button := htmlDiv.CreateElement("button")
						button.CreateAttr("class", "pure-button")
						button.CreateAttr("onclick", fmt.Sprintf("document.getElementById(--$%s$--).value = --$$--;var desc = document.getElementById(--$%sDescription$--); if (desc) { desc.value = --$ $--; };", fieldPath, fieldPath))

						icon := button.CreateElement("i")
						icon.CreateAttr("class", "fa fa-trash")
					}
				} else {
					if isFieldPassword {
						htmlInput.CreateAttr("type", "password")
					} else {
						htmlInput.CreateAttr("type", "text")
					}
					htmlInput.CreateAttr("value", fieldValue.String())
					htmlInput.CreateAttr("maxlength", MAX_INPUT_LENGTH)
				}
			}

			if IndexOf(tagHtml.Options, OPTION_DATALIST) != -1 {
				htmlInput.CreateAttr("list", fieldPath+".datalist")

				htmlDatalist := htmlDiv.CreateElement("datalist")
				htmlDatalist.CreateAttr("id", fieldPath+".datalist")

				for _, value := range fieldValueOptions {
					htmlOption := htmlDatalist.CreateElement("option")
					htmlOption.SetText(value)
				}
			}
		}

		htmlInput.CreateAttr("name", fieldPath)
		htmlInput.CreateAttr("id", fieldPath)
		htmlInput.CreateAttr("spellcheck", "false")

		if useDefaultValue {
			defaultValue := fmt.Sprintf("%v", fieldValueDefault)
			if defaultValue == "" {
				defaultValue = "nil"
			}
			htmlInput.CreateAttr("data-default-value", defaultValue)
			if field.Type.Kind() == reflect.Bool {
				htmlInput.CreateAttr("onclick", fmt.Sprintf("checkDefaultValue(--$%s$--);", fieldPath))
			} else {
				htmlInput.CreateAttr("oninput", fmt.Sprintf("checkDefaultValue(--$%s$--);", fieldPath))
			}
		}

		if isFieldReadOnly {
			htmlInput.CreateAttr("tabIndex", "-1")
		}

		if IndexOf(tagHtml.Options, OPTION_AUTOFOCUS) != -1 {
			htmlInput.CreateAttr("autofocus", "")
		}

		if IndexOf(tagHtml.Options, OPTION_REQUIRED) != -1 {
			htmlInput.CreateAttr("required", "")
		}

		if readOnly || IndexOf(tagHtml.Options, OPTION_READONLY) != -1 {
			htmlInput.CreateAttr("readonly", "")
		}

		tagPattern, err := fieldTags.Get("html_pattern")
		if err == nil {
			pattern := tagPattern.Name

			pattern = strings.ReplaceAll(pattern, ";", ",")

			htmlInput.CreateAttr("pattern", pattern)
		}
	}

	if isFirstFieldset && !hasMultipleFieldsets {
		parent.RemoveChild(htmlLegend)
	}

	if allFieldsAreExpertView {
		htmlLegend.CreateAttr("class", OPTION_EXPERTVIEW)

		if !isExpertViewActive {
			htmlLegend.CreateAttr("style", "display: none;")
		}

	}

	return expertViewFieldExists, nil
}
func NewButton(parent *etree.Element, primary bool, actionItem ActionItem) *etree.Element {
	button := parent.CreateElement("input")

	button.CreateAttr("value", actionItem.Caption)

	if actionItem.Action != "submit" && actionItem.Action != "reset" {
		button.CreateAttr("type", "button")
		if strings.Index(actionItem.Action, ";") != -1 {
			button.CreateAttr("onclick", actionItem.Action)
		} else {
			if actionItem.Message != "" {
				button.CreateAttr("onClick", fmt.Sprintf("if(confirm(--$%s$--)) { location.href=--$%s$--; }", actionItem.Message, actionItem.Action))
			} else {
				button.CreateAttr("onclick", fmt.Sprintf("location.href=--$%s$--", actionItem.Action))
			}
		}
	} else {
		button.CreateAttr("type", actionItem.Action)
	}

	if primary {
		button.CreateAttr("class", "pure-button pure-button-primary")
	} else {
		button.CreateAttr("class", "pure-button")
	}

	return button
}

func NewTable(parent *etree.Element, cells [][]string) *etree.Element {
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

	return htmlTable
}

func (this *Webpage) HTML() (string, error) {
	var onLoad strings.Builder

	if this.redirectUrl != "" {
		onLoad.WriteString(fmt.Sprintf("setTimeout(function() {redirectToUrl(--$%s$--);}, %d)", this.redirectUrl, DurationToMillisecond(this.redirectTimeout)))
	}

	attr := this.HtmlBody.SelectAttr("onload")
	if attr != nil {
		onLoad.WriteString(";")
		onLoad.WriteString(attr.Value)
	}

	if onLoad.Len() > 0 {
		this.HtmlBody.CreateAttr("onload", onLoad.String())
	}

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
	html = strings.ReplaceAll(html, "??br", "<br/>")

	if this.HtmlContent != nil && this.HtmlContent.SelectAttr("class") == nil {
		r := regexp.MustCompile(" class\\=\\\".*?\\\"")
		html = r.ReplaceAllString(html, "")
	}

	// preserve empty href attribute"

	html = strings.ReplaceAll(html, "<a href ", "<a href=\"\" ")

	strayEnds := []string{"></link>", "></meta>", "></input>", "></img>", "></br>"}
	for _, strayEnd := range strayEnds {
		html = strings.ReplaceAll(html, strayEnd, "/>")
	}

	html = fmt.Sprintf("<!DOCTYPE html>\n%s", html)

	return html, nil
}

func Hack4BrowserUpdate() string {
	if IsRunningAsExecutable() {
		return ""
	}

	return "?" + strconv.Itoa(Rnd(999999999))
}

func SessionId(context echo.Context) string {
	cookie, err := GetCookie(context)
	if cookie == nil || Error(err) {
		return ""
	}
	return cookie.ID
}
