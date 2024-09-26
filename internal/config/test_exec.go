package config

type TestExec struct {
	*API
}

type TestExecCancel struct {
	*TestExec
}

type TestExecGet struct {
	*TestExec
}

type TestExecList struct {
	*TestExec
}
