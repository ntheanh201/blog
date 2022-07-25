package main

import (
	"encoding/json"
	"fmt"
	"github.com/kjk/notionapi"
)

type CollectionView struct {
	Id       string
	Type     string
	Name     string
	PageSort []string
}

type Property struct {
	//Type [][]string `json:"`gQ~"`
}

type PageSort struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func CollectionViewToPages(d *notionapi.CachingClient) []string {
	s := []string{"68f077a6dfb346358f219875e80ea72c"}
	//s1 := append(s, "cf7b1a3766ea499a90e568028152b10a")
	out, _ := d.Client.GetBlockRecords(s)
	var pages []string
	var pagesTmp []PageSort
	//pages = append(pages, "38e8c2a0cf7146a68cc24f847f94d800")
	for _, b := range out {
		for c, e := range b.ViewIDs {
			fmt.Println("c: ", c)
			//if c == 0 || c == 1 {
			//	continue //跳过不想展示的前两个组
			//}

			//fliters := make([]*notionapi.QueryFilter, 1)
			//var fliter *notionapi.QueryFilter = new(notionapi.QueryFilter)
			//fliter.Comparator = "enum_contains"
			////fliter.ID = "f19ce6f4-1431-48e0-8390-766d7beab632"
			//fliter.Type = "multi_select"
			//for i := 0; i < 1; i++ {
			//	fliters[i] = fliter
			//	//fliters[i].ID = "f19ce6f4-1431-48e0-8390-766d7beab632"
			//	//fliters[i].Type = "multi_select"
			//}
			//var user *notionapi.User = new(notionapi.User)
			//user.Locale = "vi-vn"
			//user.TimeZone = "Asia/Ho_Chi_Minh"
			//var query *notionapi.Query = new(notionapi.Query)
			//query.Filter = fliters
			////query.CalendarBy="(WIG"
			//query.FilterOperator = "and"

			req := notionapi.QueryCollectionRequest{
				Collection: struct {
					ID      string `json:"id"`
					SpaceID string `json:"spaceId"`
				}{
					ID: b.CollectionID,
				},
				CollectionView: struct {
					ID      string `json:"id"`
					SpaceID string `json:"spaceId"`
				}{
					ID: e,
				},
			}
			query := notionapi.Query{}
			coll, _ := d.Client.QueryCollection(req, &query)

			for _, g := range coll.RecordMap.Blocks {
				//var bb CollectionView
				var bb notionapi.Record
				json.Unmarshal([]byte(g.Value), &bb)
				//pages = append(pages, bb.Page_sort...)
				//pages = append(pages, g.ID)

				pagesTmp = append(pagesTmp, PageSort{ID: g.ID, Type: "page"})
			}

		}
	}
	//log.Println("pages", pages)
	return pages
}
