package ui

// banner is the Cornholio ASCII art displayed at startup.
// Beavis in his iconic Cornholio pose — shirt pulled over his head, arms raised.
const banner = `
                          _..---.._
                        .'          '.
                       /  O        O  \
                      |        __      |
                      |      /    \    |
                       \     '----'   /
                     _.-'-.__    __.-'-._
                   .'  \     '~~'     /  '.
                 .'     \            /     '.
                /   .-.  \    /\   /  .-.   \
               /   /   \  \  /  \ /  /   \   \
              /   /     \  \/    \/  /     \   \
             |   |       |  |    |  |       |   |
             |   |       |  |    |  |       |   |
             |   |       |  |    |  |       |   |
              \   \     /  / \  / \  \     /   /
               \   '---'  /   \/   \  '---'   /
                \        /     |    \        /
                 |      |      |     |      |
                 |      |      |     |      |
                 |      |      |     |      |
                 |______|      |     |______|
                 |      |      |     |      |
                 |      |      |     |      |
                _|      |_    _|     |      |_
               (_|______|_)  (_|_____|______|_)
`

// PrintBanner displays the Cornholio ASCII art banner with the project name.
func (o *Output) PrintBanner() {
	o.println("")
	for _, line := range splitLines(banner) {
		o.println(o.color(colorYellow, line))
	}

	o.println("")
	o.println(o.color(colorBold+colorYellow, "              I AM CORNHOLIO!"))
	o.println(o.color(colorYellow, "        I NEED DISK SPACE FOR MY BUNGHOLE!"))
	o.println("")
	o.println(o.color(colorBold+colorCyan, "        ═══════════════════════════════════"))
	o.println(o.color(colorBold+colorCyan, "           🧹  m a c - c l e a n e r  🧹"))
	o.println(o.color(colorBold+colorCyan, "        ═══════════════════════════════════"))
	o.println(o.color(colorDim, "          Are you threatening my disk space?"))
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
