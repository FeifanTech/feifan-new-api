package copilot

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	vsCodeFallbackVersion = "1.104.3"
	vscodeVersionURL      = "https://aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h=visual-studio-code-bin"
)

var (
	cachedVSCodeVersion   string
	vsCodeVersionCacheMu  sync.RWMutex
	vsCodeVersionCachedAt time.Time
	vsCodeVersionCacheTTL = 24 * time.Hour
)

func GetVSCodeVersion() string {
	override := common.GetEnvOrDefaultString("COPILOT_EDITOR_VERSION", "")
	if override != "" {
		return override
	}
	vsCodeVersionCacheMu.RLock()
	if cachedVSCodeVersion != "" && time.Since(vsCodeVersionCachedAt) < vsCodeVersionCacheTTL {
		v := cachedVSCodeVersion
		vsCodeVersionCacheMu.RUnlock()
		return v
	}
	vsCodeVersionCacheMu.RUnlock()

	version := fetchLatestVSCodeVersion()
	vsCodeVersionCacheMu.Lock()
	cachedVSCodeVersion = version
	vsCodeVersionCachedAt = time.Now()
	vsCodeVersionCacheMu.Unlock()
	return version
}

func fetchLatestVSCodeVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", vscodeVersionURL, nil)
	if err != nil {
		return vsCodeFallbackVersion
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return vsCodeFallbackVersion
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return vsCodeFallbackVersion
	}
	re := regexp.MustCompile(`pkgver=([0-9.]+)`)
	if m := re.FindSubmatch(body); m != nil {
		return string(m[1])
	}
	return vsCodeFallbackVersion
}
