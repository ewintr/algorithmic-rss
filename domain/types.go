package domain

type Category struct {
	ID    int64
	Title string
}

type Feed struct {
	ID         int64
	CategoryID int64
	FeedURL    string
	SiteURL    string
	Title      string
}

type Entry struct {
	ID      int64
	FeedID  int64
	Title   string
	URL     string
	Content string
}
