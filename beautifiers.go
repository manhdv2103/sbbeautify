package main

import (
	"regexp"
	"strings"

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
				return o.String(v).Foreground(o.Color("6"))
			},
			"colon":     faint,
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
