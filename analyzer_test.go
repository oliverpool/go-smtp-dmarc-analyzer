package analyzer

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// I don't have an example file to publicliy provide here
// func TestGetRead(t *testing.T) {
// 	name := "testdata/fastmail.com.eml"

// 	a := assert.New(t)
// 	f, err := os.Open(name)
// 	if err != nil && os.IsNotExist(err) {
// 		t.SkipNow()
// 	}
// 	a.NoError(err)
// 	defer f.Close()

// 	buf := &bytes.Buffer{}
// 	ses := session{
// 		backend: &Backend{Logger: kitlog.NewLogfmtLogger(buf)},
// 		from:    "dmarc@fastmail.com",
// 	}
// 	err = ses.Data(f)
// 	a.NoError(err)
// 	a.Equal(`from=dmarc@fastmail.com org="Fastmail Pty Ltd" source_ip=64.147.123.24 success=1`, strings.TrimSpace(buf.String()))
// }

func TestFirstAttachment(t *testing.T) {
	cc := []struct {
		desc     string
		input    io.Reader
		filename string
		len      int
		err      bool
	}{
		{
			desc:  "empty",
			input: strings.NewReader(``),
			err:   true,
		},
		{
			desc: "inline file",
			input: strings.NewReader(`MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Type: multipart/mixed; boundary="157370166012.aBE27A1.4034990"
Message-Id: <20191114032100.C102DC6697F@mailuser.nyi.internal>

--157370166012.aBE27A1.4034990
Date: Wed, 13 Nov 2019 22:21:00 -0500
MIME-Version: 1.0
Content-Type: text/plain; charset="US-ASCII"
Content-Disposition: inline


This is a DMARC aggregate report


--157370166012.aBE27A1.4034990
Date: Wed, 13 Nov 2019 22:21:00 -0500
MIME-Version: 1.0
Content-Type: application/gzip;
 name="fastmail.com!example.com!1573603200!1573689599!261425809.xml.gz"
Content-Disposition: inline;
 filename="fastmail.com!example.com!1573603200!1573689599!261425809.xml.gz"
Content-Transfer-Encoding: base64

qBWBldF0XEQAg8BdigPB9x4wImFcduPMz3xMb0YnoJaWHXZFuXsoympbVDucMhNJJ8yocR4leZGw
vLm2KMtfFX5689/pL7rFM3xwBQAA

--157370166012.aBE27A1.4034990--`),
			filename: "fastmail.com!example.com!1573603200!1573689599!261425809.xml.gz",
			len:      78,
		},
		{
			desc: "attachment file",
			input: strings.NewReader(`Content-Type: application/zip;
	name="google.com!example.com!1585958400!1586044799.zip"
Content-Disposition: attachment;
	filename="google.com!example.com!1585958400!1586044799.zip"
Content-Transfer-Encoding: base64

FmzXNvvx7ysvhrDZ7UwP7QXMk/QkPck0Dy/jkDyBdVKr+12eZrsEFNdCqv5+9/PH9329Sx7oXdMB
NTk1ODQwMCExNTg2MDQ0Nzk5LnhtbFBLBQYAAAAAAQABAGYAAABcAgAAAAA=
`),
			filename: "google.com!example.com!1585958400!1586044799.zip",
			len:      101,
		},
	}
	for _, c := range cc {
		t.Run(c.desc, func(t *testing.T) {
			a := assert.New(t)

			filename, r, err := firstAttachment(c.input)
			if c.err {
				a.Error(err)
			} else {
				a.NoError(err)
			}
			a.Equal(c.filename, filename)
			if r == nil {
				a.Equal(0, c.len)
			} else {
				b, err := ioutil.ReadAll(r)
				a.Equal(c.len, len(b))
				a.NoError(err)
			}
		})
	}
}
