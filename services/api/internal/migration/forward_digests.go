package migration

import "github.com/clarityit/api/internal/migration/assets"

// WP-01 G1 forward revision identities. 0002-0004 were derived by bounded CI
// probes; 0005 was computed from the exact UTF-8 source bytes and is independently
// asserted against the embedded bytes by the permanent packaging tests.
// These identities are separate from the frozen WP-00 composite.
const (
	Forward0002SHA256 = "257baaa1d3bffe5aa0d435e811d2bfad49f346ed738c6050f46655abae6052f3"
	Forward0003SHA256 = "5e3be99e222515254b0855bd8acab84846c5064b68751a18d0363a06e3e3efeb"
	Forward0004SHA256 = "dca479675e258376af27aae99a7b997d665847081e5c01525f53abecb3da74cf"
	Forward0005SHA256 = "614e14d84464522f2c6e30b2fd2254f61136ed90066ea37399c64f2546e9af15"
)

func init() {
	// Extend the package-wide fail-closed digest registry without modifying any
	// accepted WP-00 digest or composite identity.
	FrozenDigest[assets.AssetForward0002] = Forward0002SHA256
	FrozenDigest[assets.AssetForward0003] = Forward0003SHA256
	FrozenDigest[assets.AssetForward0004] = Forward0004SHA256
	FrozenDigest[assets.AssetForward0005] = Forward0005SHA256
}
