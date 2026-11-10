package app

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

// The notch, turned off, shows itself as the first song plays, and only
// then; its button keeps it on.
func TestNotchGreetsOnce(t *testing.T) {
	macOnly(t)
	a := newTestApp()
	n := &notch{a: a, w0: 200, h0: 32, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	a.pl.Cue([]music.Track{{ID: "7", Name: "宝船", Duration: 200000}})
	a.syncWindows()
	if n.hello {
		t.Fatal("greeted before the song played")
	}
	out := audio.NewPlayer(&silence{})
	out.SetVolume(0)
	a.pl.Attach(out)
	defer out.Close()
	a.syncWindows()
	if !n.hello || !a.settings.NotchGreeted {
		t.Fatal("did not greet as the song played")
	}

	tt := ui.NewTester(n.view, int(n.winW), int(n.winH))
	tt.Advance(time.Second)
	tt.Frame()
	if err := tt.Click("Turn On the Notch Player"); err != nil {
		t.Fatal(err)
	}
	if !a.settings.Notch || n.hello {
		t.Fatalf("kept: notch %v, greeting %v", a.settings.Notch, n.hello)
	}

	// Turned off again, it does not greet the next song.
	a.settings.Notch = false
	a.pl.Cue([]music.Track{{ID: "8", Name: "下一首", Duration: 200000}})
	a.pl.Attach(out)
	a.syncWindows()
	if n.hello {
		t.Error("greeted twice")
	}
}

// The pointer resting on the greeting leaves it as it is, so its button can
// be clicked. It used to open the player, which hid the button, and leaving
// brought it back only to open again as the pointer came back to it.
func TestNotchGreetingKeptUnderPointer(t *testing.T) {
	macOnly(t)
	a := newTestApp()
	n := &notch{a: a, real: true, w0: 185, h0: 32, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	a.pl.Cue([]music.Track{{ID: "7", Name: "宝船", Duration: 200000}})
	n.greet()
	if !n.hello {
		t.Fatal("did not greet")
	}

	defer func(f func() mygo.Point) { cursorPoint = f }(cursorPoint)
	cursorPoint = func() mygo.Point {
		return mygo.Point{X: int(n.screenX + n.winW/2 + 120), Y: int(n.screenY + n.h0 + 28)}
	}
	n.poll()
	n.pollAt(cursorPoint(), n.lastInside.Add(time.Second))
	if n.open {
		t.Fatal("opened the player over the greeting")
	}

	tt := ui.NewTester(n.view, int(n.winW), int(n.winH))
	tt.Advance(time.Second)
	tt.Frame()
	if err := tt.Click("Turn On the Notch Player"); err != nil {
		t.Fatal(err)
	}
	if !a.settings.Notch || n.hello {
		t.Fatalf("kept: notch %v, greeting %v", a.settings.Notch, n.hello)
	}
}

// With MM_FRAMES set, saves the notch as it greets, growing and grown.
func TestFirstUseFrames(t *testing.T) {
	dir := os.Getenv("MM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	a := newTestApp()
	n := &notch{a: a, real: true, w0: 185, h0: 32, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	a.pl.Cue([]music.Track{{ID: "7", Name: "NAINAINAI", Artists: []music.Artist{{Name: "ATARASHII GAKKO!"}}, Duration: 193877}})
	main := ui.NewTester(a.view, 1200, 800) // for the app's colors
	main.Frame()
	tt := ui.NewTester(n.view, int(n.winW), int(n.winH))
	tt.Frame()
	n.greet()
	for _, ms := range []int{120, 600} {
		tt.Advance(time.Duration(ms) * time.Millisecond)
		tt.Frame()
		saveFrame(t, dir, fmt.Sprintf("notch-hello-%d.png", ms), tt.Image())
	}
}

// macOnly skips a test of the greeting, which only macOS shows: the notch
// is a Mac's.
func macOnly(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("the notch greets only on macOS")
	}
}

// Hovering only peeks; clicking opens the player, and leaving closes it.
func TestNotchClickToOpen(t *testing.T) {
	a := newTestApp()
	a.settings.Notch = true
	n := &notch{a: a, real: true, w0: 185, h0: 32, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	inside := mygo.Point{X: int(n.winW / 2), Y: 20}
	outside := mygo.Point{X: 900, Y: 400}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n.pollAt(inside, now)
	n.pollAt(inside, now.Add(time.Second))
	if n.open || !n.pop {
		t.Fatal("hover did not stay a preview")
	}
	if _, h := n.target(); h != n.h0+4 {
		t.Fatalf("hover grew to %v, want %v", h, n.h0+4)
	}
	tt := ui.NewTester(n.view, int(n.winW), int(n.winH))
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	if dir := os.Getenv("MM_FRAMES"); dir != "" {
		saveFrame(t, dir, "notch-hover-preview.png", tt.Image())
	}
	if err := tt.Click("Open Notch Player"); err != nil {
		t.Fatal(err)
	}
	if !n.open || n.pop {
		t.Fatal("click did not open the player")
	}
	n.pollAt(inside, now.Add(2*time.Second))
	n.pollAt(outside, now.Add(2100*time.Millisecond))
	if !n.open {
		t.Fatal("closed without a grace period")
	}
	n.pollAt(outside, now.Add(3*time.Second))
	if n.open || n.pop || !n.passing {
		t.Fatal("leaving did not close and pass input through")
	}
	tt.Advance(time.Second)
	tt.Frame()
	if err := tt.Click("Open Notch Player"); err != nil {
		t.Fatal(err)
	}
	if !n.open {
		t.Fatal("could not reopen after closing")
	}
}

// A click over the camera housing opens through the native event monitor.
func TestNotchHousingClick(t *testing.T) {
	a := newTestApp()
	a.settings.Notch = true
	n := &notch{a: a, real: true, w0: 185, h0: 32, winW: notchOpenW + 60, winH: 240}
	defer func(f func() mygo.Point) { cursorPoint = f }(cursorPoint)
	pt := mygo.Point{X: int(n.winW / 2), Y: 10}
	cursorPoint = func() mygo.Point { return pt }
	n.pointerButton(true)
	if !n.open || !n.held || n.passing {
		t.Fatal("housing click did not open and retain input")
	}
	pt = mygo.Point{X: 900, Y: 400}
	n.pollAt(pt, n.lastInside.Add(time.Second))
	if !n.open || n.passing {
		t.Fatal("a held press lost the player outside the window")
	}
	n.pointerButton(false)
	n.pollAt(pt, n.lastInside.Add(time.Second))
	if n.open || n.held || !n.passing {
		t.Fatal("releasing outside did not let the player close")
	}
}

// Playback and liking keep the player open; a seek dragged outside finishes before it closes.
func TestNotchControlsKeepOpen(t *testing.T) {
	a := newTestApp()
	a.settings.Notch = true
	a.account = &music.Account{ID: "1"}
	liked := make(chan bool, 1)
	fake(a).like = func(_ string, on bool) error { liked <- on; return nil }
	a.pl.Cue([]music.Track{song("7", "宝船")})
	a.pl.SetLyrics(nil)
	out := audio.NewPlayer(&silence{})
	out.SetVolume(0)
	a.pl.Attach(out)
	defer out.Close()
	n := &notch{a: a, real: true, open: true, w0: 185, h0: 32, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	tt := ui.NewTester(n.view, int(n.winW), int(n.winH))
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	if dir := os.Getenv("MM_FRAMES"); dir != "" {
		saveFrame(t, dir, "notch-click-expanded.png", tt.Image())
	}
	if err := tt.Click("Play/Pause"); err != nil {
		t.Fatal(err)
	}
	if a.pl.Playing() || !n.open {
		t.Fatal("pausing did not keep the player open")
	}
	if err := tt.Click("Play/Pause"); err != nil {
		t.Fatal(err)
	}
	if !a.pl.Playing() || !n.open {
		t.Fatal("playing did not keep the player open")
	}
	if err := tt.Click("Like"); err != nil {
		t.Fatal(err)
	}
	if !a.liked["7"] || !n.open || !wait(t, liked, "the like") {
		t.Fatal("liking did not keep the player open")
	}
	bar, ok := tt.Find("Progress")
	if !ok {
		t.Fatal("no progress bar")
	}
	x, y := bar.X+bar.W*0.6, bar.Y+bar.H/2
	tt.Press(x, y)
	tt.Frame()
	// The old hover-only close logic could remove the slider mid-drag.
	tt.Move(x, 300)
	tt.Frame()
	outside := mygo.Point{X: int(x), Y: 300}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n.pollAt(outside, now)
	n.pollAt(outside, now.Add(time.Second))
	if !n.open || n.passing || !n.scrub.held {
		t.Fatal("dragging outside lost the player or its input")
	}
	tt.Release(x, 300)
	tt.Frame()
	tt.Frame()
	if pos := a.pl.Position(); pos < 110*time.Second || pos > 130*time.Second {
		t.Fatalf("seek did not finish at 60%%: %v", pos)
	}
	n.pollAt(outside, now.Add(2*time.Second))
	if n.open || !n.passing {
		t.Fatal("player did not close after the drag finished outside")
	}
}

// Without a persistent virtual notch, only this screen's top edge reveals the click target.
func TestNotchVirtualPeek(t *testing.T) {
	a := newTestApp()
	a.settings.Notch = true
	n := &notch{a: a, w0: 200, h0: 28, winW: notchOpenW + 60, winH: 240, screenY: 300}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n.pollAt(mygo.Point{X: int(n.winW / 2), Y: 200}, now)
	if n.pop || !n.passing {
		t.Fatal("a screen above the window triggered the virtual notch")
	}
	n.pollAt(mygo.Point{X: int(n.winW / 2), Y: 300}, now)
	if !n.pop || n.open || n.passing {
		t.Fatal("top edge did not reveal the click target")
	}
	n.pollAt(mygo.Point{X: 900, Y: 400}, now)
	if w, h := n.target(); w != 0 || h != 0 {
		t.Fatal("virtual notch stayed visible after leaving")
	}
}
