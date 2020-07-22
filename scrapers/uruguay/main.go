package main

import (
	scraper "github.com/Eitol/yellow_pages_scrapper/scrapers/uruguay/src"
	"log"
)

func main() {
	scp := scraper.YellowPagesUruguayScraper{
		MaxThreads: 1,
	}
	err := scp.Scrap(
		scraper.BeginPage(),
		scraper.EndPage(),
		&scraper.CachePolicy{
			UseCacheForCategories:             true,
			UseCacheForCategoriesPublications: true,
			UseCacheForPublications:           true,
			OutPath:                           "./out",
		},
	)
	log.Print("Finish")
	if err != nil {
		log.Printf("error: %v", err)
	}
}
