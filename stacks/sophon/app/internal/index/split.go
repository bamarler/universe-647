package index

import "strings"

const (
	maxChunkChars = 1600 // ~400 tokens
	overlapChars  = 200  // ~12% overlap between windows
)

// splitMarkdown does heading-aware splitting: sections by markdown heading,
// long sections windowed with overlap, every chunk prefixed with the doc
// title + heading path (contextual chunk headers).
func splitMarkdown(title, body string) []string {
	if strings.TrimSpace(body) == "" {
		return []string{"Note: " + title}
	}

	type section struct {
		heading string
		text    strings.Builder
	}
	sections := []*section{{heading: ""}}
	for _, line := range strings.Split(body, "\n") {
		if h := strings.TrimLeft(line, "#"); len(h) < len(line) && strings.HasPrefix(line, "#") {
			sections = append(sections, &section{heading: strings.TrimSpace(h)})
			continue
		}
		cur := sections[len(sections)-1]
		cur.text.WriteString(line)
		cur.text.WriteString("\n")
	}

	var out []string
	for _, sec := range sections {
		text := strings.TrimSpace(sec.text.String())
		if text == "" {
			continue
		}
		prefix := "Note: " + title
		if sec.heading != "" {
			prefix += " > " + sec.heading
		}
		for _, window := range windows(text) {
			out = append(out, prefix+"\n"+window)
		}
	}
	if len(out) == 0 {
		out = []string{"Note: " + title}
	}
	return out
}

func windows(text string) []string {
	if len(text) <= maxChunkChars {
		return []string{text}
	}
	var out []string
	for start := 0; start < len(text); start += maxChunkChars - overlapChars {
		end := min(start+maxChunkChars, len(text))
		out = append(out, text[start:end])
		if end == len(text) {
			break
		}
	}
	return out
}
