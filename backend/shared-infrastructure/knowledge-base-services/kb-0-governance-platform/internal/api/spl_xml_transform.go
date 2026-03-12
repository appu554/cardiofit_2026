package api

import (
	"regexp"
	"strings"
)

// =============================================================================
// SPL XML → HTML TRANSFORMER
// =============================================================================
// Converts HL7 Structured Product Labeling (SPL) inner XML into browser-
// renderable HTML. SPL uses non-standard tags like <paragraph>, <content>,
// <list>, <item>, and <linkHtml> that browsers don't render natively.
//
// Tag mapping:
//   <paragraph>              → <p>
//   <content styleCode="bold"> → <strong>
//   <content styleCode="italics"> → <em>
//   <content styleCode="underline"> → <u>
//   <content>                → <span>
//   <list listType="ordered"> → <ol>
//   <list ...>               → <ul>
//   <item>                   → <li>
//   <linkHtml href="...">    → <a href="...">
//   <table> family           → kept as-is (already valid HTML)
//   styleCode on <td>/<tr>   → converted to CSS border styles
// =============================================================================

var (
	// Tag replacements — compiled once at init
	reParagraphOpen  = regexp.MustCompile(`<paragraph\b[^>]*>`)
	reParagraphClose = regexp.MustCompile(`</paragraph>`)
	reParagraphSelf  = regexp.MustCompile(`<paragraph\s*/>`)

	reContentBold      = regexp.MustCompile(`<content\b[^>]*styleCode="[^"]*bold[^"]*"[^>]*>`)
	reContentItalics   = regexp.MustCompile(`<content\b[^>]*styleCode="[^"]*italics[^"]*"[^>]*>`)
	reContentUnderline = regexp.MustCompile(`<content\b[^>]*styleCode="[^"]*underline[^"]*"[^>]*>`)
	reContentOther     = regexp.MustCompile(`<content\b[^>]*>`)
	reContentClose     = regexp.MustCompile(`</content>`)

	reListOrdered = regexp.MustCompile(`<list\b[^>]*listType="ordered"[^>]*>`)
	reListAny     = regexp.MustCompile(`<list\b[^>]*>`)
	reListClose   = regexp.MustCompile(`</list>`)

	reItemOpen  = regexp.MustCompile(`<item\b[^>]*>`)
	reItemClose = regexp.MustCompile(`</item>`)

	reLinkOpen  = regexp.MustCompile(`<linkHtml\b([^>]*)>`)
	reLinkClose = regexp.MustCompile(`</linkHtml>`)
	reLinkHref  = regexp.MustCompile(`href="([^"]*)"`)

	// styleCode on table cells → CSS border classes
	reStyleCode   = regexp.MustCompile(`\s*styleCode="([^"]*)"`)
	reTdWithStyle = regexp.MustCompile(`<(t[dh])\b([^>]*)styleCode="([^"]*)"([^>]*)>`)

	// XML namespace declarations and processing instructions
	reXMLNS = regexp.MustCompile(`\s+xmlns(?::\w+)?="[^"]*"`)
	reXMLPI = regexp.MustCompile(`<\?xml[^?]*\?>`)

	// Footnote markers
	reFootnoteRef = regexp.MustCompile(`<sup\b[^>]*>`)

	// Render section headers from <title> tags
	reTitleOpen  = regexp.MustCompile(`<title\b[^>]*>`)
	reTitleClose = regexp.MustCompile(`</title>`)

	// Clean empty paragraphs
	reEmptyP = regexp.MustCompile(`<p>\s*</p>`)

	// Clean up excessive whitespace between tags
	reMultiNewline = regexp.MustCompile(`\n{3,}`)
)

