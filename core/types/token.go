package types

// TokenType = compact enum, maar functioneel hetzelfde als cosmos denom
type TokenType uint8

const (
	TokenGORR  TokenType = iota // native, infinite
	TokenUSDCC                  // second stablecoin
)

// String = UX + RPC + debugging
func (t TokenType) String() string {
	switch t {
	case TokenGORR:
		return "GORR"
	case TokenUSDCC:
		return "USDcc"
	default:
		return "UNKNOWN"
	}
}

// Cosmos-achtige denom
func (t TokenType) Denom() string {
	switch t {
	case TokenGORR:
		return "ugorr"
	case TokenUSDCC:
		return "uusdcc"
	default:
		return "unknown"
	}
}
