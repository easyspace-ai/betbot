package ctf

import "github.com/ethereum/go-ethereum/accounts/abi/bind"

// Backend combines contract and receipt backends needed for transactions.
type Backend interface {
	bind.ContractBackend
	bind.DeployBackend
}
