func (e *ethRPC) GetBalance(addr common.Address) string {
	bal := e.bc.State.GetBalance(addr)
	return "0x" + bal.Text(16)
}
