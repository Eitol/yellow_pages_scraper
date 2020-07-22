package scraper

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gojetpack/pyos"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

const defaultBaseUrl = "https://www.paginasamarillas.com.uy"
const categoriesCacheFile = "categories.json"
const publicationListDir = "publicationList"
const publicationDir = "publications"

func httpGet(url string) ([]byte, int, error) {
	resp, err := http.Get(url)
	defer func() {
		if resp.Body == nil {
			return
		}
		err := resp.Body.Close()
		if err != nil {
			log.Print(err)
		}
	}()
	data, err := ioutil.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

type Page struct {
	Word   rune
	Number int
}

func BeginPage() Page {
	return Page{
		Word:   'A',
		Number: 1,
	}
}

func EndPage() Page {
	return Page{
		Word:   'Z',
		Number: -1,
	}
}

type CachePolicy struct {
	UseCacheForCategories             bool
	UseCacheForCategoriesPublications bool
	UseCacheForPublications           bool
	OutPath                           string
}

type YellowPagesUruguayScraper struct {
	MaxThreads int
	BaseUrl    string
}

func (y *YellowPagesUruguayScraper) setup() {
	if y.BaseUrl == "" {
		y.BaseUrl = defaultBaseUrl
	}
	if y.MaxThreads == 0 {
		y.MaxThreads = 1
	}

}

func _getQueryDoc(data []byte) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func respToCategoriesList(data []byte, target *[]Category) error {
	doc, err := _getQueryDoc(data)
	if err != nil {
		return err
	}
	// Find each table
	doc.Find(".t-p0").Each(func(_ int, list *goquery.Selection) {
		list.Find("li").Each(func(_ int, item *goquery.Selection) {
			item.Find("a").Each(func(_ int, link *goquery.Selection) {
				attr, exists := link.Attr("href")
				if !exists {
					return
				}
				attr = strings.Replace(attr, "/1/", "", -1)
				*target = append(*target, Category{
					URL:  attr,
					Name: link.Text(),
				})
			})
		})
	})
	return nil
}

func respToPublicationList(data []byte, target *[]string) error {
	doc, err := _getQueryDoc(data)
	if err != nil {
		return err
	}
	// Find each table

	doc.Find(".results-list__item section .button").Each(func(index int, item *goquery.Selection) {
		attr, exists := item.Attr("href")
		if !exists {
			return
		}
		*target = append(*target, attr)
	})

	return nil
}

func nameForCategoryPubsFile(catName string) string {
	catName = strings.Replace(catName, " ", "_", -1)
	catName = strings.Replace(catName, "(", "_", -1)
	catName = strings.Replace(catName, ")", "_", -1)
	catName = strings.Replace(catName, "+", "_", -1)
	catName = strings.Replace(catName, ".", "_", -1)
	catName = strings.Replace(catName, "%", "_", -1)
	return catName
}

func getPublicationUrlListByCategory(category Category, cachePolicy *CachePolicy) []string {
	publicationList := make([]string, 0)
	baseDir := filepath.Join(cachePolicy.OutPath, publicationListDir)
	fileName := nameForCategoryPubsFile(cleanField(category.Name)) + ".json"
	path := filepath.Join(baseDir, fileName)
	if cachePolicy.UseCacheForCategoriesPublications {
		if pyos.Path.Exist(path) {
			jsonData, err := ioutil.ReadFile(path)
			if err != nil {
				log.Print(err)
			}
			err = json.Unmarshal(jsonData, &publicationList)
			if err != nil {
				log.Print(err)
			}
		}
	}
	if len(publicationList) == 0 {
		for pageNumber := 1; true; pageNumber++ {
			url := defaultBaseUrl + category.URL + "/" + strconv.Itoa(pageNumber)
			resp, status, err := httpGet(url)
			if err != nil || status != 200 {
				break
			}
			err = respToPublicationList(resp, &publicationList)
			if err != nil {
				log.Printf("Error getPublicationUrlListByCategory: %s number %d : %v", category.URL, pageNumber, err)
			}
		}
		if cachePolicy.UseCacheForCategoriesPublications {
			jsonList, err := json.MarshalIndent(publicationList, "", " ")
			if err != nil {
				log.Print(err)
			}
			if !pyos.Path.Exist(baseDir) {
				err := os.MkdirAll(baseDir, 0777)
				if err != nil {
					log.Print(err)
				}
			}
			if pyos.Path.Exist(baseDir) {
				err = ioutil.WriteFile(path, jsonList, 0777)
				if err != nil {
					log.Print(err)
				}
			}
		}
	}
	return publicationList
}

func getRealCachePath(cachePath string) string {
	if cachePath != "" {
		return cachePath
	}
	wd, err := os.Getwd()
	if err != nil {
		return "./"
	}
	return wd
}

func getCategoryList(startPage Page, stopPage Page, cachePolicy *CachePolicy) ([]Category, error) {
	var catList []Category
	if cachePolicy != nil && cachePolicy.UseCacheForCategories {
		cachePolicy.OutPath = getRealCachePath(cachePolicy.OutPath)
		err := os.MkdirAll(cachePolicy.OutPath, 0777)
		if err != nil {
			return nil, err
		}
		if !pyos.Path.IsDir(cachePolicy.OutPath) {
			return nil, fmt.Errorf("the expected cache dir must be an directory")
		}
		cacheFile := filepath.Join(cachePolicy.OutPath, categoriesCacheFile)
		if pyos.Path.IsFile(cacheFile) {
			content, err := ioutil.ReadFile(cacheFile)
			if err == nil {
				_ = json.Unmarshal(content, &catList)
			}
		}
	}
	if catList == nil || len(catList) == 0 {
		var err error
		catList, err = scrapCategoryList(startPage, stopPage)
		if err != nil {
			return nil, err
		}
		if cachePolicy != nil && cachePolicy.UseCacheForCategories {
			cacheFile := filepath.Join(cachePolicy.OutPath, categoriesCacheFile)
			err = saveCategoryList(&catList, cacheFile)
			if err != nil {
				return nil, err
			}
		}
	}
	return catList, nil
}

func scrapCategoryList(startPage Page, stopPage Page) ([]Category, error) {
	if stopPage.Number == -1 {
		stopPage.Number = math.MaxInt32
	}
	categories := make([]Category, 0)
	for word := startPage.Word; word <= stopPage.Word; word++ {
		for number := startPage.Number; number < stopPage.Number; number++ {
			resp, _, err := httpGet(defaultBaseUrl + "/categorias/" + string(word) + "/" + strconv.Itoa(number))
			if err != nil {
				break
			}
			prevLen := len(categories)
			err = respToCategoriesList(resp, &categories)
			if err != nil {
				return nil, err
			}
			if prevLen == len(categories) {
				break
			}
		}
	}
	return categories, nil
}

func saveCategoryList(catList *[]Category, outputFile string) error {
	jsonCatList, err := json.MarshalIndent(catList, "", " ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(outputFile, jsonCatList, 0777)
	if err != nil {
		return err
	}
	return nil
}

func getAddress(doc *goquery.Document) Address {
	out := Address{}
	doc.Find("address").Each(func(_ int, address *goquery.Selection) {
		sp := strings.Split(address.Text(), ",")
		out.City = strings.Trim(sp[len(sp)-1], " \n")
		address.Find("meta").Each(func(_ int, meta *goquery.Selection) {
			itemProp, exist := meta.Attr("itemprop")
			if !exist {
				return
			}
			val, exist := meta.Attr("content")
			switch itemProp {
			case "streetAddress":
				out.StreetAddress = val
			case "addressLocality":
				out.Locality = val
			case "zip":
				out.Zip = val
			}

		})
	})
	return out
}

func cleanField(phone string) string {
	phone = strings.Trim(phone, " ")
	phone = strings.Replace(phone, " ", "", -1)
	phone = strings.Replace(phone, "-", "", -1)
	phone = strings.Replace(phone, ".", "", -1)
	phone = strings.Replace(phone, "\t", " ", -1)
	phone = strings.Replace(phone, "\r", " ", -1)
	phone = strings.Replace(phone, "\n", " ", -1)
	phone = strings.Trim(phone, " ")
	return phone
}

type Contacts struct {
	Phones []Phone
	Emails []string
	Webs   []string
}

func getContacts(doc *goquery.Document) Contacts {
	emails := make([]string, 0)
	webs := make([]string, 0)
	var phones []Phone
	mainPhone := ""
	whatsappNumber := ""
	doc.Find(".fl.icon-phone.mr10 span").Each(func(_ int, span *goquery.Selection) {
		mainPhone = cleanField(span.Text())

	})
	doc.Find(".fl.icon-whatsup span").Each(func(_ int, span *goquery.Selection) {
		whatsappNumber = cleanField(span.Text())
	})
	doc.Find(".info.mb10  ").Each(func(_ int, ul *goquery.Selection) {
		ul.Find(".dt.w100").Each(func(_ int, li *goquery.Selection) {
			li.Find(".title.dtc.w30").Each(func(_ int, span *goquery.Selection) {

				switch span.Text() {
				case "Fax":
					fallthrough
				case "Tel.":
					fallthrough
				case "Cel.":
					li.Find(".dtc.w70").Each(func(_ int, spanText *goquery.Selection) {
						phoneType := Mobil
						if span.Text() == "Tel." {
							phoneType = Fixed
						} else if span.Text() == "Fax" {
							phoneType = Fax
						}
						phone := cleanField(spanText.Text())
						phoneObj := Phone{
							PhoneType:    phoneType,
							Number:       phone,
							IsMain:       phone == mainPhone,
							HaveWhatsapp: phone == whatsappNumber,
						}
						phones = append(phones, phoneObj)
					})
				case "Web":
					li.Find("a").Each(func(_ int, a *goquery.Selection) {
						webs = append(webs, a.Text())
					})
				case "Email":
					li.Find("a").Each(func(_ int, a *goquery.Selection) {
						emails = append(emails, a.Text())
					})
				}
			})
		})
	})
	return Contacts{
		Phones: phones,
		Emails: emails,
		Webs:   webs,
	}
}

func getInformationForPublication(doc *goquery.Document) string {
	out := ""
	doc.Find(".information.mb30 p").Each(func(_ int, p *goquery.Selection) {
		if len(p.Text()) < 3 {
			return
		}
		if p.Contents().Nodes == nil {
			return
		}
		for _, node := range p.Contents().Nodes {
			if node.Data == "br" {
				out += "\n\r"
				continue
			}
			out += node.Data
		}
	})
	return out
}

func getCoordinatesForPublication(doc *goquery.Document) Coordinates {
	out := Coordinates{}
	doc.Find(".profile-gold.cf.mb100").Each(func(_ int, article *goquery.Selection) {
		location, exist := article.Attr("data-location")
		if !exist {
			return
		}
		location = strings.Replace(location, "&quot;", "\"", -1)
		jsonMap := make(map[string]float64, 0)
		err := json.Unmarshal([]byte(location), &jsonMap)
		if err != nil {
			return
		}
		out = Coordinates{
			Latitude:  jsonMap["lat"],
			Longitude: jsonMap["lng"],
		}
	})
	return out
}

func getNameForPublication(doc *goquery.Document) string {
	name := ""
	doc.Find(".bold.fl").Each(func(_ int, h1 *goquery.Selection) {
		name = h1.Text()
	})
	return name
}

func toDayOfWeek(day string) DayOfWeek {
	switch day {
	case "lunes":
		return Monday
	case "martes":
		return Tuesday
	case "miércoles":
		return Wednesday
	case "jueves":
		return Thursday
	case "viernes":
		return Friday
	case "sábado":
		return Saturday
	case "domingo":
		return Sunday
	default:
		return UnknownDay
	}
}

func getTimetableForPublication(doc *goquery.Document) Timetable {
	out := Timetable{}
	doc.Find(".opening-hours.mb30 table tr").Each(func(_ int, tr *goquery.Selection) {
		dayOfWeek := UnknownDay
		hourRange := HourRange{}
		tr.Find("td").Each(func(_ int, td *goquery.Selection) {
			_, exists := td.Attr("itemprop")
			if exists {
				dayOfWeek = toDayOfWeek(td.Text())
				if dayOfWeek == UnknownDay {
					log.Printf("unknown prop of prop: %s", dayOfWeek)
				}
			} else {
				tr.Find("span").Each(func(_ int, span *goquery.Selection) {
					type_, exists := span.Attr("itemprop")
					if !exists {
						return
					}
					if type_ == "opens" {
						hourRange.Start = span.Text()
					} else if type_ == "closes" {
						hourRange.End = span.Text()
					} else {
						log.Printf("invalid open/close range %s", type_)
					}
					_, ok := out[dayOfWeek]
					if !ok {
						out[dayOfWeek] = make([]HourRange, 0, 1)
					}
				})
			}

		})
		out[dayOfWeek] = append(out[dayOfWeek], hourRange)
	})
	return out
}

func getCategoriesForPublication(doc *goquery.Document) []string {
	categories := make([]string, 0)
	doc.Find(".category.kw.mb30").Each(func(_ int, section *goquery.Selection) {
		section.Find("span").Each(func(_ int, span *goquery.Selection) {
			categories = append(categories, span.Text())
		})
	})
	return categories
}

func getPublicationByUrl(url string) (*Publication, error) {
	url = defaultBaseUrl + url
	resp, status, err := httpGet(url)
	if err != nil || status != 200 {
		return nil, err
	}
	doc, err := _getQueryDoc(resp)
	if err != nil {
		return nil, err
	}

	contacts := getContacts(doc)
	p := Publication{
		Name:           getNameForPublication(doc),
		Phones:         contacts.Phones,
		Mails:          contacts.Emails,
		Address:        getAddress(doc),
		Coordinates:    getCoordinatesForPublication(doc),
		Webs:           contacts.Webs,
		Categories:     getCategoriesForPublication(doc),
		Timetable:      getTimetableForPublication(doc),
		PublicationUrl: url,
		Information:    getInformationForPublication(doc),
	}

	return &p, nil
}

func (*YellowPagesUruguayScraper) Scrap(startPage Page, stopPage Page, cachePolicy *CachePolicy) error {
	allCatPubList := make([]Publication, 0)
	catList, err := getCategoryList(startPage, stopPage, cachePolicy)
	if err != nil {
		return err
	}
	visited := map[string]bool{}
	for _, category := range catList {
		log.Printf("Category: %s", category.CleanName())
		catPubList := make([]Publication, 0)
		path := ""
		existCatPubListInFile := false
		if cachePolicy.UseCacheForPublications {
			baseDir := filepath.Join(cachePolicy.OutPath, publicationDir)
			fileName := nameForCategoryPubsFile(cleanField(category.Name))
			path = filepath.Join(baseDir, fileName)
			existCatPubListInFile = pyos.Path.Exist(path + ".json")
		}
		if existCatPubListInFile {
			content, err := ioutil.ReadFile(path + ".json")
			if err == nil {
				_ = json.Unmarshal(content, &catPubList)
			}
		} else {
			pubList := getPublicationUrlListByCategory(category, cachePolicy)
			for _, pubUrl := range pubList {
				_, ok := visited[pubUrl]
				if ok {
					continue
				}
				visited[pubUrl] = true
				pub, err := getPublicationByUrl(pubUrl)
				if err != nil {
					log.Printf("Error getting the publication: %v", err)
					continue
				}
				catPubList = append(catPubList, *pub)
			}
		}
		if len(catPubList) > 0 && cachePolicy.UseCacheForPublications {
			err := saveCategoryPublicationsInJson(catPubList, path)
			if err != nil {
				log.Print(err)
			}
		}
		allCatPubList = append(allCatPubList, catPubList...)
	}
	csvPath := filepath.Join(cachePolicy.OutPath, "out.csv")
	return saveCategoryPublicationsInCSV(allCatPubList, csvPath)
}

func phoneListToString(phones []Phone) string {
	out := ""
	for i, phone := range phones {
		if phone.IsMain {
			out += "M"
		}
		if phone.HaveWhatsapp {
			out += "W"
		}
		out += "( " + phone.Number + " )"
		if i != len(phones)-1 {
			out += " | "
		}
	}
	return out
}

func timeTableToStr(t Timetable) string {
	out := ""
	for week, ranges := range t {
		out += "[ " + string(week)
		for i, hourRange := range ranges {
			out += hourRange.Start + " -> " + hourRange.End
			if i != len(ranges)-1 {
				out += " | "
			}
		}
		out += " ]"
	}
	return out
}

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ = transform.String(t, s)
	return s
}

func saveCategoryPublicationsInCSV(catPubList []Publication, outPath string) error {
	out := [][]string{
		{
			"name",
			"categories",
			"information",
			"url",
			"phones",
			"mails",
			"address_type",
			"street",
			"number",
			"locality",
			"zip",
			"city",
			"coordinates",
			"webs",
			"timetable",
		},
	}
	for _, p := range catPubList {
		out = append(out, []string{
			removeAccents(p.Name),
			removeAccents(strings.Join(p.Categories, " | ")),
			removeAccents(p.Information),
			p.PublicationUrl,
			phoneListToString(p.Phones),
			strings.Join(p.Mails, " | "),
			string(p.Address.AddressType),
			removeAccents(p.Address.StreetAddress),
			p.Address.Number,
			removeAccents(p.Address.Locality),
			p.Address.Zip,
			removeAccents(p.Address.City),
			p.Coordinates.String(),
			strings.Join(p.Webs, " | "),
			timeTableToStr(p.Timetable),
		})
	}
	csvFile, err := os.Create(outPath + ".csv")
	if err != nil {
		return err
	}
	w := csv.NewWriter(csvFile)
	err = w.WriteAll(out)
	if err := w.Error(); err != nil {
		return err
	}
	return nil
}

func saveCategoryPublicationsInJson(catPubList []Publication, outPath string) error {
	jsonStr, err := json.MarshalIndent(catPubList, "", " ")
	if err != nil {
		return err
	}
	base := filepath.Dir(outPath)
	err = os.MkdirAll(base, 0777)
	if err != nil {
		log.Print(err)
	}
	err = ioutil.WriteFile(outPath+".json", jsonStr, 0777)
	return err
}
