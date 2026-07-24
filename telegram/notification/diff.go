package notification

import (
	"unicode"

	tele "gopkg.in/telebot.v4"
)

type wordToken struct {
	text   string
	offset int
	length int
}

func utf16RuneLen(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}

func tokenizeWithOffsets(text string) []wordToken {
	runes := []rune(text)
	var tokens []wordToken
	utf16Off := 0
	i := 0
	for i < len(runes) {
		if unicode.IsSpace(runes[i]) {
			utf16Off += utf16RuneLen(runes[i])
			i++
			continue
		}
		start := i
		startOff := utf16Off
		for i < len(runes) && !unicode.IsSpace(runes[i]) {
			utf16Off += utf16RuneLen(runes[i])
			i++
		}
		tokens = append(tokens, wordToken{
			text:   string(runes[start:i]),
			offset: startOff,
			length: utf16Off - startOff,
		})
	}
	return tokens
}

func lcsTable(a, b []wordToken) [][]int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1].text == b[j-1].text {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	return dp
}

func findChangedWords(a, b []wordToken) (changedA, changedB []bool) {
	dp := lcsTable(a, b)
	changedA = make([]bool, len(a))
	changedB = make([]bool, len(b))
	i, j := len(a), len(b)
	for i > 0 && j > 0 {
		if a[i-1].text == b[j-1].text {
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			changedA[i-1] = true
			i--
		} else {
			changedB[j-1] = true
			j--
		}
	}
	for i > 0 {
		changedA[i-1] = true
		i--
	}
	for j > 0 {
		changedB[j-1] = true
		j--
	}
	return
}

func boldEntitiesForChanged(tokens []wordToken, changed []bool) []tele.MessageEntity {
	var entities []tele.MessageEntity
	i := 0
	for i < len(tokens) {
		if !changed[i] {
			i++
			continue
		}
		startOff := tokens[i].offset
		endOff := tokens[i].offset + tokens[i].length
		i++
		for i < len(tokens) && changed[i] {
			endOff = tokens[i].offset + tokens[i].length
			i++
		}
		entities = append(entities, tele.MessageEntity{
			Type:   tele.EntityBold,
			Offset: startOff,
			Length: endOff - startOff,
		})
	}
	return entities
}

func computeDiffBoldEntities(oldText, newText string) (oldEntities, newEntities []tele.MessageEntity) {
	oldTokens := tokenizeWithOffsets(oldText)
	newTokens := tokenizeWithOffsets(newText)
	changedOld, changedNew := findChangedWords(oldTokens, newTokens)
	oldEntities = boldEntitiesForChanged(oldTokens, changedOld)
	newEntities = boldEntitiesForChanged(newTokens, changedNew)
	return
}
