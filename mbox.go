package main

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

////////////////////////////////////////////////////////////////////////////
// Constant and data type/structure definitions

const envelopeFrom = "cloudmail"

type mbox struct {
	io.Writer
}

type validation func([]byte, time.Time) error

////////////////////////////////////////////////////////////////////////////
// Function definitions

func newMbox(w io.Writer) *mbox {
	return &mbox{w}
}

func (m *mbox) writeMessage(rfc822 []byte, okToGo validation) error {

	r := regexp.MustCompile(`\nDate: +(.*?)\r*\n`).FindSubmatch(rfc822)
	r1 := regexp.MustCompile(` \([A-Z]{3}[0-9:+-]*\)$`).ReplaceAll(r[1], []byte(""))
	//r1 = regexp.MustCompile(` {2,}`).ReplaceAll(r1, []byte(" "))

	sendDate := string(r1)
	//fmt.Printf("\nsendDate: %+v\n", sendDate)
	// Standard RFC1123(Z)
	t, err := time.Parse(time.RFC1123, sendDate)
	if err != nil {
		t, err = time.Parse(time.RFC1123Z, sendDate)
	}
	// Go time.Parse is too strict
	// https://groups.google.com/d/msg/golang-nuts/1iLoXTx3qxU/z4VolJPKGQAJ
	// Deal with RFC1123(Z) without leading zero or space
	if err != nil {
		t, err = time.Parse("Mon, 2 Jan 2006 15:04:05 MST", sendDate)
	}
	if err != nil {
		t, err = time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", sendDate)
	}

	if err != nil {
		t = time.Now()
	}
	envelopeDate := t.Format(time.ANSIC)

	_ = validation(okToGo) // confirm okToGo satisfies validation at runtime
	if err = okToGo(rfc822, t); err != nil {
		return err
	}

	_, err = m.Write([]byte(fmt.Sprintf("From %s %s\r\n", envelopeFrom, envelopeDate)))
	if err != nil {
		return err
	}

	// fromEncoded
	_, err = m.Write(regexp.MustCompile(`(\n)(From )`).
		ReplaceAll(rfc822, []byte("$1>$2")))
	if err != nil {
		return err
	}

	_, err = m.Write([]byte("\r\n"))
	if err != nil {
		return err
	}

	return nil
}
