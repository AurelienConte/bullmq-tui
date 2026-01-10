package components

import (
	"github.com/charmbracelet/lipgloss"
)

var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

type Sparkline struct {
	values []int
	width  int
	style  lipgloss.Style
}

func NewSparkline(width int) Sparkline {
	return Sparkline{
		width: width,
		style: lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
	}
}

func (s *Sparkline) SetValues(values []int) {
	s.values = values
}

func (s *Sparkline) SetWidth(width int) {
	s.width = width
}

func (s Sparkline) View() string {
	if len(s.values) == 0 || s.width == 0 {
		return ""
	}

	// Find min/max
	min, max := s.values[0], s.values[0]
	for _, v := range s.values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Handle flat line
	if max == min {
		max = min + 1
	}

	// Build sparkline
	var result []rune

	// Sample or pad to fit width
	data := s.resample(s.values, s.width)

	for _, v := range data {
		// Normalize to 0-7 range
		normalized := float64(v-min) / float64(max-min)
		index := int(normalized * float64(len(sparkBlocks)-1))
		if index >= len(sparkBlocks) {
			index = len(sparkBlocks) - 1
		}
		result = append(result, sparkBlocks[index])
	}

	return s.style.Render(string(result))
}

func (s Sparkline) resample(values []int, targetLen int) []int {
	if len(values) == targetLen {
		return values
	}

	if len(values) < targetLen {
		// Pad with zeros at the beginning
		result := make([]int, targetLen)
		offset := targetLen - len(values)
		copy(result[offset:], values)
		return result
	}

	// Downsample by taking evenly spaced points
	result := make([]int, targetLen)
	step := float64(len(values)) / float64(targetLen)
	for i := 0; i < targetLen; i++ {
		idx := int(float64(i) * step)
		if idx >= len(values) {
			idx = len(values) - 1
		}
		result[i] = values[idx]
	}
	return result
}
