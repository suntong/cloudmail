package main

import (
	"fmt"
	"io"
	"regexp"
)

////////////////////////////////////////////////////////////////////////////
// Constant and data type/structure definitions

const envelopeFrom = "cloudmail"

type mbox struct {
	io.Writer
}

////////////////////////////////////////////////////////////////////////////
// Function definitions

func newMbox(w io.Writer) *mbox {
	return &mbox{w}
}

func (m *mbox) writeMessage(rfc822 []byte) error {

	r := regexp.MustCompile(`\nDate: (.*)\r*\n`).FindSubmatch(rfc822)
  //fmt.Printf("%+v\n", r)
	envelopeDate := string(r[1])

	_, err := m.Write([]byte(fmt.Sprintf("From %s %s\r\n", envelopeFrom, envelopeDate)))
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
