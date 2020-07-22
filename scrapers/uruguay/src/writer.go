package scraper

type PublicationWriter interface {
	Write(publication Publication) error
}
