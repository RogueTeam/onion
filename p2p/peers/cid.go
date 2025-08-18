package peers

import (
	"github.com/ipfs/go-cid"
	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
)

func HashedCidFromData[T string | []byte](data T) cid.Cid {
	bytes := []byte(data)

	mh, _ := multihash.Sum(bytes, multihash.SHA3_224, -1)
	return cid.NewCidV1(uint64(multicodec.DagCbor), mh)
}

func IdentityCidFromData[T string | []byte](data T) cid.Cid {
	bytes := []byte(data)

	mh, _ := multihash.Sum(bytes, multihash.IDENTITY, -1)
	return cid.NewCidV1(uint64(multicodec.DagCbor), mh)
}

func PayloadFromIdentityCid[T string | []byte](c cid.Cid) (v T) {
	r, _ := base58.Decode(c.Hash().B58String())
	var payload []byte
	if len(r) >= 2 {
		payload = r[2:]
	}

	return T(payload)
}
