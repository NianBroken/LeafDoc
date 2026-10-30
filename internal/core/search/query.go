package search

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/excerpt"
)

const defaultSearchLimit = 50

// InnoDB supports at most 128 tokens per phrase. A rune bound also covers
// ngram_token_size=1; longer literals use the complete substring predicate.
const maxFullTextPhraseRunes = 128

// Search queries MySQL, applies the caller's page filter, and escapes rendered text.
// Full-text matches use ngram indexing; literal substring matching also handles
// single characters, code punctuation, and words omitted by full-text stop rules.
func (s *MySQLIndex) Search(query string, pageIDs []string, offset, limit int) (*SearchResult, error) {
	query = strings.TrimSpace(query)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	result := &SearchResult{Offset: offset, Limit: limit, Items: []SearchResultItem{}, TagFacets: []SearchTagFacet{}}
	if (pageIDs != nil && len(pageIDs) == 0) || (query == "" && len(pageIDs) == 0) {
		return result, nil
	}
	where, args := buildSearchWhereClause(query, pageIDs)
	err := s.withExecutorRead(func(db database.Executor) (resultErr error) {
		if err := db.QueryRow("SELECT COUNT(*) FROM search_documents WHERE "+where, args...).Scan(&result.Count); err != nil {
			return err
		}
		// Explicit title and heading weights keep concise document titles prominent.
		rank := "0.0"
		rankArgs := []any{}
		if query != "" {
			rank = "CASE WHEN title LIKE ? ESCAPE '=' THEN 20 ELSE 0 END + CASE WHEN headings LIKE ? ESCAPE '=' THEN 5 ELSE 0 END + MATCH(title, headings, content) AGAINST (? IN NATURAL LANGUAGE MODE)"
			literal := "%" + escapeSearchLike(displaySearchQuery(query)) + "%"
			rankArgs = append(rankArgs, literal, literal, displaySearchQuery(query))
		}
		statement := "SELECT pageID, path, kind, title, content, (" + rank + ") AS relevance FROM search_documents WHERE " + where + " ORDER BY relevance DESC, title ASC, pageID ASC LIMIT ? OFFSET ?"
		queryArgs := append(append([]any{}, rankArgs...), args...)
		queryArgs = append(queryArgs, limit, offset)
		rows, err := db.Query(statement, queryArgs...)
		if err != nil {
			return err
		}
		defer database.CloseResource(rows, &resultErr)
		for rows.Next() {
			var item SearchResultItem
			var content string
			if err := rows.Scan(&item.PageID, &item.Path, &item.Kind, &item.Title, &content, &item.Rank); err != nil {
				return err
			}
			item.Title = highlightLiteral(item.Title, displaySearchQuery(query))
			item.Excerpt = highlightLiteral(searchExcerpt(content, displaySearchQuery(query)), displaySearchQuery(query))
			item.Rank = (1 + item.Rank) / (2 + item.Rank)
			result.Items = append(result.Items, item)
		}
		return rows.Err()
	})
	return result, err
}

// SearchPageIDs uses the identical filter as paginated search for tag facets.
func (s *MySQLIndex) SearchPageIDs(query string, pageIDs []string) ([]string, error) {
	query = strings.TrimSpace(query)
	result := []string{}
	if (pageIDs != nil && len(pageIDs) == 0) || (query == "" && len(pageIDs) == 0) {
		return result, nil
	}
	where, args := buildSearchWhereClause(query, pageIDs)
	err := s.withExecutorRead(func(db database.Executor) (resultErr error) {
		rows, err := db.Query("SELECT pageID FROM search_documents WHERE "+where+" ORDER BY title, pageID", args...)
		if err != nil {
			return err
		}
		defer database.CloseResource(rows, &resultErr)
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			result = append(result, id)
		}
		return rows.Err()
	})
	return result, err
}

// buildSearchWhereClause binds all input, including wildcard characters, as data.
func buildSearchWhereClause(query string, pageIDs []string) (string, []any) {
	clauses := []string{}
	args := []any{}
	for _, raw := range strings.Fields(query) {
		term := strings.TrimSuffix(raw, "*")
		column := "CONCAT_WS(' ', title, headings, content)"
		scope := ""
		if key, value, ok := strings.Cut(term, ":"); ok {
			if actual, known := searchColumns[strings.ToLower(key)]; known {
				column = actual
				scope = actual
				term = value
			}
		}
		term = strings.Trim(term, `"`)
		if term == "" {
			continue
		}
		literal := "%" + escapeSearchLike(term) + "%"
		clause := column + " LIKE ? ESCAPE '='"
		termRunes := utf8.RuneCountInString(term)
		if scope == "" && termRunes >= 2 && termRunes <= maxFullTextPhraseRunes && isSearchWord(term) {
			clause = "(MATCH(title, headings, content) AGAINST (? IN BOOLEAN MODE) OR " + clause + ")"
			args = append(args, `"`+term+`"*`)
		}
		clauses = append(clauses, clause)
		args = append(args, literal)
	}
	if len(pageIDs) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(pageIDs)), ",")
		clauses = append(clauses, fmt.Sprintf("pageID IN (%s)", placeholders))
		for _, id := range pageIDs {
			args = append(args, id)
		}
	}
	if len(clauses) == 0 {
		return "1=0", args
	}
	return strings.Join(clauses, " AND "), args
}

func isSearchWord(value string) bool {
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return false
		}
	}
	return value != ""
}

func escapeSearchLike(value string) string {
	return strings.NewReplacer("=", "==", "%", "=%", "_", "=_").Replace(value)
}

// highlightLiteral escapes each text fragment before adding the allowed bold tag.
func highlightLiteral(value, query string) string {
	if query == "" {
		return html.EscapeString(value)
	}
	expression := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	indexes := expression.FindAllStringIndex(value, -1)
	var output strings.Builder
	start := 0
	for _, index := range indexes {
		output.WriteString(html.EscapeString(value[start:index[0]]))
		output.WriteString("<b>")
		output.WriteString(html.EscapeString(value[index[0]:index[1]]))
		output.WriteString("</b>")
		start = index[1]
	}
	output.WriteString(html.EscapeString(value[start:]))
	return output.String()
}

func searchExcerpt(content, query string) string {
	text := []rune(content)
	if len(text) <= excerpt.MaxRunes {
		return content
	}
	start := 0
	if query != "" {
		found := strings.Index(strings.ToLower(content), strings.ToLower(query))
		if found > 0 {
			start = utf8.RuneCountInString(content[:found]) - 40
			if start < 0 {
				start = 0
			}
		}
	}
	end := start + excerpt.MaxRunes
	if end > len(text) {
		end = len(text)
	}
	result := string(text[start:end])
	if start > 0 {
		result = "..." + result
	}
	if end < len(text) {
		result += "..."
	}
	return result
}

// searchColumns is an allow-list of SQL identifiers accepted by column filters.
var searchColumns = map[string]string{
	"title": "title", "headings": "headings", "content": "content",
	"path": "path", "filepath": "filepath", "pageid": "pageID", "kind": "kind",
}

func displaySearchQuery(query string) string {
	fields := strings.Fields(query)
	for i, field := range fields {
		if key, value, ok := strings.Cut(field, ":"); ok {
			if _, known := searchColumns[strings.ToLower(key)]; known {
				field = value
			}
		}
		fields[i] = strings.Trim(strings.TrimSuffix(field, "*"), `"`)
	}
	return strings.Join(fields, " ")
}
