package wtypes

// Network represents different Quai networks
type Network string

const (
	Colosseum  Network = "colosseum"
	Garden     Network = "garden"
	Orchard    Network = "orchard"
	Lighthouse Network = "lighthouse"
	Local      Network = "local"
)

// ValidNetworks contains all valid network types
var ValidNetworks = map[Network]bool{
	Colosseum:  true,
	Garden:     true,
	Orchard:    true,
	Lighthouse: true,
	Local:      true,
}
