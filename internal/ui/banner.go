package ui

// Banner is the stylized text logo displayed at startup.
const Banner = `
    ╔══════════════════════════════════════════════════╗
    ║                                                  ║
    ║   ███╗   ███╗ █████╗  ██████╗                    ║
    ║   ████╗ ████║██╔══██╗██╔════╝                    ║
    ║   ██╔████╔██║███████║██║                         ║
    ║   ██║╚██╔╝██║██╔══██║██║                         ║
    ║   ██║ ╚═╝ ██║██║  ██║╚██████╗                    ║
    ║   ╚═╝     ╚═╝╚═╝  ╚═╝ ╚═════╝                    ║
    ║    ██████╗██╗     ███████╗ █████╗ ███╗   ██╗     ║
    ║   ██╔════╝██║     ██╔════╝██╔══██╗████╗  ██║     ║
    ║   ██║     ██║     █████╗  ███████║██╔██╗ ██║     ║
    ║   ██║     ██║     ██╔══╝  ██╔══██║██║╚██╗██║     ║
    ║   ╚██████╗███████╗███████╗██║  ██║██║ ╚████║     ║
    ║    ╚═════╝╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝     ║
    ║                                                  ║
    ╚══════════════════════════════════════════════════╝`

// PrintBanner displays the stylized banner with the project tagline.
func (o *Output) PrintBanner() {
	o.println("")
	for _, line := range splitLines(Banner) {
		o.println(o.color(colorYellow, line))
	}
	o.println("")
	o.println(o.color(colorBold+colorCyan, "       \"I need disk space for my bunghole!\""))
	o.println(o.color(colorDim, "        Are you threatening my disk space?"))
	o.println("")
}

// splitLines splits a string into individual lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
