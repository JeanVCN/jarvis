package main

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	headingRE     = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.+?)\s*#*\s*$`)
	unorderedRE   = regexp.MustCompile(`^(\s*)[-+*]\s+`)
	linkRE        = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)]+)\)`)
	strongStarRE  = regexp.MustCompile(`\*\*(.+?)\*\*`)
	strongUnderRE = regexp.MustCompile(`__(.+?)__`)
	strikeRE      = regexp.MustCompile(`~~(.+?)~~`)
	italicRE      = regexp.MustCompile(`(^|[^*])\*([^*\n]+)\*([^*]|$)`)
	tableSepRE    = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$`)
)

// formatWhatsAppMarkdown adapts common GitHub Markdown to WhatsApp's smaller
// formatting set while leaving fenced and inline code untouched.
func formatWhatsAppMarkdown(input string) string {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	var out []string
	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := "```"
			if strings.HasPrefix(trimmed, "~~~") {
				marker = "~~~"
			}
			language := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, marker), marker))
			if language != "" {
				out = append(out, "Código ("+language+"):")
			}
			out = append(out, "```")
			i++
			for i < len(lines) {
				current := strings.TrimSpace(lines[i])
				if strings.HasPrefix(current, marker) {
					out = append(out, "```")
					i++
					break
				}
				out = append(out, lines[i])
				i++
			}
			continue
		}

		// Render GitHub tables as one compact, labeled bullet per row.
		if i+1 < len(lines) && strings.Contains(line, "|") && tableSepRE.MatchString(lines[i+1]) {
			headers := splitTableRow(line)
			for n := range headers {
				headers[n] = formatInline(headers[n])
			}
			i += 2
			for i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != "" {
				cells := splitTableRow(lines[i])
				var fields []string
				for n, value := range cells {
					if value == "" {
						continue
					}
					label := "Coluna " + itoa(n+1)
					if n < len(headers) && headers[n] != "" {
						label = headers[n]
					}
					fields = append(fields, label+": "+formatInline(value))
				}
				if len(fields) > 0 {
					out = append(out, "• "+strings.Join(fields, " · "))
				}
				i++
			}
			continue
		}

		if heading := headingRE.FindStringSubmatch(line); heading != nil {
			out = append(out, "*"+formatInline(heading[1])+"*")
			i++
			continue
		}
		if strings.Trim(trimmed, "-*_ ") == "" && len(trimmed) >= 3 {
			out = append(out, "────────────────")
			i++
			continue
		}
		line = unorderedRE.ReplaceAllString(line, "$1• ")
		line = formatInline(line)
		out = append(out, line)
		i++
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func formatInline(line string) string {
	// Replace inline code with placeholders so Markdown replacements do not
	// alter code contents, then restore it after formatting.
	var code []string
	var b strings.Builder
	for i := 0; i < len(line); {
		if line[i] != '`' {
			b.WriteByte(line[i])
			i++
			continue
		}
		end := strings.IndexByte(line[i+1:], '`')
		if end < 0 {
			b.WriteByte(line[i])
			i++
			continue
		}
		end += i + 1
		code = append(code, line[i:end+1])
		b.WriteString("\x00CODE")
		b.WriteString(itoa(len(code) - 1))
		b.WriteByte('\x00')
		i = end + 1
	}
	line = b.String()
	line = linkRE.ReplaceAllString(line, "$1 ($2)")
	var bold []string
	line = strongStarRE.ReplaceAllStringFunc(line, func(match string) string {
		inner := match[2 : len(match)-2]
		bold = append(bold, inner)
		return "\x00BOLD" + itoa(len(bold)-1) + "\x00"
	})
	line = strongUnderRE.ReplaceAllStringFunc(line, func(match string) string {
		inner := match[2 : len(match)-2]
		bold = append(bold, inner)
		return "\x00BOLD" + itoa(len(bold)-1) + "\x00"
	})
	line = italicRE.ReplaceAllString(line, "${1}_${2}_${3}")
	line = strikeRE.ReplaceAllString(line, "~$1~")
	for i, content := range bold {
		line = strings.ReplaceAll(line, "\x00BOLD"+itoa(i)+"\x00", "*"+content+"*")
	}
	for i, snippet := range code {
		line = strings.ReplaceAll(line, "\x00CODE"+itoa(i)+"\x00", snippet)
	}
	return line
}

// splitWhatsAppMessage keeps messages below WhatsApp's practical text limit.
// It prefers line and word boundaries and balances fenced code blocks when a
// response spans multiple messages.
func splitWhatsAppMessage(input string, maxBytes int) []string {
	if maxBytes < 128 {
		maxBytes = 128
	}
	if len(input) <= maxBytes {
		return []string{input}
	}
	var chunks []string
	var current strings.Builder
	fenceOpen := false
	appendLine := func(line string) {
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)
	}
	flush := func() {
		value := strings.TrimRight(current.String(), "\n")
		if fenceOpen {
			value += "\n```"
		}
		if value != "" {
			chunks = append(chunks, value)
		}
		current.Reset()
		if fenceOpen {
			current.WriteString("```\n")
		}
	}
	for _, line := range strings.Split(input, "\n") {
		for len(line) > maxBytes-16 {
			cut := maxBytes - 16
			if space := strings.LastIndex(line[:cut], " "); space > 0 {
				cut = space
			}
			for cut > 0 && !utf8.RuneStart(line[cut]) {
				cut--
			}
			if cut == 0 {
				_, cut = utf8.DecodeRuneInString(line)
			}
			piece := line[:cut]
			line = strings.TrimLeft(line[cut:], " ")
			if current.Len()+len(piece)+2 > maxBytes {
				flush()
			}
			appendLine(piece)
			flush()
		}
		if current.Len()+len(line)+2 > maxBytes {
			flush()
		}
		appendLine(line)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			fenceOpen = !fenceOpen
		}
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
