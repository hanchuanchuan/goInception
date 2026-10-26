package session

import (
	"fmt"
	"io"

	ghostlog "github.com/openark/golib/log"
	log "github.com/sirupsen/logrus"
)

// ghostLogger adapts gh-ost's logger to the goInception process. It keeps the
// gh-ost output available to the existing progress parser through an io.Pipe,
// while also forwarding diagnostics to goInception's logger.
type ghostLogger struct {
	writer io.Writer
}

func (l *ghostLogger) write(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if l.writer != nil {
		_, _ = io.WriteString(l.writer, message+"\n")
	}
	log.Debug(message)
}

func (l *ghostLogger) Debug(args ...interface{})                 { l.write("%s", fmt.Sprint(args...)) }
func (l *ghostLogger) Debugf(format string, args ...interface{}) { l.write(format, args...) }
func (l *ghostLogger) Info(args ...interface{})                  { l.write("%s", fmt.Sprint(args...)) }
func (l *ghostLogger) Infof(format string, args ...interface{})  { l.write(format, args...) }
func (l *ghostLogger) Warning(args ...interface{}) error {
	l.write("%s", fmt.Sprint(args...))
	return nil
}
func (l *ghostLogger) Warningf(format string, args ...interface{}) error {
	l.write(format, args...)
	return nil
}
func (l *ghostLogger) Error(args ...interface{}) error {
	l.write("%s", fmt.Sprint(args...))
	return nil
}
func (l *ghostLogger) Errorf(format string, args ...interface{}) error {
	l.write(format, args...)
	return nil
}
func (l *ghostLogger) Errore(err error) error {
	if err != nil {
		l.write("%s", err.Error())
	}
	return err
}
func (l *ghostLogger) Fatal(args ...interface{}) error {
	err := fmt.Errorf("%s", fmt.Sprint(args...))
	l.write("%s", err.Error())
	return err
}
func (l *ghostLogger) Fatalf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	l.write("%s", err.Error())
	return err
}
func (l *ghostLogger) Fatale(err error) error {
	if err != nil {
		l.write("%s", err.Error())
	}
	return err
}
func (l *ghostLogger) SetLevel(ghostlog.LogLevel) {}
func (l *ghostLogger) SetPrintStackTrace(bool)    {}
