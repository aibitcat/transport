package elasticsearch

import (
	"encoding/json"
	"fmt"
	"github.com/luopengift/gohttp"
	"github.com/luopengift/golibs/logger"
)

type Document struct {
	Index  string                 `json:"_index"`
	Type   string                 `json:"_type"`
	Id     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

type Shards struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
}

type Hits struct {
	Total    int        `json:"total"`
	MaxScore float64    `json:"max_score"`
	Hits     []Document `json:"hits"`
}

type Scroll struct {
	ScrollId string `json:"_scroll_id"`
	Took     int    `json:"took"`
	TimeOut  bool   `json:"time_out"`
	Shards   Shards `json:"_shards"`
	Hits     Hits   `json:hits`
}

type ScrollQuery struct {
	Url       string
	Index     string
	Type      string
	Scroll    string
	QueryBody interface{}
	ScrollId  string
	Ch        chan map[string]interface{}
	IsEnd     bool
}

func NewScroll(url, _index, _type, scroll string, querybody interface{}) *ScrollQuery {
	s := &ScrollQuery{
		Url:       url,
		Index:     _index,
		Type:      _type,
		Scroll:    scroll,
		QueryBody: querybody,
		Ch:        make(chan map[string]interface{}, 1000),
		IsEnd:     false,
	}
	s.First()
	return s
}

func (self *ScrollQuery) Read(p []byte) (int, error) {
	data, ok := <-self.Ch
	if !ok {
		return 0, fmt.Errorf("scroll channel is closed.")
	}

	bp, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	n := copy(p, bp)
	return n, nil
}

func (self *ScrollQuery) First() {
	resp, err := gohttp.NewClient().Url(self.Url).Path(self.Index+"/"+self.Type+"/_search").
		Query(map[string]string{"scroll": "10m", "size": "500"}).
		//Query(map[string]string{"search_type": "scan", "scroll": "10m", "size": "50"}).
		Header("Content-Type", "application/json").Body(self.QueryBody).Get()
	self.parseResponse(resp, err)
}

func (self *ScrollQuery) parseResponse(resp *gohttp.Response, err error) {
	if err != nil {
		logger.Error("response error:%#v", err)
		return
	}
	if resp.Code() != 200 {
		logger.Error("response:%+v,error:%+v", resp.String(), err)
		return
	}
	res := resp.Bytes()
	data := Scroll{}
	err = json.Unmarshal(res, &data)
	if err != nil {
		logger.Error("%+v", err)
		return
	}
	self.ScrollId = data.ScrollId
	if len(data.Hits.Hits) == 0 {
		self.IsEnd = true
		return
	}
	for _, v := range data.Hits.Hits {
		self.Ch <- v.Source
	}
}

func (self *ScrollQuery) Next() error {
	client := gohttp.NewClient().Url(self.Url).Path("/_search/scroll").Header("Content-Type", "application/json")
	for !self.IsEnd {
		resp, err := client.Body(map[string]string{"scroll": self.Scroll, "scroll_id": self.ScrollId}).Get()
		self.parseResponse(resp, err)
	}
	self.Close()
	return nil
}

func (self *ScrollQuery) Close() error {
	close(self.Ch)
	return nil
}