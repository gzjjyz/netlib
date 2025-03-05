package netlib

type Agent interface {
	Run()
	OnClose()
}
