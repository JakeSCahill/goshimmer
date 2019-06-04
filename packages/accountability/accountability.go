package accountability

import (
    "github.com/dgraph-io/badger"
    "github.com/iotaledger/goshimmer/packages/settings"
    "github.com/iotaledger/goshimmer/packages/identity"
	"sync"
)

var ownId *identity.Identity

var lazyInit sync.Once

func GetOwnId() *identity.Identity {
	lazyInit.Do(initOwnId)

	return ownId
}

func initOwnId() {
	ownId = getIdentity()
}

func generateNewIdentity() *identity.Identity {
    newIdentity := identity.GenerateRandomIdentity()

    if err := settings.Set([]byte("ACCOUNTABILITY_PUBLIC_KEY"), newIdentity.PublicKey); err != nil {
        panic(err)
    }

    if err := settings.Set([]byte("ACCOUNTABILITY_PRIVATE_KEY"), newIdentity.PrivateKey); err != nil {
        panic(err)
    }

    return newIdentity
}

func getIdentity() *identity.Identity {
    publicKey, err := settings.Get([]byte("ACCOUNTABILITY_PUBLIC_KEY"))
    if err != nil {
        if err == badger.ErrKeyNotFound {
            return generateNewIdentity()
        } else {
            panic(err)
        }
    }

    privateKey, err := settings.Get([]byte("ACCOUNTABILITY_PRIVATE_KEY"))
    if err != nil {
        if err == badger.ErrKeyNotFound {
            return generateNewIdentity()
        } else {
            panic(err)
        }
    }

    return identity.NewIdentity(publicKey, privateKey)
}
