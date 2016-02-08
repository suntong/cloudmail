package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

import (
	"github.com/suntong/go-imap"
	"github.com/voxelbrain/goptions"
	"gopkg.in/yaml.v2"
)

////////////////////////////////////////////////////////////////////////////
// Constant and data type/structure definitions

////////////////////////////////////////////////////////////////////////////
// Global variables definitions

var (
	dumpProtocol *bool = flag.Bool("dumpprotocol", false, "dump imap stream")
)

////////////////////////////////////////////////////////////////////////////
// Function definitions

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type progressMessage struct {
	cur, total int
	text       string
}

type UI struct {
	statusChan chan interface{}
	netmon     *netmonReader
}

func (ui *UI) log(format string, args ...interface{}) {
	ui.statusChan <- fmt.Sprintf(format, args...)
}

func (ui *UI) progress(cur, total int, format string, args ...interface{}) {
	message := &progressMessage{cur, total, fmt.Sprintf(format, args...)}
	ui.statusChan <- message
}

func loadAuth(path string) (string, string, string) {
	f, err := os.Open(path)
	check(err)
	r := bufio.NewReader(f)

	imap, isPrefix, err := r.ReadLine()
	check(err)
	if isPrefix {
		panic("prefix")
	}

	user, isPrefix, err := r.ReadLine()
	check(err)
	if isPrefix {
		panic("prefix")
	}

	pass, isPrefix, err := r.ReadLine()
	check(err)
	if isPrefix {
		panic("prefix")
	}

	return string(imap), string(user), string(pass)
}

func readExtra(im *imap.IMAP) {
	for {
		select {
		case msg := <-im.Unsolicited:
			log.Printf("*** unsolicited: %T %+v", msg, msg)
		default:
			return
		}
	}
}

func (ui *UI) connect(useNetmon bool) *imap.IMAP {
	imapAddr, user, pass := loadAuth("auth")

	ui.log("connecting...")
	conn, err := tls.Dial("tcp", imapAddr, nil)
	check(err)

	var r io.Reader = conn
	if *dumpProtocol {
		r = newLoggingReader(r, 300)
	}
	if useNetmon {
		ui.netmon = newNetmonReader(r)
		r = ui.netmon
	}
	im := imap.New(r, conn)
	im.Unsolicited = make(chan interface{}, 100)

	hello, err := im.Start()
	check(err)
	ui.log("server hello: %s", hello)

	ui.log("logging in...")
	resp, caps, err := im.Auth(user, pass)
	check(err)
	ui.log("%s", resp)
	ui.log("server capabilities: %s", caps)

	return im
}

func (ui *UI) fetch(im *imap.IMAP, mailbox string) {
	ui.log("opening %s...", mailbox)
	examine, err := im.Examine(mailbox)
	check(err)
	ui.log("mailbox status: %+v", examine)
	readExtra(im)

	f, err := os.Create(mailbox + ".mbox")
	check(err)
	mbox := newMbox(f)

	query := fmt.Sprintf("1:%d", examine.Exists)
	ui.log("requesting messages %s", query)

	ch, err := im.FetchAsync(query, []string{"RFC822"})
	check(err)

	i := 0
	total := examine.Exists
	ui.progress(i, total, "fetching messages", i, total)
L:
	for {
		r := <-ch
		switch r := r.(type) {
		case *imap.ResponseFetch:
			if err = mbox.writeMessage(r.Rfc822, messageFilter); err != nil &&
				VERBOSITY > 0 {
				ui.log("message ignored: %v", err)
			}
			i++
			ui.progress(i, total, "fetching messages")
		case *imap.ResponseStatus:
			ui.log("complete %v\n", r)
			break L
		}
	}
	readExtra(im)
}

