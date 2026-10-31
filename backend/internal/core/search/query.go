// 本文件生成参数化搜索条件、分页排序，并对返回标题与摘要进行安全高亮。
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

// defaultSearchLimit 用于非正数分页大小，负偏移另行归零。
const defaultSearchLimit = 50

// maxFullTextPhraseRunes 限制进入全文短语分支的字符数，超长词仍保留字面子串条件。
const maxFullTextPhraseRunes = 128

// Search 应用搜索词和可选页面集合，先查总数再查当前页，两个读取不创建共同事务。
// nil 页面集合表示不额外限制，非 nil 空集合直接返回空结果；空查询且无页面 ID 也为空。
// 成功结果标题和摘要先转义再插入 b 标签，标签统计保持空列表交给上层填充。
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
		// 排序分数包含标题字面命中二十分、标题段字面命中五分及自然语言全文分数。
		// 计分使用完整显示查询串，不逐个重用 WHERE 中的分词条件。
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
			// 将数据库相关度映射为接口分数，排序仍使用映射前 SQL relevance。
			item.Rank = (1 + item.Rank) / (2 + item.Rank)
			result.Items = append(result.Items, item)
		}
		return rows.Err()
	})
	return result, err
}

// SearchPageIDs 复用同一过滤条件返回全部匹配页面 ID，供标签统计范围计算。
// 不分页、不计算相关度，按标题及页面 ID 排序。
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

// buildSearchWhereClause 按空白分词，各词条件和页面集合条件用 AND 连接。
// 仅允许表内列名映射参与 SQL 拼接，用户词和页面 ID 全部通过参数绑定。
// 每词移除一个尾部星号和两端引号，已知列前缀指定范围，未知前缀作为文字保留。
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
		// 未指定列且为规定长度的纯字母数字词时，全文短语命中与字面子串命中任选其一。
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

// isSearchWord 只允许 Unicode 字母或数字，空串、空格及标点均返回 false。
func isSearchWord(value string) bool {
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return false
		}
	}
	return value != ""
}

// escapeSearchLike 使用等号作为 SQL LIKE 转义符，使百分号和下划线按文字匹配。
func escapeSearchLike(value string) string {
	return strings.NewReplacer("=", "==", "%", "=%", "_", "=_").Replace(value)
}

// highlightLiteral 对完整查询串进行忽略大小写的字面匹配，正则特殊字符均先转义。
// 匹配和未匹配片段分别做 HTML 转义，只有函数写入的 b 标签作为标记保留。
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

// searchExcerpt 短文本完整保留，长文本默认从头截取公共摘要长度。
// 找到非开头命中时尝试从命中前四十字符开始，两侧被截断时各加省略号。
// 查找使用小写字符串索引，窗口截取使用原内容的字符切片。
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

// searchColumns 是可接受列过滤前缀到真实 SQL 列名的唯一映射，输入前缀忽略大小写。
var searchColumns = map[string]string{
	"title": "title", "headings": "headings", "content": "content",
	"path": "path", "filepath": "filepath", "pageid": "pageID", "kind": "kind",
}

// displaySearchQuery 去掉已知列前缀、每词一个尾部星号和两端双引号，再用空格连接。
// 该结果用于计分、高亮和摘要定位，不是对 WHERE 查询语法的完整重建。
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
