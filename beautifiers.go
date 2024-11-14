package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/muesli/termenv"
)

type Beautifier = func(*termenv.Output, string) (string, bool)

type FormatFn = func(*termenv.Output, string) termenv.Style
type TextPart struct {
	Name  string
	Value string
}
type BeautifierData struct {
	Pattern    *regexp.Regexp
	FormatFns  map[string]FormatFn
	Preprocess func(*termenv.Output, []TextPart) []TextPart
}

var BEAUTIFIERS = []Beautifier{
	makeBeautifier(BeautifierData{
		Pattern: regexp.MustCompile(`^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}[+-]\d{2}:\d{2})(\s)(?P<level>\s*\w+)(\s)(?P<pid>\d+)(\s+)(?P<separator>---)(\s+)(?P<thread>\[.*?\])(\s+)(?P<logger>\S+)(\s+)(?P<colon>:)(\s+)(?P<message>.*)$`),
		FormatFns: map[string]FormatFn{
			"timestamp": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Italic().Faint()
			},
			"level": func(o *termenv.Output, v string) termenv.Style {
				color := "15"
				fatal := false
				switch strings.TrimSpace(v) {
				case "TRACE":
					color = "6"
				case "DEBUG":
					color = "4"
				case "INFO":
					color = "2"
				case "WARN":
					color = "3"
				case "ERROR":
					color = "1"
				case "FATAL":
					color = "9"
					fatal = true
				}

				res := o.String(" " + v + " ").Foreground(o.Color("0")).Background(o.Color(color)).Bold()
				if fatal {
					res = res.Italic().Underline()
				}
				return res
			},
			"pid": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("5"))
			},
			"separator": faint,
			"thread":    faint,
			"logger": func(o *termenv.Output, v string) termenv.Style {
				// djb2 hashing function
				var hash uint32 = 5381
				for i := 0; i < len(v); i++ {
					hash = ((hash << 5) + hash) + uint32(v[i])
				}

				// bright colors only
				h := float64(hash % 360)
				s := float64(70 + (hash % 30))
				l := float64(60 + (hash % 10))

				// convert hsl to hex: https://stackoverflow.com/a/44134328
				l /= 100
				a := s * min(l, 1-l) / 100
				f := func(n float64) string {
					k := math.Mod(n+h/30, 12)
					color := l - a*max(min(min(k-3, 9-k), 1), -1)
					hex := strconv.FormatInt(int64(math.Round(255*color)), 16)
					if len(hex) == 1 {
						hex = "0" + hex
					}
					return hex
				}
				color := fmt.Sprintf("#%s%s%s", f(0), f(8), f(4))

				parts := strings.Split(v, ".")
				mainPart := ""
				var sb strings.Builder
				for i, part := range parts {
					if i != 0 {
						sb.WriteString(".")
					}

					if i == len(parts)-1 && isFirstCharUppercase(part) {
						mainPart = part
					} else {
						sb.WriteString(part)
					}
				}

				return o.String(
					o.String(sb.String()).Foreground(o.Color(color)).String() +
						o.String(mainPart).Foreground(o.Color(color)).Bold().String(),
				)
			},
			"colon":     faint,
			"message":   highlightUrls,
			"sql_debug": faint,
		},
		Preprocess: func(o *termenv.Output, ps []TextPart) []TextPart {
			sqlDebug := false
			for i := range ps {
				if sqlDebug {
					ps[i].Name = "sql_debug"
				}

				p := ps[i]
				if p.Name == "logger" && p.Value == "org.hibernate.SQL" {
					sqlDebug = true
				}
			}

			return ps
		},
	}),
	makeBeautifier(BeautifierData{
		Pattern:   regexp.MustCompile(`^(Hibernate: )(?P<query>.*)$`),
		FormatFns: map[string]FormatFn{"query": faint},
	}),
	makeBeautifier(BeautifierData{
		Pattern: regexp.MustCompile(`^(?P<caused_by>Caused by: )?(?P<exception_path>[^ ]+\.)(?P<exception_name>[^ ]*Exception)(?P<colon>:)(?P<message>.*)$`),
		FormatFns: map[string]FormatFn{
			"caused_by": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("1")).Bold()
			},
			"exception_path": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("1"))
			},
			"exception_name": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("1")).Bold()
			},
			"colon": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("1")).Faint()
			},
			"message": highlightUrls,
		},
	}),
	makeBeautifier(BeautifierData{
		Pattern: regexp.MustCompile(`(?P<at>\tat )(?P<class>[^ ]+\.)(?P<method>[^ ]+\()(?P<file>[^ ]+:\d+)(?P<method>\))(?P<jar> ~\[[^ ]+:[^ ]+\])?`),
		FormatFns: map[string]FormatFn{
			"at": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("1")).Bold()
			},
			"class":     faint,
			"file":      bold,
			"jar":       faint,
			"java_base": faint,
		},
		Preprocess: func(o *termenv.Output, ps []TextPart) []TextPart {
			isJavaBase := false
			for i := range ps {
				p := ps[i]
				if p.Name == "class" && strings.HasPrefix(p.Value, "java.base") {
					isJavaBase = true
					break
				}
			}

			if isJavaBase {
				for i := range ps {
					ps[i].Name = "java_base"
				}
			}

			return ps
		},
	}),
	makeBeautifier(BeautifierData{
		Pattern: regexp.MustCompile(`^(> Task )(?P<name>:[^ ]+)(?P<state> [^ ]+)?$`),
		FormatFns: map[string]FormatFn{
			"name": bold,
			"state": func(o *termenv.Output, v string) termenv.Style {
				color := "15"
				switch strings.TrimSpace(v) {
				case "UP-TO-DATE":
					color = "2"
				case "FAILED":
					color = "1"
				}

				return o.String(v).Foreground(o.Color(color))
			},
		},
	}),
	makeBeautifier(BeautifierData{
		Pattern: regexp.MustCompile(`^(?:(?P<successful>BUILD SUCCESSFUL)|(?P<failed>BUILD FAILED))( in .+)$`),
		FormatFns: map[string]FormatFn{
			"successful": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("2")).Bold()
			},
			"failed": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("1")).Bold()
			},
		},
	}),
	makeBeautifier(BeautifierData{
		Pattern: regexp.MustCompile("^(?:(?P<crystal>  \\.)(?P<logo>   ____          _            )(?P<chevrons>__ _ _))|(?:(?P<crystal> /\\\\\\\\)(?P<logo> / ___'_ __ _ _\\(_\\)_ __  __ _ )(?P<chevrons>\\\\ \\\\ \\\\ \\\\))|(?:(?P<crystal>\\( \\( \\))(?P<logo>\\\\___ \\| '_ \\| '_\\| \\| '_ \\\\/ _` \\| )(?P<chevrons>\\\\ \\\\ \\\\ \\\\))|(?:(?P<crystal> \\\\\\\\/)(?P<logo>  ___\\)\\| \\|_\\)\\| \\| \\| \\| \\| \\|\\| \\(_\\| \\|  )(?P<chevrons>\\) \\) \\) \\)))|(?:(?P<crystal>  '  )(?P<logo>\\|____\\| \\.__\\|_\\| \\|_\\|_\\| \\|_\\\\__, \\|)(?P<chevrons> / / / /))|(?:(?P<underline> =========)(?P<logo>\\|_\\|)(?P<underline>==============)(?P<logo>\\|___/)(?P<underline>=)(?P<chevrons>/_/_/_/))$"),

		FormatFns: map[string]FormatFn{
			"logo": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("2"))
			},
			"crystal": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("2")).Bold()
			},
			"chevrons": func(o *termenv.Output, v string) termenv.Style {
				return o.String(v).Foreground(o.Color("2"))
			},
			"underline": bold,
		},
	}),
	makeBeautifier(BeautifierData{
		Pattern:   regexp.MustCompile(`^(?P<name> :: Spring Boot :: )(\s*)(?P<version>.+)$`),
		FormatFns: map[string]FormatFn{"name": bold, "version": faint},
	}),
}

