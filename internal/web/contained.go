package web

import "mochil/internal/safely"

// contained runs work that no request is waiting on. A panic there would end the whole process, and with it every
// stream and library, so it is logged and ends only that piece of work.
func contained(what string, work func()) { safely.Run(what, work) }
