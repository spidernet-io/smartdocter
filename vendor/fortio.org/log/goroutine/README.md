If you just want a thread ID / goroutine ID use the original this was imported from:

https://github.com/kortschak/goroutine/

Credits goes to Dan Kortschak and Laevus Dexter

# goroutine identification

Package goroutine provides a single function that will return the runtime's
ID number for the calling goroutine.

The implementation is derived from Laevus Dexter's comment in Gophers' Slack
#darkarts, https://gophers.slack.com/archives/C1C1YSQBT/p1593885226448300
post which linked to this playground snippet https://play.golang.org/p/CSOp9wyzydP.

The code here is an exercise in minimalism, doing as little as possible by
deferring nearly all of the logic to runtime functions co-opted via
`//go:linkname` comments.