// TransformSPLToHTML converts SPL inner XML to browser-renderable HTML.
func TransformSPLToHTML(splXML string) string {
	if splXML == "" {
		return ""
	}

	html := splXML

	// Remove XML namespace declarations and processing instructions
	html = reXMLNS.ReplaceAllString(html, "")
	html = reXMLPI.ReplaceAllString(html, "")

	// ── Content tags (must process BEFORE paragraph to handle nesting) ──
	// Order matters: bold/italics/underline first, then generic <content>
	html = reContentBold.ReplaceAllString(html, "<strong>")
	html = reContentItalics.ReplaceAllString(html, "<em>")
	html = reContentUnderline.ReplaceAllString(html, "<u>")
	html = reContentOther.ReplaceAllString(html, "<span>")
	html = reContentClose.ReplaceAllString(html, "</span>") // generic close

	// Fix close tags for styled content (</span> should match the open)
	// Since we replaced opens with <strong>/<em>/<u>, the closes need fixing.
	// Use a stack-based approach for proper nesting.
	html = fixContentCloseTags(html)

	// ── Paragraph tags ──────────────────────────────────────────────────
	html = reParagraphSelf.ReplaceAllString(html, "<br/>")
	html = reParagraphOpen.ReplaceAllString(html, "<p>")
	html = reParagraphClose.ReplaceAllString(html, "</p>")

	// ── List tags ────────────────────────────────────────────────────────
	html = reListOrdered.ReplaceAllString(html, "<ol>")
	html = reListAny.ReplaceAllString(html, "<ul>")
	html = reListClose.ReplaceAllString(html, "</ul>") // We'll fix ol closes below

	// ── Item tags ────────────────────────────────────────────────────────
	html = reItemOpen.ReplaceAllString(html, "<li>")
	html = reItemClose.ReplaceAllString(html, "</li>")

	// ── Link tags ────────────────────────────────────────────────────────
	html = reLinkOpen.ReplaceAllStringFunc(html, func(match string) string {
		hrefMatch := reLinkHref.FindStringSubmatch(match)
		if len(hrefMatch) > 1 {
			return `<a href="` + hrefMatch[1] + `" class="text-blue-600 underline">`
		}
		return "<a>"
	})
	html = reLinkClose.ReplaceAllString(html, "</a>")

	// ── Title tags → h3 ─────────────────────────────────────────────────
	html = reTitleOpen.ReplaceAllString(html, `<h3 class="font-bold text-gray-900 mt-4 mb-2">`)
	html = reTitleClose.ReplaceAllString(html, "</h3>")

	// ── Table cell styleCode → CSS borders ──────────────────────────────
	html = reTdWithStyle.ReplaceAllStringFunc(html, convertStyleCodeToCSS)

	// Remove remaining styleCode attributes on other elements
	html = reStyleCode.ReplaceAllString(html, "")

	// ── Cleanup ──────────────────────────────────────────────────────────
	html = reEmptyP.ReplaceAllString(html, "")
	html = reMultiNewline.ReplaceAllString(html, "\n\n")
	html = strings.TrimSpace(html)

	return html
}

// fixContentCloseTags fixes mismatched close tags from the regex replacement.
// Since we replace <content styleCode="bold"> with <strong> but all </content>
// with </span>, we need to properly close <strong>, <em>, <u> tags.
func fixContentCloseTags(html string) string {
	// Track open tags and replace </span> with the correct close tag
	var result strings.Builder
	result.Grow(len(html))

	stack := make([]string, 0, 16) // stack of open tag types

	i := 0
	for i < len(html) {
		// Check for opening styled tags
		if i+8 <= len(html) && html[i:i+8] == "<strong>" {
			stack = append(stack, "strong")
			result.WriteString("<strong>")
			i += 8
			continue
		}
		if i+4 <= len(html) && html[i:i+4] == "<em>" {
			stack = append(stack, "em")
			result.WriteString("<em>")
			i += 4
			continue
		}
		if i+3 <= len(html) && html[i:i+3] == "<u>" {
			stack = append(stack, "u")
			result.WriteString("<u>")
			i += 3
			continue
		}
		if i+6 <= len(html) && html[i:i+6] == "<span>" {
			stack = append(stack, "span")
			result.WriteString("<span>")
			i += 6
			continue
		}

		// Check for </span> — replace with matching close tag
		if i+7 <= len(html) && html[i:i+7] == "</span>" {
			if len(stack) > 0 {
				tag := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				result.WriteString("</" + tag + ">")
			} else {
				result.WriteString("</span>")
			}
			i += 7
			continue
		}

		result.WriteByte(html[i])
		i++
	}

	return result.String()
}

// convertStyleCodeToCSS converts SPL styleCode on table cells to CSS border styles.
func convertStyleCodeToCSS(match string) string {
	parts := reTdWithStyle.FindStringSubmatch(match)
	if len(parts) < 5 {
		return match
	}

	tag := parts[1]       // "td" or "th"
	before := parts[2]    // attrs before styleCode
	styleCode := parts[3] // "Botrule Lrule Rrule" etc.
	after := parts[4]     // attrs after styleCode

	var styles []string
	codes := strings.Fields(styleCode)
	for _, code := range codes {
		switch strings.ToLower(code) {
		case "botrule":
			styles = append(styles, "border-bottom: 1px solid #d1d5db")
		case "toprule":
			styles = append(styles, "border-top: 1px solid #d1d5db")
		case "lrule":
			styles = append(styles, "border-left: 1px solid #d1d5db")
		case "rrule":
			styles = append(styles, "border-right: 1px solid #d1d5db")
		}
	}

	styleAttr := ""
	if len(styles) > 0 {
		styleAttr = ` style="` + strings.Join(styles, "; ") + `"`
	}

	return "<" + tag + before + after + styleAttr + ">"
}
