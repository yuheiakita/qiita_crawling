package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const spreadsheetID = "***"

type Item struct {
	RenderedBody  string    `json:"rendered_body"`
	Body          string    `json:"body"`
	Coediting     bool      `json:"coediting"`
	CommentsCount int       `json:"comments_count"`
	CreatedAt     time.Time `json:"created_at"`
	Group         struct {
		CreatedAt   time.Time `json:"created_at"`
		Description string    `json:"description"`
		Name        string    `json:"name"`
		Private     bool      `json:"private"`
		UpdatedAt   time.Time `json:"updated_at"`
		UrlName     string    `json:"url_name"`
	} `json:"group"`
	Id             string `json:"id"`
	LikesCount     int    `json:"likes_count"`
	Private        bool   `json:"private"`
	ReactionsCount int    `json:"reactions_count"`
	StocksCount    int    `json:"stocks_count"`
	Tags           []struct {
		Name     string   `json:"name"`
		Versions []string `json:"versions"`
	} `json:"tags"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
	Url       string    `json:"url"`
	User      struct {
		Description       string `json:"description"`
		FacebookId        string `json:"facebook_id"`
		FolloweesCount    int    `json:"followees_count"`
		FollowersCount    int    `json:"followers_count"`
		GithubLoginName   string `json:"github_login_name"`
		Id                string `json:"id"`
		ItemsCount        int    `json:"items_count"`
		LinkedinId        string `json:"linkedin_id"`
		Location          string `json:"location"`
		Name              string `json:"name"`
		Organization      string `json:"organization"`
		PermanentId       int    `json:"permanent_id"`
		ProfileImageUrl   string `json:"profile_image_url"`
		TeamOnly          bool   `json:"team_only"`
		TwitterScreenName string `json:"twitter_screen_name"`
		WebsiteUrl        string `json:"website_url"`
	} `json:"user"`
	PageViewsCount int `json:"page_views_count"`
	TeamMembership struct {
		Name string `json:"name"`
	} `json:"team_membership"`
}

type SheetClient struct {
	srv           *sheets.Service
	spreadsheetID string
}

func NewSheetClient(ctx context.Context, spreadsheetID string) (*SheetClient, error) {
	b, err := ioutil.ReadFile("secret.json")
	if err != nil {
		return nil, err
	}
	jwt, err := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return nil, err
	}
	srv, err := sheets.New(jwt.Client(ctx))
	if err != nil {
		return nil, err
	}
	return &SheetClient{
		srv:           srv,
		spreadsheetID: spreadsheetID,
	}, nil
}

func main() {
	amazonLinkList := map[string]int{}
	var IDList []string
	for i := 0; i < 100; i++ {
		bin := getBinForURL(i)
		var items []Item
		if err := json.Unmarshal(bin, &items); err != nil {
			panic(err)
		}
		for _, v := range items {
			if slices.Contains(IDList, v.Id) {
				continue
			}
			IDList = append(IDList, v.Id)
			doc, _ := goquery.NewDocumentFromReader(strings.NewReader(v.RenderedBody))
			doc.Find("body a").Each(func(i int, s *goquery.Selection) {
				href, _ := s.Attr("href")
				link, err := url.QueryUnescape(href)
				if err != nil {
					log.Fatalf("http get error: %v", err)
				}
				if isAmazonURL(link) {
					if _, ok := amazonLinkList[link]; ok {
						count := amazonLinkList[link]
						amazonLinkList[link] = count + 1
					} else {
						amazonLinkList[link] = 1
					}
				}
			})
		}
	}

	InsertSpreadSeatAmazonLinks(amazonLinkList)
	println("finish")
}

func getBinForURL(pageCount int) []byte {
	client := http.Client{}
	qiitaURL := "https://qiita.com/api/v2/items"
	u, err := url.Parse(qiitaURL)
	if err != nil {
		log.Fatalf("url perse error: %v", err)
	}
	u.RawQuery = fmt.Sprintf("page=%s&per_page=100", strconv.Itoa(pageCount+1))
	println(u.String())
	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	req.Header.Add("Authorization", "Bearer ***")
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("http get error: %v", err)
	}
	if res == nil {
		log.Fatalf("http response is null")
	}
	bin, _ := io.ReadAll(res.Body)
	return bin
}

func isAmazonURL(url string) bool {
	if strings.Contains(url, "https://www.amazon.com") || strings.Contains(url, "https://www.amazon.co.jp") {
		return true
	}
	return false
}

func InsertSpreadSeatAmazonLinks(amazonLinkList map[string]int) {
	ctx := context.Background()
	client, err := NewSheetClient(ctx, spreadsheetID)
	values := [][]interface{}{
		{
			"リンク",
			"カウント",
		},
	}
	if err != nil {
		fmt.Printf("読み込み出来ませんでした: %v", err)
	}
	for k, v := range amazonLinkList {
		values = append(values, []interface{}{k, v})
	}
	if err := client.Update("A1", values); err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
}

func (s *SheetClient) Update(range_ string, values [][]interface{}) error {
	_, err := s.srv.Spreadsheets.Values.Update(s.spreadsheetID, range_, &sheets.ValueRange{
		Values: values,
	}).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}
	return nil
}
