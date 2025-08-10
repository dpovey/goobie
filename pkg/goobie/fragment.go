package goobie

import (
	"strconv"
	"strings"
)

type Fragment struct {
	SQL  string
	Args []any
}

func countPlaceholders(sql string) int {
	count := 0
	inQuote := false
	escaped := false

	for _, char := range sql {
		switch {
		case char == '\\' && !escaped:
			escaped = true
		case char == '\'' && !escaped:
			inQuote = !inQuote
		case char == '?' && !inQuote && !escaped:
			count++
		default:
			escaped = false
		}
	}

	return count
}

func splitOnPlaceholders(s string) []string {
	var parts []string
	var current strings.Builder
	inString := false
	escaped := false

	for _, char := range s {
		switch {
		case char == '\'':
			if !escaped {
				inString = !inString
			}
			current.WriteRune(char)
			escaped = false
		case char == '\\' && inString:
			current.WriteRune(char)
			escaped = !escaped
		case char == '?' && !inString:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(char)
			escaped = false
		}
	}

	parts = append(parts, current.String())
	return parts
}

func (fr Fragment) ToPostgres() Fragment {
	// Handle case where we have placeholders but no arguments (batch operations)
	if len(fr.Args) == 0 {
		// Convert ? placeholders to PostgreSQL $1, $2, etc. format
		placeholderCount := countPlaceholders(fr.SQL)
		if placeholderCount == 0 {
			return fr
		}
		
		newSQL := fr.SQL
		for i := 1; i <= placeholderCount; i++ {
			newSQL = strings.Replace(newSQL, "?", "$"+strconv.Itoa(i), 1)
		}
		
		return Fragment{SQL: newSQL, Args: fr.Args}
	}

	argMap := make(map[any]int)
	var uniqueArgs []any
	var newSQL strings.Builder
	argCount := 1

	parts := splitOnPlaceholders(fr.SQL)
	for i, part := range parts {
		newSQL.WriteString(part)
		if i < len(fr.Args) {
			arg := fr.Args[i]
			if placeholderNum, exists := argMap[arg]; exists {
				newSQL.WriteString("$" + strconv.Itoa(placeholderNum))
			} else {
				argMap[arg] = argCount
				uniqueArgs = append(uniqueArgs, arg)
				newSQL.WriteString("$" + strconv.Itoa(argCount))
				argCount++
			}
		}
	}

	return Fragment{
		SQL:  newSQL.String(),
		Args: uniqueArgs,
	}
}

func FrConst(sql string) Fragment {
	nrPlaceholders := countPlaceholders(sql)
	if nrPlaceholders != 0 {
		panic("Unexpected placeholders in: " + sql)
	}
	return Fr0Unchecked(sql)
}
func Fr0Unchecked(sql string) Fragment {
	return Fragment{
		SQL:  sql,
		Args: nil,
	}
}
func SQL(sql string, params ...any) Fragment {
	return Fr(sql, params...)
}

func Fr(sql string, args ...any) Fragment {
	return Fragment{
		SQL:  sql,
		Args: args,
	}
}

func (fr Fragment) Concat(b Fragment) Fragment {
	newArgs := make([]any, 0, len(fr.Args)+len(b.Args))
	newArgs = append(newArgs, fr.Args...)
	newArgs = append(newArgs, b.Args...)
	return Fragment{
		SQL:  fr.SQL + "\n" + b.SQL,
		Args: newArgs,
	}
}
func Combine(fragments []Fragment) Fragment {
	if len(fragments) == 0 {
		return Fragment{}
	}

	totalLength := 0
	totalArgs := 0
	for _, f := range fragments {
		totalLength += len(f.SQL)
		totalArgs += len(f.Args)
	}

	combinedSQL := strings.Builder{}
	combinedSQL.Grow(totalLength + len(fragments) - 1) // Extra space for newlines

	combinedArgs := make([]any, 0, totalArgs)

	for i, f := range fragments {
		if i > 0 {
			combinedSQL.WriteByte('\n')
		}
		combinedSQL.WriteString(f.SQL)
		combinedArgs = append(combinedArgs, f.Args...)
	}

	return Fragment{
		SQL:  combinedSQL.String(),
		Args: combinedArgs,
	}
}
