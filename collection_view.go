package main

import (
	"encoding/json"
	"github.com/kjk/notionapi"
)

func CollectionViewToPages(d *notionapi.CachingClient) []string {
	s := []string{"68f077a6dfb346358f219875e80ea72c"}
	out, _ := d.Client.GetBlockRecords(s)
	var pages []string
	for _, b := range out {
		for _, e := range b.ViewIDs {
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
				var bb notionapi.Record
				err := json.Unmarshal(g.Value, &bb)
				if err != nil {
					return nil
				}
				pages = append(pages, g.ID)
			}
		}
	}
	return pages
}
