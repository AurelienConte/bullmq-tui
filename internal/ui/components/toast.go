package components

import (
	"time"
)

type ToastType int

const (
	ToastInfo ToastType = iota
	ToastError
	ToastSuccess
)

type Toast struct {
	message   string
	toastType ToastType
	createdAt time.Time
	duration  time.Duration
}

func NewToast(message string, toastType ToastType) *Toast {
	return &Toast{
		message:   message,
		toastType: toastType,
		createdAt: time.Now(),
		duration:  3 * time.Second,
	}
}

func (t *Toast) IsExpired() bool {
	return time.Since(t.createdAt) > t.duration
}

func (t Toast) View() string {
	style := ToastStyle
	switch t.toastType {
	case ToastError:
		style = ToastErrorStyle
	case ToastSuccess:
		style = ToastStyle
	}

	return style.Render(t.message)
}
