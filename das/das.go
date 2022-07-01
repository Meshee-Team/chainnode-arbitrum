// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type DataAvailabilityServiceWriter interface {
	// Requests that the message be stored until timeout (UTC time in unix epoch seconds).
	Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error)
}

type DataAvailabilityService interface {
	arbstate.DataAvailabilityReader
	DataAvailabilityServiceWriter
	fmt.Stringer
}

type DataAvailabilityConfig struct {
	Enable bool `koanf:"enable"`

	RequestTimeout time.Duration `koanf:"request-timeout"`

	LocalCacheConfig BigCacheConfig `koanf:"local-cache"`
	RedisCacheConfig RedisConfig    `koanf:"redis-cache"`

	LocalDBStorageConfig   LocalDBStorageConfig   `koanf:"local-db-storage"`
	LocalFileStorageConfig LocalFileStorageConfig `koanf:"local-file-storage"`
	S3StorageServiceConfig S3StorageServiceConfig `koanf:"s3-storage"`

	KeyConfig KeyConfig `koanf:"key"`

	AggregatorConfig              AggregatorConfig              `koanf:"rpc-aggregator"`
	RestfulClientAggregatorConfig RestfulClientAggregatorConfig `koanf:"rest-aggregator"`

	L1NodeURL             string `koanf:"l1-node-url"`
	SequencerInboxAddress string `koanf:"sequencer-inbox-address"`
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{
	RequestTimeout:                5 * time.Second,
	Enable:                        false,
	RestfulClientAggregatorConfig: DefaultRestfulClientAggregatorConfig,
}

func OptionalAddressFromString(s string) (*common.Address, error) {
	if s == "none" {
		return nil, nil
	}
	if s == "" {
		return nil, errors.New("must provide address for signer or specify 'none'")
	}
	if !common.IsHexAddress(s) {
		return nil, fmt.Errorf("invalid address for signer: %v", s)
	}
	addr := common.HexToAddress(s)
	return &addr, nil
}

func DataAvailabilityConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultDataAvailabilityConfig.Enable, "enable Anytrust Data Availability mode")

	f.Duration(prefix+".request-timeout", DefaultDataAvailabilityConfig.RequestTimeout, "Data Availability Service request timeout duration")

	// Cache options
	BigCacheConfigAddOptions(prefix+".local-cache", f)
	RedisConfigAddOptions(prefix+".redis-cache", f)

	// Storage options
	LocalDBStorageConfigAddOptions(prefix+".local-db-storage", f)
	LocalFileStorageConfigAddOptions(prefix+".local-file-storage", f)
	S3ConfigAddOptions(prefix+".s3-storage", f)

	// Key config for storage
	KeyConfigAddOptions(prefix+".key", f)

	// Aggregator options
	AggregatorConfigAddOptions(prefix+".rpc-aggregator", f)
	RestfulClientAggregatorConfigAddOptions(prefix+".rest-aggregator", f)

	f.String(prefix+".l1-node-url", DefaultDataAvailabilityConfig.L1NodeURL, "URL for L1 node, only used in standalone daserver; when running as part of a node that node's L1 configuration is used")
	f.String(prefix+".sequencer-inbox-address", DefaultDataAvailabilityConfig.SequencerInboxAddress, "L1 address of SequencerInbox contract")
}

func Serialize(c *arbstate.DataAvailabilityCertificate) []byte {

	flags := arbstate.DASMessageHeaderFlag
	if c.Version != 0 {
		flags |= arbstate.TreeDASMessageHeaderFlag
	}

	buf := make([]byte, 0)
	buf = append(buf, flags)
	buf = append(buf, c.KeysetHash[:]...)
	buf = append(buf, c.SerializeSignableFields()...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.SignersMask)
	buf = append(buf, intData[:]...)

	return append(buf, blsSignatures.SignatureToBytes(c.Sig)...)
}
