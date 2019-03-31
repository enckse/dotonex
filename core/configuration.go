package core

type Configuration struct {
	Debug      bool
	Cache      bool
	Host       string
	Accounting bool
	To         int
	Bind       int
	Dir        string
	NoReject   bool
	Log        string
	Plugins    []string
	Disable    struct {
		Accounting []string
		Preauth    []string
		Trace      []string
		Postauth   []string
	}
	backing []byte
}

func defaultString(given, dflt string) string {
	if len(given) == 0 {
		return dflt
	} else {
		return given
	}
}

func (c *Configuration) Defaults(backing []byte) {
	c.Host = defaultString(c.Host, "localhost")
	c.Dir = defaultString(c.Dir, "/var/lib/radiucal/")
	c.Log = defaultString(c.Log, "/var/log/radiucal/")
	if c.Bind <= 0 {
		if c.Accounting {
			c.Bind = 1813
		} else {
			c.Bind = 1812
		}
	}
	c.backing = backing
}