func makeBeautifier(b BeautifierData) Beautifier {
	return func(o *termenv.Output, v string) (string, bool) {
		matches := b.Pattern.FindStringSubmatch(v)
		if matches == nil {
			return "", false
		}

		parts := make([]TextPart, 0)
		for i, name := range b.Pattern.SubexpNames() {
			if i == 0 {
				continue
			}

			parts = append(parts, TextPart{
				Name:  name,
				Value: matches[i],
			})
		}

		if b.Preprocess != nil {
			parts = b.Preprocess(o, parts)
		}

		var sb strings.Builder
		for _, part := range parts {
			format := b.FormatFns[part.Name]
			if format == nil {
				sb.WriteString(part.Value)
			} else {
				sb.WriteString(format(o, part.Value).String())
			}
		}

		return sb.String(), true
	}
}

func faint(o *termenv.Output, v string) termenv.Style {
	return o.String(v).Faint()
}

func bold(o *termenv.Output, v string) termenv.Style {
	return o.String(v).Bold()
}

func isFirstCharUppercase(s string) bool {
	if len(s) == 0 {
		return false
	}
	firstRune, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(firstRune)
}

var URL_REGEX = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)

func highlightUrls(o *termenv.Output, v string) termenv.Style {
	parts := URL_REGEX.FindAllStringIndex(v, -1)
	lastIndex := 0
	var sb strings.Builder

	for _, url := range parts {
		if url[0] > lastIndex {
			sb.WriteString(v[lastIndex:url[0]])
		}

		sb.WriteString(o.String(v[url[0]:url[1]]).Underline().String())
		lastIndex = url[1]
	}

	if lastIndex < len(v) {
		sb.WriteString(v[lastIndex:])
	}

	return o.String(sb.String())
}
