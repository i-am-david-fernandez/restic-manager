package resticmanager

import (
	"bytes"
	"fmt"
	"html/template"
	"testing"

	"github.com/i-am-david-fernandez/glog"
	"github.com/onsi/gomega"
)

func TestEmailTemplate(t *testing.T) {

	g := gomega.NewGomegaWithT(t)

	const logNameSession = "session"
	sessionBackend := glog.NewListBackend("", glog.Debug)
	glog.SetBackend(logNameSession, sessionBackend)

	glog.Debugf("Debug message")
	glog.Infof("Info message")
	glog.Noticef("Notice message")
	glog.Warningf("Warning message")
	glog.Errorf("Error message")
	glog.Criticalf("Critical message")

	data := struct {
		Preamble   string
		LogSummary []*glog.RecordSummary
		LogRecords []glog.Record
	}{
		"Preamble",
		sessionBackend.Summary(),
		sessionBackend.Get(glog.Debug),
	}

	appConfig := NewAppConfiguration()

	tpl := appConfig.EmailTemplate()

	tplEngine, err := template.New("test").Parse(tpl)
	if err != nil {
		fmt.Printf("%v", err)
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	var buffer bytes.Buffer

	err = tplEngine.Execute(&buffer, data)
	if err != nil {
		fmt.Printf("%v", err)
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	fmt.Println(buffer.String())
}
