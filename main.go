package main

import (
	"bytes"
	"fmt"
	"log"
	"net/mail"
	"os"
	"os/exec"
	"strings"
	"time"

	// "github.com/davecgh/go-spew/spew"
	"github.com/mxk/go-imap/imap"
)

var curMsgCount uint32 = 0
var folder = "INBOX"
var user = "#####"
var domain = "@gmail.com"
var imapServer = "imap.gmail.com:993"
var password = "#####"

func main() {
	imap.DefaultLogger = log.New(os.Stdout, "", 0)
	imap.DefaultLogMask = imap.LogConn | imap.LogRaw

	c := Dial(imapServer)
	defer func() {
		ReportOK(c.Unsubscribe(folder))
		ReportOK(c.Logout(30 * time.Second))
	}()

	if c.Caps["STARTTLS"] {
		ReportOK(c.StartTLS(nil))
	}

	if c.Caps["ID"] {
		ReportOK(c.ID("name", "goimap"))
	}
	ReportOK(c.Noop())
	ReportOK(Login(c, user+domain, password))
	if c.Caps["QUOTA"] {
		ReportOK(c.GetQuotaRoot("INBOX"))
	}

	ReportOK(c.Subscribe(folder))
	cmd := ReportOK(c.Select(folder, true))
	for _, rsp := range cmd.Data {
		if rsp.Label == "EXISTS" {
			curMsgCount = rsp.Fields[0].(uint32)
		}
	}
	kickOffWatcher(c)
}

func kickOffWatcher(c *imap.Client) {
	cmd, _ := c.Idle()
	for cmd.InProgress() {
		c.Data = nil
		c.Recv(-1)
		for _, rsp := range c.Data {
			if rsp.Label == "EXISTS" {
				newEmailCount := rsp.Fields[0].(uint32)
				if newEmailCount > curMsgCount {
					c.IdleTerm()
					notifyNewEmail(c, newEmailCount)
					kickOffWatcher(c)
				}
				curMsgCount = newEmailCount
			}
		}
	}
}

func notifyNewEmail(c *imap.Client, mId uint32) {
	set, _ := imap.NewSeqSet("")
	set.AddNum(mId)
	cmd := ReportOK(c.Fetch(set, "FLAGS", "RFC822.HEADER"))
	for _, rsp := range cmd.Data {
		header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
		if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
			subj := msg.Header.Get("Subject")
			from := msg.Header.Get("From")
			formattedFrom := strings.Split(from, " ")[0]
			exec.Command(
				"terminal-notifier",
				"-message", subj,
				"-title", "Mail From: "+formattedFrom+"",
				"-sender", "com.apple.Mail",
			).Output()
		}
	}
}

func Dial(addr string) (c *imap.Client) {
	var err error
	if strings.HasSuffix(addr, ":993") {
		log.Println("Dialing with tls: ", addr)
		c, err = imap.DialTLS(addr, nil)
	} else {
		log.Println("Dialing without tls: ", addr)
		c, err = imap.Dial(addr)
	}
	if err != nil {
		panic(err)
	}
	return c
}

func Login(c *imap.Client, user, pass string) (cmd *imap.Command, err error) {
	defer c.SetLogMask(Sensitive(c, "LOGIN"))
	return c.Login(user, pass)
}

func Sensitive(c *imap.Client, action string) imap.LogMask {
	mask := c.SetLogMask(imap.LogConn)
	hide := imap.LogCmd | imap.LogRaw
	if mask&hide != 0 {
		c.Logln(imap.LogConn, "Raw logging disabled during", action)
	}
	c.SetLogMask(mask &^ hide)
	return mask
}

func ReportOK(cmd *imap.Command, err error) *imap.Command {
	var rsp *imap.Response
	log.Println("err", err)
	if cmd == nil {
		fmt.Printf("--- ??? ---\n%v\n\n", err)
		panic(err)
	} else if err == nil {
		rsp, err = cmd.Result(imap.OK)
	}
	if err != nil {
		fmt.Printf("--- %s ---\n%v\n\n", cmd.Name(true), err)
		panic(err)
	}
	c := cmd.Client()
	fmt.Printf("--- %s ---\n"+
		"%d command response(s), %d unilateral response(s)\n"+
		"%s %s\n\n",
		cmd.Name(true), len(cmd.Data), len(c.Data), rsp.Status, rsp.Info)
	c.Data = nil
	return cmd
}
