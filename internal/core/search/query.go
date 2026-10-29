package search

import (
	"database/sql"
	"fmt"
	"log/slog"

	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/excerpt"

	"github.com/NianBroken/LeafDoc/internal/core/shared/htmlutil"

	_ "modernc.org/sqlite"
)

func (s *SQLiteIndex) Search(query string, pageIDs []string, offset, limit int) (*SearchResult, error) {
	query = strings.TrimSpace(query)
	// ftsQuery, not the raw query, decides whether there's anything to MATCH
	// on: a query built entirely from characters outside the tokenizer's
	// word alphabet (e.g. "," or "'" alone) sanitizes to "" even though the
	// raw query wasn't empty, and `pages MATCH ''` is itself an FTS5 syntax
	// error — the same crash class #1577 reported, for the simplest
	// possible input.
	ftsQuery := buildFuzzyQuery(query)

	if len(pageIDs) == 0 && pageIDs != nil {
		return &SearchResult{
			Count:     0,
			Items:     []SearchResultItem{},
			Offset:    offset,
			Limit:     limit,
			TagFacets: []SearchTagFacet{},
		}, nil
	}

	if ftsQuery == "" && len(pageIDs) == 0 {
		return &SearchResult{
			Count:     0,
			Items:     []SearchResultItem{},
			Offset:    offset,
			Limit:     limit,
			TagFacets: []SearchTagFacet{},
		}, nil
	}

	sr := &SearchResult{TagFacets: []SearchTagFacet{}}

	err := s.withDBRead(func(db *sql.DB) error {
		var total int
		whereClause, whereArgs := buildSearchWhereClause(ftsQuery, pageIDs)

		countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM pages WHERE %s;`, whereClause)
		if err := db.QueryRow(countQuery, whereArgs...).Scan(&total); err != nil {
			return err
		}
		sr.Count = total

		searchQuery := fmt.Sprintf(`
		SELECT
			pageID,
			path,
			kind,
			%s AS highlighted_title,
			%s AS excerpt,
			content,
			%s AS bm25_score
		FROM pages
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?;
	`,
			searchTitleExpr(ftsQuery != ""),
			searchExcerptExpr(ftsQuery != ""),
			searchRankExpr(ftsQuery != ""),
			whereClause,
			searchOrderByExpr(ftsQuery != ""),
		)

		queryArgs := append(append([]interface{}{}, whereArgs...), limit, offset)
		rows, err := db.Query(searchQuery, queryArgs...)
		if err != nil {
			return err
		}
		defer func() {
			if err := rows.Close(); err != nil {
				slog.Default().Error("could not close rows", "error", err)
			}
		}()

		var results []SearchResultItem
		for rows.Next() {
			var r SearchResultItem
			var bm25Score float64
			var content string

			if err := rows.Scan(&r.PageID, &r.Path, &r.Kind, &r.Title, &r.Excerpt, &content, &bm25Score); err != nil {
				return err
			}
			r.Title = sanitizeSearchTitle(r.Title)
			if strings.TrimSpace(r.Excerpt) == "" {
				r.Excerpt = excerpt.FromBody(content)
			}

			if ftsQuery == "" {
				r.Rank = 1
			} else {
				// Convert bm25 score to a rank (lower score = higher rank)
				if bm25Score < 0 {
					bm25Score = 0
				}
				r.Rank = 1.0 / (1.0 + bm25Score)
			}

			results = append(results, r)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		sr.Items = results
		sr.Offset = offset
		sr.Limit = limit
		return nil
	})

	return sr, err
}

func (s *SQLiteIndex) SearchPageIDs(query string, pageIDs []string) ([]string, error) {
	query = strings.TrimSpace(query)
	ftsQuery := buildFuzzyQuery(query)

	if len(pageIDs) == 0 && pageIDs != nil {
		return []string{}, nil
	}

	if ftsQuery == "" && len(pageIDs) == 0 {
		return []string{}, nil
	}

	var result []string

	err := s.withDBRead(func(db *sql.DB) error {
		whereClause, whereArgs := buildSearchWhereClause(ftsQuery, pageIDs)
		searchQuery := fmt.Sprintf(`
		SELECT pageID, %s AS bm25_score
		FROM pages
		WHERE %s
		ORDER BY %s;
	`, searchRankExpr(ftsQuery != ""), whereClause, searchOrderByExpr(ftsQuery != ""))

		rows, err := db.Query(searchQuery, whereArgs...)
		if err != nil {
			return err
		}
		defer func() {
			if err := rows.Close(); err != nil {
				slog.Default().Error("could not close rows", "error", err)
			}
		}()

		for rows.Next() {
			var pageID string
			var bm25Score float64
			if err := rows.Scan(&pageID, &bm25Score); err != nil {
				return err
			}
			result = append(result, pageID)
		}

		return rows.Err()
	})

	return result, err
}

func buildSearchWhereClause(ftsQuery string, pageIDs []string) (string, []interface{}) {
	clauses := make([]string, 0, 2)
	args := make([]interface{}, 0, 1+len(pageIDs))

	if ftsQuery != "" {
		clauses = append(clauses, "pages MATCH ?")
		args = append(args, ftsQuery)
	}

	if len(pageIDs) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pageIDs)), ",")
		clauses = append(clauses, fmt.Sprintf("pageID IN (%s)", placeholders))
		for _, pageID := range pageIDs {
			args = append(args, pageID)
		}
	}

	return strings.Join(clauses, " AND "), args
}

func searchTitleExpr(hasQuery bool) string {
	if hasQuery {
		return "highlight(pages, 4, char(2), char(3))"
	}
	return "title"
}

func sanitizeSearchTitle(title string) string {
	return htmlutil.EscapeTextWithAllowedMarkers(
		title,
		htmlutil.AllowedMarker{Marker: "\u0002", HTML: "<b>"},
		htmlutil.AllowedMarker{Marker: "\u0003", HTML: "</b>"},
	)
}

func searchExcerptExpr(hasQuery bool) string {
	if hasQuery {
		return "snippet(pages, 6, '<b>', '</b>', '...', 16)"
	}
	return "''"
}

func searchRankExpr(hasQuery bool) string {
	if hasQuery {
		return `bm25(pages,
				0.0,  -- path
				0.0,  -- filepath
				0.0,  -- pageID
				0.0,  -- kind
				20.0, -- title
				5.0,   -- headings
				1.0    -- content
			)`
	}
	return "0.0"
}

func searchOrderByExpr(hasQuery bool) string {
	if hasQuery {
		return "bm25_score ASC"
	}
	return "title COLLATE NOCASE ASC, path COLLATE NOCASE ASC"
}
