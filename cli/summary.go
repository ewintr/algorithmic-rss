package main

import (
	"fmt"
	"strings"

	"go-mod.ewintr.nl/algorithmic-rss/storage"
)

type Summary struct {
	TotalEntries   int64
	ByCategory     map[int64]int64
	ByRating       map[string]int64
	CategoryRating map[int64]map[string]int64
	AllRatings     []string
	CategoryNames  map[int64]string
}

func GenerateSummary(repo *storage.CliRepo) Summary {
	total, err := repo.TotalEntries()
	if err != nil {
		fmt.Printf("Warning: could not get total entries: %v\n", err)
	}
	byCategory, err := repo.EntriesByCategory()
	if err != nil {
		fmt.Printf("Warning: could not get entries by category: %v\n", err)
	}
	byRating, err := repo.RatingsByStatus()
	if err != nil {
		fmt.Printf("Warning: could not get ratings by status: %v\n", err)
	}
	categoryRating, err := repo.CategoryRatingMatrix()
	if err != nil {
		fmt.Printf("Warning: could not get category rating matrix: %v\n", err)
	}
	allRatings, err := repo.AllRatings()
	if err != nil {
		fmt.Printf("Warning: could not get all ratings: %v\n", err)
	}
	categoryNames, err := repo.CategoryNames()
	if err != nil {
		fmt.Printf("Warning: could not get category names: %v\n", err)
	}

	return Summary{
		TotalEntries:   total,
		ByCategory:     byCategory,
		ByRating:       byRating,
		CategoryRating: categoryRating,
		AllRatings:     allRatings,
		CategoryNames:  categoryNames,
	}
}

func PrintMatrix(s Summary) {
	if len(s.AllRatings) == 0 {
		fmt.Println("(no data)")
		return
	}

	ratings := []string{"not_opened", "only_comments", "not_finished", "finished"}

	fmt.Println("Database Summary")
	fmt.Println("================")
	fmt.Printf("Total: %d entries\n\n", s.TotalEntries)
	fmt.Println("Category × Rating Matrix:")

	if len(s.CategoryRating) == 0 {
		fmt.Println("(no data)")
		return
	}

	colWidth := 15

	header := fmt.Sprintf("| %-24s ", "")
	for _, rating := range ratings {
		header += fmt.Sprintf("| %-*s ", colWidth-1, rating)
	}
	header += "|"
	fmt.Println(header)
	fmt.Println(strings.Repeat("+", len(header)))

	for catID, counts := range s.CategoryRating {
		name := fmt.Sprintf("%d", catID)
		if title, ok := s.CategoryNames[catID]; ok {
			name = fmt.Sprintf("%d (%s)", catID, title)
		}
		row := fmt.Sprintf("| %-24s ", name)
		for _, rating := range ratings {
			count := counts[rating]
			row += fmt.Sprintf("| %-*d ", colWidth-1, count)
		}
		row += "|"
		fmt.Println(row)
	}

	fmt.Println(strings.Repeat("+", len(header)))
}
