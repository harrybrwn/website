package email

import "testing"

func TestIsEmail(t *testing.T) {
	var valid = []string{
		"sindresorhus@gmail.com",
		"foo@bar",
		"test@about.museum",
		"test@nominet.org.uk",
		"test.test@sindresorhus.com",
		"test@255.255.255.255",
		"a@sindresorhus.com",
		"test@e.com",
		"test@xn--hxajbheg2az3al.xn--jxalpdlp",
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghiklm@sindresorhus.com",
		"test@g--a.com",
		"a@abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghikl.abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghikl.abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghikl.abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefg.hij",
		"123@sindresorhus.com",
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghiklmn@sindresorhus.com",
		"test@iana.co-uk",
		"a@a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v",
		"test@foo-bar.com",
		"foo@x.solutions",
	}
	var invalid = []string{
		"@",
		"@io",
		"@sindresorhus.com",
		"test..sindresorhus.com",
		"test@iana..com",
		"test@sindresorhus.com.",
		"sindre@sindre@sindre.com",
		"mailto:sindresorhus@gmail.com",
		"foo.example.com",
		//
		"!#$%&amp;`*+/=?^`{|}~@sindresorhus.com",
		"\"\\a\"@sindresorhus.com",
		"\"\"@sindresorhus.com",
		"\"test\"@sindresorhus.com",
		"\"\\\"\"@sindresorhus.com",
		"",
	}
	for _, e := range valid {
		if !Valid(e) {
			t.Errorf("expected %q to be a valid email", e)
		}
	}
	for _, e := range invalid {
		if Valid(e) {
			t.Errorf("expected %q not to be a valid email", e)
		}
	}
}
