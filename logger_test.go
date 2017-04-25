package main

import (
	"bytes"
	"errors"
	"testing"
)

func Test_PrefixValidStringForLogLevel(t *testing.T) {
	for i := LevelEmergency; i <= LevelDebug; i++ {
		p := getPrefix(i)

		switch i {
		case LevelEmergency:
			if p != "[emergency] " {
				t.Error("Unxpected prefix at level ", i, ": ", p)
			}

		case LevelAlert:
			if p != "[alert] " {
				t.Error("Unxpected prefix at level ", i, ": ", p)
			}

		case LevelCritical:
			if p != "[critical] " {
				t.Error("Unxpected prefix at level ", i, ": ", p)
			}

		case LevelError:
			if p != "[error] " {
				t.Error("Unxpected prefix at level ", i, ": ", p)
			}

		case LevelWarning:
			if p != "[warning] " {
				t.Error("Unxpected prefix at level ", i, ": ", p)
			}

		case LevelNotice:
			if p != "[notice] " {
				t.Error("Unxpected prefix at level ", i, ": ", p)
			}

		case LevelInformational:
			if p != "[info] " {
				t.Error("Unxpected prefix at level ", i, ": ", p)
			}

		case LevelDebug:
			if p != "[debug] " {
				t.Error("Unxpected prefix at level ", i, ": ", p)
			}
		}
	}
}

func Test_PrivatePrintValid(t *testing.T) {
	CONSOLELOG = 7

	var (
		b = bytes.NewBuffer(nil)
		l = NewLogger("sbss-vbook", 7)

		wrap = func(level int, v ...interface{}) {
			l.print(level, v)
		}
	)

	l.syslog = nil
	l.SetOutput(b)

	wrap(LevelError, errors.New("Unknown error"))
	if !bytes.Contains(b.Bytes(), []byte("[error] Unknown error")) {
		t.Errorf("Unexpected message: %s", b.String())
	}

	b.Reset()

	wrap(LevelInformational, "Some action done at line %d", 12)
	if !bytes.Contains(b.Bytes(), []byte("[info] Some action done at line 12")) {
		t.Errorf("Unexpected message: %s", b.String())
	}

	b.Reset()

	wrap(LevelInformational)
	if b.Len() > 0 {
		t.Errorf("Unexpected buffer size")
	}
}

func Test_LevelControlValid(t *testing.T) {
	var (
		b   = bytes.NewBuffer(nil)
		log = NewLogger("sbss-vbook", 7)
	)

	log.syslog = nil
	log.SetOutput(b)

	for i := LevelEmergency; i < LevelDebug+1; i++ {
		log.SetLevel(i)

		for l := LevelEmergency; l < LevelDebug+1; l++ {
			if l > i && log.level(l) == true {
				t.Errorf("Expected true for the %d at level %d", l, i)
			}
		}
	}
}