func (ui *UI) reportOnStatus() {

	ticker := time.NewTicker(1000 * 1000 * 1000)
	overprint := false
	status := ""
	overprintLast := false
	for ui.statusChan != nil {
		select {
		case s, stillOpen := <-ui.statusChan:
			switch s := s.(type) {
			case string:
				status = s
				overprint = false
			case *progressMessage:
				status = fmt.Sprintf("%s [%d/%d]", s.text, s.cur, s.total)
				overprint = true
			default:
				if s != nil {
					status = s.(error).Error()
					ui.statusChan = nil
					ticker.Stop()
				}
			}
			if !stillOpen {
				ui.statusChan = nil
				ticker.Stop()
			}
		case <-ticker.C:
			if ui.netmon != nil {
				ui.netmon.Tick()
			}
		}

		if overprintLast {
			fmt.Printf("\r\x1B[K")
		} else {
			fmt.Printf("\n")
		}
		overprintLast = overprint
		fmt.Printf("%s", status)
		if overprint && ui.netmon != nil {
			fmt.Printf(" [%.1fk/s]", ui.netmon.Bandwidth()/1000.0)
		}
	}
	fmt.Printf("\n")
}

func (ui *UI) runFetch(mailbox string) {
	ui.statusChan = make(chan interface{})
	go func() {
		defer func() {
			if e := recover(); e != nil {
				ui.statusChan <- e
			}
		}()
		im := ui.connect(true)
		ui.fetch(im, mailbox)
		close(ui.statusChan)
	}()

	ui.reportOnStatus()
}

func (ui *UI) runList() {
	ui.statusChan = make(chan interface{})
	go func() {
		im := ui.connect(false)
		mailboxes, err := im.List("", imap.WildcardAny)
		check(err)
		fmt.Printf("Available mailboxes:\n")
		for _, mailbox := range mailboxes {
			fmt.Printf("  %s\n", mailbox.Name)
		}
		readExtra(im)
	}()

	ui.reportOnStatus()
}

////////////////////////////////////////////////////////////////////////////
// Main

type Options struct {
	Verbosity []bool        `goptions:"-v, --verbose, description='Be verbose'"`
	Help      goptions.Help `goptions:"-h, --help, description='Show this help\n\nSub-commands (Verbs):\n\tlist\t\tList mailboxes\n\tfetch\t\tDownload mailbox'"`

	goptions.Verbs

	List struct{} `goptions:"list"`

	Fetch struct {
		Folder      string `goptions:"-f, --folder, description='Mail folder to fetch', obligatory"`
		WithinYear  int    `goptions:"--wy, description='Within years, only to fetch mails within this number of years'"`
		WithinMonth int    `goptions:"--wm, description='Within months, ditto for months'"`
		WithinDay   int    `goptions:"--wd, description='Within days, ditto for days'"`
	} `goptions:"fetch"`
}

var options = Options{ // Default values goes here
}

type Command func(Options) error

var commands = map[goptions.Verbs]Command{
	"list":  listCmd,
	"fetch": fetchCmd,
}

var (
	VERBOSITY = 0
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	goptions.ParseAndFail(&options)
	//fmt.Printf("] %#v\r\n", options)

	if len(options.Verbs) == 0 {
		goptions.PrintHelp()
		os.Exit(2)
	}

	VERBOSITY = len(options.Verbosity)

	if cmd, found := commands[options.Verbs]; found {
		err := cmd(options)
		check(err)
	}

}

////////////////////////////////////////////////////////////////////////////
// Dispatch function definitions

func listCmd(options Options) error {
	ui := new(UI)
	ui.runList()
	println("done")
	return nil
}

func fetchCmd(options Options) error {
	ui := new(UI)

	if options.Fetch.WithinYear+options.Fetch.WithinMonth+
		options.Fetch.WithinDay > 0 {
		messageFilterFunc = validation(messageFilter)
		validFrom = time.Now()
		validFrom = validFrom.AddDate(-options.Fetch.WithinYear,
			-options.Fetch.WithinMonth, -options.Fetch.WithinDay)
	}

	ui.runFetch(options.Fetch.Folder)
	return nil
}

////////////////////////////////////////////////////////////////////////////
// message filtering

var (
	messageFilterFunc = validation(emptyFilter)
	validFrom         time.Time
)

func messageFilter(rfc822 []byte, envelopeDate time.Time) error {
	if envelopeDate.Before(validFrom) {
		return errors.New("older than picked date")
	}
	return nil
}
