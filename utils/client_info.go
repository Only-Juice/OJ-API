package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ClientInfo contains information about the client making the request
type ClientInfo struct {
	IPAddress string
	Location  string
	UserAgent string
	Browser   string
	OS        string
	Country   string
}

// LocationCacheEntry represents a cached location entry
type LocationCacheEntry struct {
	Location  string
	Country   string
	Timestamp time.Time
}

// Global cache for IP location lookups
var (
	locationCache = make(map[string]*LocationCacheEntry)
	cacheMutex    sync.RWMutex
	cacheTimeout  = 24 * time.Hour // Cache for 24 hours
)

// GetClientInfo extracts client information from the request
func GetClientInfo(c *gin.Context) *ClientInfo {
	clientInfo := &ClientInfo{}

	// Get IP Address
	clientInfo.IPAddress = getClientIP(c)

	// Get User Agent
	clientInfo.UserAgent = c.GetHeader("User-Agent")

	// Parse browser and OS from User Agent
	clientInfo.Browser, clientInfo.OS = parseUserAgent(clientInfo.UserAgent)

	// Get location (this would typically require a GeoIP service)
	clientInfo.Location, clientInfo.Country = getLocationFromIP(clientInfo.IPAddress)

	return clientInfo
}

// getClientIP gets the real client IP address
func getClientIP(c *gin.Context) string {
	// Check for X-Forwarded-For header (common in proxy setups)
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, get the first one
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for X-Real-IP header
	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// Check for CF-Connecting-IP (Cloudflare)
	cfConnectingIP := c.GetHeader("CF-Connecting-IP")
	if cfConnectingIP != "" {
		return cfConnectingIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// parseUserAgent extracts browser and OS information from User-Agent string
func parseUserAgent(userAgent string) (browser, os string) {
	userAgent = strings.ToLower(userAgent)

	// Detect Browser
	switch {
	case strings.Contains(userAgent, "edg/"):
		browser = "Microsoft Edge"
	case strings.Contains(userAgent, "chrome/") && !strings.Contains(userAgent, "edg/"):
		browser = "Google Chrome"
	case strings.Contains(userAgent, "firefox/"):
		browser = "Mozilla Firefox"
	case strings.Contains(userAgent, "safari/") && !strings.Contains(userAgent, "chrome/"):
		browser = "Safari"
	case strings.Contains(userAgent, "opera/") || strings.Contains(userAgent, "opr/"):
		browser = "Opera"
	case strings.Contains(userAgent, "msie") || strings.Contains(userAgent, "trident/"):
		browser = "Internet Explorer"
	default:
		browser = "Unknown Browser"
	}

	// Detect Operating System
	switch {
	case strings.Contains(userAgent, "windows nt"):
		if strings.Contains(userAgent, "windows nt 10.0") {
			os = "Windows 10/11"
		} else if strings.Contains(userAgent, "windows nt 6.3") {
			os = "Windows 8.1"
		} else if strings.Contains(userAgent, "windows nt 6.2") {
			os = "Windows 8"
		} else if strings.Contains(userAgent, "windows nt 6.1") {
			os = "Windows 7"
		} else {
			os = "Windows"
		}
	case strings.Contains(userAgent, "mac os x"):
		os = "macOS"
	case strings.Contains(userAgent, "linux"):
		if strings.Contains(userAgent, "ubuntu") {
			os = "Ubuntu Linux"
		} else if strings.Contains(userAgent, "debian") {
			os = "Debian Linux"
		} else if strings.Contains(userAgent, "centos") {
			os = "CentOS Linux"
		} else {
			os = "Linux"
		}
	case strings.Contains(userAgent, "android"):
		os = "Android"
	case strings.Contains(userAgent, "iphone") || strings.Contains(userAgent, "ipad"):
		os = "iOS"
	default:
		os = "Unknown OS"
	}

	return browser, os
}

// IPLocationResponse represents the response from ip-api.com
type IPLocationResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
}

// getLocationFromIP gets location information from IP address
// Uses ip-api.com free service with caching to avoid rate limits
func getLocationFromIP(ip string) (location, country string) {
	// Check for local/private IP addresses
	if isPrivateIP(ip) {
		return "本地網路", "本地"
	}

	// Check cache first
	cacheMutex.RLock()
	if cached, exists := locationCache[ip]; exists {
		// Check if cache entry is still valid
		if time.Since(cached.Timestamp) < cacheTimeout {
			cacheMutex.RUnlock()
			return cached.Location, cached.Country
		}
	}
	cacheMutex.RUnlock()

	// Cache miss or expired, fetch from API
	location, country = fetchLocationFromAPI(ip)

	// Update cache
	cacheMutex.Lock()
	locationCache[ip] = &LocationCacheEntry{
		Location:  location,
		Country:   country,
		Timestamp: time.Now(),
	}
	cacheMutex.Unlock()

	return location, country
}

// fetchLocationFromAPI fetches location data from ip-api.com
func fetchLocationFromAPI(ip string) (location, country string) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Use ip-api.com free service
	// Note: This service has rate limits (45 requests per minute)
	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)

	resp, err := client.Get(url)
	if err != nil {
		Warnf("Failed to get location for IP %s: %v", ip, err)
		return "未知位置", "未知國家"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Warnf("IP location API returned status %d for IP %s", resp.StatusCode, ip)
		return "未知位置", "未知國家"
	}

	var locationResp IPLocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&locationResp); err != nil {
		Warnf("Failed to decode location response for IP %s: %v", ip, err)
		return "未知位置", "未知國家"
	}

	if locationResp.Status != "success" {
		Warnf("IP location lookup failed for IP %s: %s", ip, locationResp.Status)
		return "未知位置", "未知國家"
	}

	// Format location string
	location = formatLocation(locationResp)
	country = locationResp.Country

	return location, country
}

// formatLocation formats the location information into a readable string
func formatLocation(resp IPLocationResponse) string {
	var parts []string

	if resp.City != "" {
		parts = append(parts, resp.City)
	}

	if resp.RegionName != "" && resp.RegionName != resp.City {
		parts = append(parts, resp.RegionName)
	}

	if resp.Country != "" {
		parts = append(parts, resp.Country)
	}

	if len(parts) == 0 {
		return "未知位置"
	}

	return strings.Join(parts, ", ")
}

// CleanExpiredLocationCache removes expired entries from the location cache
func CleanExpiredLocationCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	now := time.Now()
	for ip, entry := range locationCache {
		if now.Sub(entry.Timestamp) > cacheTimeout {
			delete(locationCache, ip)
		}
	}
}

// GetLocationCacheStats returns statistics about the location cache
func GetLocationCacheStats() (total int, expired int) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	now := time.Now()
	total = len(locationCache)
	for _, entry := range locationCache {
		if now.Sub(entry.Timestamp) > cacheTimeout {
			expired++
		}
	}
	return total, expired
}

// isPrivateIP checks if an IP address is private/local
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check for loopback
	if parsedIP.IsLoopback() {
		return true
	}

	// Check for private IP ranges
	privateRanges := []string{
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// FormatClientInfo formats client information for display
func (ci *ClientInfo) FormatClientInfo() string {
	return fmt.Sprintf("IP: %s | Browser: %s | OS: %s | Location: %s, %s",
		ci.IPAddress, ci.Browser, ci.OS, ci.Location, ci.Country)
}

// GetShortClientInfo returns a shortened version of client info
func (ci *ClientInfo) GetShortClientInfo() string {
	return fmt.Sprintf("%s (%s)", ci.IPAddress, ci.Browser)
}
