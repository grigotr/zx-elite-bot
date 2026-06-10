package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ─────────────────────────────────────────────────────────────────────────────
// DATA STRUCTURES
// ─────────────────────────────────────────────────────────────────────────────

type PlayerState struct {
	Username       string    `json:"username"`
	Balance        int64     `json:"balance"`
	TapLevel       int       `json:"tapLevel"`
	EnergyLevel    int       `json:"energyLevel"`
	LastSync       time.Time `json:"lastSync"`
	LastBalance    int64     `json:"lastBalance"`
	LastCheckin    string    `json:"lastCheckin"`  // "2006-01-02"
	CheckinStreak  int       `json:"checkinStreak"`
	ReferralCode   string    `json:"referralCode"`
	ReferredBy     string    `json:"referredBy"`
	Referrals      []string  `json:"referrals"`
	LastPassive    time.Time `json:"lastPassive"`
	PassiveLevel   int       `json:"passiveLevel"`
	// Anti-cheat rolling window
	SyncHistory    []SyncRecord `json:"syncHistory"`
	TelegramID     int64        `json:"telegramId"`
}

type SyncRecord struct {
	T   time.Time `json:"t"`
	Bal int64     `json:"bal"`
}

type Database struct {
	Players map[string]*PlayerState `json:"players"`
	mu      sync.RWMutex
}

// ─────────────────────────────────────────────────────────────────────────────
// API REQUEST/RESPONSE TYPES
// ─────────────────────────────────────────────────────────────────────────────

type SyncRequest struct {
	Username     string `json:"username"`
	Balance      int64  `json:"balance"`
	TapLevel     int    `json:"tapLevel"`
	EnergyLevel  int    `json:"energyLevel"`
	PassiveLevel int    `json:"passiveLevel"`
	TelegramID   int64  `json:"telegramId"`
}

type SyncResponse struct {
	Status       string `json:"status"`
	Balance      int64  `json:"balance"`
	PassiveEarned int64 `json:"passiveEarned"`
}

type LeaderboardEntry struct {
	Username string `json:"username"`
	Balance  int64  `json:"balance"`
	Rank     int    `json:"rank"`
}

type CheckinResponse struct {
	Success bool   `json:"success"`
	Reward  int64  `json:"reward"`
	Streak  int    `json:"streak"`
	Message string `json:"message"`
}

type ReferralResponse struct {
	Code     string   `json:"code"`
	Referrals []string `json:"referrals"`
	Earnings int64    `json:"earnings"`
}

type ClaimReferralRequest struct {
	Username   string `json:"username"`
	ReferralCode string `json:"referralCode"`
	TelegramID int64  `json:"telegramId"`
}

type PassiveResponse struct {
	Earned    int64 `json:"earned"`
	PerHour   int64 `json:"perHour"`
	Balance   int64 `json:"balance"`
}

// ─────────────────────────────────────────────────────────────────────────────
// GLOBAL STATE
// ─────────────────────────────────────────────────────────────────────────────

var (
	db         *Database
	dbPath     = "database.json"
	saveTimer  *time.Timer
	saveMu     sync.Mutex
	bot        *tgbotapi.BotAPI
)

// ─────────────────────────────────────────────────────────────────────────────
// DATABASE PERSISTENCE
// ─────────────────────────────────────────────────────────────────────────────

func newDatabase() *Database {
	return &Database{
		Players: make(map[string]*PlayerState),
	}
}

func loadDatabase() *Database {
	data, err := os.ReadFile(dbPath)
	if err != nil {
		log.Println("📦 Baza de date nouă creată (nu există fișier anterior).")
		return newDatabase()
	}
	d := newDatabase()
	if err := json.Unmarshal(data, d); err != nil {
		log.Println("⚠️ Eroare la parsarea bazei de date, se resetează:", err)
		return newDatabase()
	}
	log.Printf("✅ Baza de date încărcată: %d jucători.", len(d.Players))
	return d
}

func saveDatabase() {
	db.mu.RLock()
	data, err := json.MarshalIndent(db, "", "  ")
	db.mu.RUnlock()
	if err != nil {
		log.Println("❌ Eroare la serializarea bazei de date:", err)
		return
	}
	tmp := dbPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Println("❌ Eroare la scrierea bazei de date:", err)
		return
	}
	os.Rename(tmp, dbPath)
}

// Schedule a debounced save (max once every 5s)
func scheduleSave() {
	saveMu.Lock()
	defer saveMu.Unlock()
	if saveTimer != nil {
		saveTimer.Stop()
	}
	saveTimer = time.AfterFunc(5*time.Second, saveDatabase)
}

// Periodic full save every 60s as safety net
func startPeriodicSave() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			saveDatabase()
		}
	}()
}

// ─────────────────────────────────────────────────────────────────────────────
// PLAYER HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func generateReferralCode(username string) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := "ZX"
	for i := 0; i < 6; i++ {
		code += string(chars[seed.Intn(len(chars))])
	}
	return code
}

func getOrCreatePlayer(username string, telegramID int64) *PlayerState {
	// Must be called with db.mu.Lock held
	p, ok := db.Players[username]
	if !ok {
		p = &PlayerState{
			Username:    username,
			Balance:     0,
			LastSync:    time.Now(),
			LastBalance: 0,
			LastPassive: time.Now(),
			Referrals:   []string{},
			SyncHistory: []SyncRecord{},
			TelegramID:  telegramID,
		}
		p.ReferralCode = generateReferralCode(username)
		db.Players[username] = p
	}
	if telegramID != 0 {
		p.TelegramID = telegramID
	}
	return p
}

// ─────────────────────────────────────────────────────────────────────────────
// PASSIVE INCOME CALCULATION
// ─────────────────────────────────────────────────────────────────────────────

// PassivePerHour returns ZX/hour for a given passive level
func passivePerHour(passiveLevel int) int64 {
	if passiveLevel <= 0 {
		return 0
	}
	// Level 1 → 100 ZX/h, each level multiplies by 1.8
	return int64(math.Round(100 * math.Pow(1.8, float64(passiveLevel-1))))
}

// computePassiveEarned returns how many ZX to award since lastPassive,
// capped at 8 hours (prevents abuse from long offline periods)
func computePassiveEarned(p *PlayerState) int64 {
	pph := passivePerHour(p.PassiveLevel)
	if pph == 0 {
		return 0
	}
	elapsed := time.Since(p.LastPassive)
	if elapsed > 8*time.Hour {
		elapsed = 8 * time.Hour
	}
	return int64(elapsed.Hours() * float64(pph))
}

// ─────────────────────────────────────────────────────────────────────────────
// ANTI-CHEAT ENGINE
// ─────────────────────────────────────────────────────────────────────────────

// maxLegitimateIncreasePerSecond computes the theoretical maximum ZX/s for
// a player based on their tap level + passive income.
// tapLevel contributes: (1 + tapLevel) ZX per tap × 20 taps/s max
// passive contributes: passivePerHour(passiveLevel) / 3600
func maxLegitimateRate(tapLevel, passiveLevel int) float64 {
	tapsPerSec := float64(1+tapLevel) * 20.0
	passive := float64(passivePerHour(passiveLevel)) / 3600.0
	return tapsPerSec + passive + 50 // +50 ZX/s grace buffer
}

// validateBalanceIncrease checks the claimed balance against the anti-cheat
// engine and returns the validated (possibly reduced) balance.
func validateBalanceIncrease(p *PlayerState, newBalance int64, tapLevel, passiveLevel, energyLevel int) int64 {
	now := time.Now()

	// Update levels from client
	p.TapLevel = tapLevel
	p.EnergyLevel = energyLevel
	p.PassiveLevel = passiveLevel

	// First sync ever
	if p.LastBalance == 0 && newBalance <= 10000 {
		p.Balance = newBalance
		p.LastBalance = newBalance
		p.LastSync = now
		return newBalance
	}

	if newBalance <= p.LastBalance {
		p.Balance = newBalance
		p.LastBalance = newBalance
		p.LastSync = now
		return newBalance
	}

	elapsed := now.Sub(p.LastSync).Seconds()
	if elapsed < 0.5 {
		elapsed = 0.5
	}

	maxRate := maxLegitimateRate(tapLevel, passiveLevel)
	maxAllowed := p.LastBalance + int64(elapsed*maxRate)

	// Rolling window: check last 30s of syncs for suspicious patterns
	// Prune old history
	cutoff := now.Add(-30 * time.Second)
	fresh := p.SyncHistory[:0]
	for _, rec := range p.SyncHistory {
		if rec.T.After(cutoff) {
			fresh = append(fresh, rec)
		}
	}
	p.SyncHistory = fresh

	// Check rolling window rate
	if len(p.SyncHistory) >= 2 {
		oldest := p.SyncHistory[0]
		windowElapsed := now.Sub(oldest.T).Seconds()
		if windowElapsed > 0 {
			windowGain := newBalance - oldest.Bal
			windowRate := float64(windowGain) / windowElapsed
			windowMax := maxRate * 1.2 // allow 20% burst
			if windowRate > windowMax {
				// Clamp to window max
				maxAllowed2 := oldest.Bal + int64(windowElapsed*windowMax)
				if maxAllowed2 < maxAllowed {
					maxAllowed = maxAllowed2
				}
			}
		}
	}

	validated := newBalance
	if validated > maxAllowed {
		validated = maxAllowed
	}

	// Record this sync
	p.SyncHistory = append(p.SyncHistory, SyncRecord{T: now, Bal: validated})
	if len(p.SyncHistory) > 60 {
		p.SyncHistory = p.SyncHistory[len(p.SyncHistory)-60:]
	}

	p.Balance = validated
	p.LastBalance = validated
	p.LastSync = now
	return validated
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Username == "guest" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SyncResponse{Status: "ignored", Balance: req.Balance})
		return
	}

	db.mu.Lock()
	p := getOrCreatePlayer(req.Username, req.TelegramID)

	// Award passive income earned since last sync
	passiveEarned := computePassiveEarned(p)
	p.LastPassive = time.Now()

	// Add passive to the claimed balance before anti-cheat check
	adjustedBalance := req.Balance + passiveEarned

	validBalance := validateBalanceIncrease(p, adjustedBalance, req.TapLevel, req.PassiveLevel, req.EnergyLevel)
	db.mu.Unlock()

	scheduleSave()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SyncResponse{
		Status:        "ok",
		Balance:       validBalance,
		PassiveEarned: passiveEarned,
	})
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	db.mu.RLock()
	defer db.mu.RUnlock()

	var list []LeaderboardEntry
	for user, state := range db.Players {
		if user != "guest" && user != "" {
			list = append(list, LeaderboardEntry{Username: user, Balance: state.Balance})
		}
	}

	sort.Slice(list, func(i, j int) bool { return list[i].Balance > list[j].Balance })

	if len(list) > 10 {
		list = list[:10]
	}
	for i := range list {
		list[i].Rank = i + 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func handleCheckin(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username   string `json:"username"`
		TelegramID int64  `json:"telegramId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Username == "guest" {
		http.Error(w, "Not logged in", http.StatusForbidden)
		return
	}

	today := time.Now().Format("2006-01-02")

	db.mu.Lock()
	p := getOrCreatePlayer(req.Username, req.TelegramID)

	if p.LastCheckin == today {
		db.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CheckinResponse{
			Success: false,
			Message: "Ai deja check-in-ul de azi!",
			Streak:  p.CheckinStreak,
		})
		return
	}

	// Check if yesterday's date matches to maintain streak
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if p.LastCheckin == yesterday {
		p.CheckinStreak++
	} else {
		p.CheckinStreak = 1
	}
	p.LastCheckin = today

	// Rewards: base 500 + 100*streak, capped at 3000 per day
	reward := int64(500 + 100*p.CheckinStreak)
	if reward > 3000 {
		reward = 3000
	}
	// Bonus at streak milestones
	if p.CheckinStreak == 7 {
		reward += 5000
	} else if p.CheckinStreak == 30 {
		reward += 25000
	}

	p.Balance += reward
	p.LastBalance = p.Balance
	streak := p.CheckinStreak
	db.mu.Unlock()

	scheduleSave()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CheckinResponse{
		Success: true,
		Reward:  reward,
		Streak:  streak,
		Message: fmt.Sprintf("🎁 Zi %d consecutivă! +%s ZX", streak, formatInt(reward)),
	})
}

func handleReferral(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("username")
	telegramIDStr := r.URL.Query().Get("telegramId")
	var telegramID int64
	if telegramIDStr != "" {
		telegramID, _ = strconv.ParseInt(telegramIDStr, 10, 64)
	}

	if username == "" || username == "guest" {
		http.Error(w, "Not logged in", http.StatusForbidden)
		return
	}

	db.mu.RLock()
	p := db.Players[username]
	db.mu.RUnlock()

	if p == nil {
		db.mu.Lock()
		p = getOrCreatePlayer(username, telegramID)
		db.mu.Unlock()
		scheduleSave()
	}

	earnings := int64(len(p.Referrals)) * 500

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReferralResponse{
		Code:      p.ReferralCode,
		Referrals: p.Referrals,
		Earnings:  earnings,
	})
}

func handleClaimReferral(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ClaimReferralRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Username == "guest" || req.ReferralCode == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	// Find the owner of this referral code
	var owner *PlayerState
	for _, p := range db.Players {
		if p.ReferralCode == req.ReferralCode {
			owner = p
			break
		}
	}

	if owner == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Cod de referral invalid."})
		return
	}

	if owner.Username == req.Username {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Nu poți folosi propriul cod!"})
		return
	}

	newPlayer := getOrCreatePlayer(req.Username, req.TelegramID)

	if newPlayer.ReferredBy != "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Ai folosit deja un cod de referral."})
		return
	}

	// Check not already in referrals
	for _, ref := range owner.Referrals {
		if ref == req.Username {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Deja înregistrat."})
			return
		}
	}

	// Apply referral
	newPlayer.ReferredBy = owner.Username
	newPlayer.Balance += 1000 // new player bonus
	newPlayer.LastBalance = newPlayer.Balance

	owner.Referrals = append(owner.Referrals, req.Username)
	owner.Balance += 500 // referrer bonus per invite
	owner.LastBalance = owner.Balance

	scheduleSave()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "✅ Cod acceptat! +1000 ZX pentru tine, +500 ZX pentru invitant.",
		"reward":  1000,
	})
}

func handlePassiveInfo(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	username := r.URL.Query().Get("username")
	if username == "" || username == "guest" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PassiveResponse{Earned: 0, PerHour: 0})
		return
	}

	db.mu.RLock()
	p := db.Players[username]
	db.mu.RUnlock()

	if p == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PassiveResponse{Earned: 0, PerHour: 0})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PassiveResponse{
		Earned:  computePassiveEarned(p),
		PerHour: passivePerHour(p.PassiveLevel),
		Balance: p.Balance,
	})
}

func formatInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// TELEGRAM WEBHOOK
// ─────────────────────────────────────────────────────────────────────────────

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if bot == nil {
		http.Error(w, "Bot not initialized", http.StatusInternalServerError)
		return
	}

	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if update.Message == nil {
		return
	}

	webAppURL := os.Getenv("WEBAPP_URL")
	if webAppURL == "" {
		webAppURL = "https://your-app.onrender.com"
	}

	msg := update.Message

	switch msg.Text {
	case "/start":
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonWebApp("🎮 Joacă ZX Network", tgbotapi.WebAppInfo{URL: webAppURL}),
			),
		)
		reply := tgbotapi.NewMessage(msg.Chat.ID,
			"🧬 Bine ai venit în nucleul ZX Network!\n\n"+
				"Apasă butonul de mai jos pentru a accesa aplicația și a genera resurse:\n\n"+
				"⚡ Tap → Earn → Upgrade → Dominate")
		reply.ReplyMarkup = keyboard
		bot.Send(reply)

	case "/referral":
		user := msg.From
		username := user.UserName
		if username == "" {
			username = fmt.Sprintf("id%d", user.ID)
		}
		db.mu.RLock()
		p := db.Players[username]
		db.mu.RUnlock()

		if p == nil {
			db.mu.Lock()
			p = getOrCreatePlayer(username, user.ID)
			db.mu.Unlock()
			scheduleSave()
		}

		botUsername := bot.Self.UserName
		refLink := fmt.Sprintf("https://t.me/%s?start=ref_%s", botUsername, p.ReferralCode)
		text := fmt.Sprintf("🔗 Codul tău de referral: `%s`\n\nLink de invitație:\n%s\n\n👥 Ai invitat %d jucători\n💰 Câștiguri referral: %s ZX",
			p.ReferralCode, refLink, len(p.Referrals), formatInt(int64(len(p.Referrals))*500))
		reply := tgbotapi.NewMessage(msg.Chat.ID, text)
		reply.ParseMode = "Markdown"
		bot.Send(reply)

	case "/balance":
		user := msg.From
		username := user.UserName
		if username == "" {
			username = fmt.Sprintf("id%d", user.ID)
		}
		db.mu.RLock()
		p := db.Players[username]
		db.mu.RUnlock()

		balance := int64(0)
		if p != nil {
			balance = p.Balance
		}
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("💰 Balanța ta: *%s ZX*", formatInt(balance)))
		reply.ParseMode = "Markdown"
		bot.Send(reply)

	case "/leaderboard":
		db.mu.RLock()
		var list []LeaderboardEntry
		for user, state := range db.Players {
			if user != "guest" {
				list = append(list, LeaderboardEntry{Username: user, Balance: state.Balance})
			}
		}
		db.mu.RUnlock()

		sort.Slice(list, func(i, j int) bool { return list[i].Balance > list[j].Balance })
		if len(list) > 5 {
			list = list[:5]
		}

		text := "🏆 *Top 5 ZX Network*\n\n"
		medals := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣"}
		for i, entry := range list {
			text += fmt.Sprintf("%s @%s — %s ZX\n", medals[i], entry.Username, formatInt(entry.Balance))
		}
		reply := tgbotapi.NewMessage(msg.Chat.ID, text)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FRONTEND HTML
// ─────────────────────────────────────────────────────────────────────────────

const webAppHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0,maximum-scale=1.0,user-scalable=no,viewport-fit=cover">
<meta name="mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
<title>ZX Network</title>
<script src="https://telegram.org/js/telegram-web-app.js"></script>
<style>
:root{
  --bg:#070b10;
  --panel:#0d1420;
  --panel2:#111b2b;
  --green:#00f5d4;
  --green2:#00ff87;
  --text:#f3fffc;
  --muted:#87a39d;
  --purple:#9d4dff;
  --orange:#ff9f43;
  --border:rgba(255,255,255,.08);
}
*{ margin:0; padding:0; box-sizing:border-box; -webkit-tap-highlight-color: transparent !important; }
html,body{
  overscroll-behavior: none;
  -webkit-text-size-adjust: none;
  text-size-adjust: none;
}
body{
  background: radial-gradient(circle at top left, rgba(0,245,212,.12), transparent 40%),
              radial-gradient(circle at top right, rgba(0,255,135,.08), transparent 40%),
              #070b10;
  color:var(--text);
  font-family: -apple-system, 'Inter', 'Segoe UI', sans-serif;
  min-height:100vh;
  min-height: -webkit-fill-available;
  overflow-x:hidden;
  -webkit-user-select: none;
  user-select: none;
  touch-action: pan-y;
}
body::before{
  content:"";
  position:fixed;
  inset:0;
  background-image: linear-gradient(rgba(255,255,255,.03) 1px, transparent 1px),
                    linear-gradient(90deg, rgba(255,255,255,.03) 1px, transparent 1px);
  background-size:40px 40px;
  pointer-events:none;
  z-index:0;
}
.app{
  position:relative;
  z-index:1;
  max-width:600px;
  margin:auto;
  padding-bottom:130px;
}
.header{
  position:sticky;
  top:0;
  display:flex;
  justify-content:space-between;
  align-items:center;
  padding:14px 16px;
  backdrop-filter:blur(15px);
  -webkit-backdrop-filter:blur(15px);
  background: rgba(7,11,16,.9);
  border-bottom: 1px solid var(--border);
  z-index:100;
}
.userBox{ display:flex; flex-direction:column; gap:2px; }
.userLabel{ font-size:11px; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px; }
.userName{ font-weight:800; font-size:15px; color:#fff; }
.netBadge{
  background: rgba(0,245,212,.15);
  color:var(--green);
  border: 1px solid rgba(0,245,212,.3);
  border-radius:999px;
  padding: 8px 12px;
  font-size:11px;
  font-weight:800;
  box-shadow: 0 0 18px rgba(0,245,212,.2);
}
.page{ padding:16px; display:flex; flex-direction:column; gap:14px; }
.section{
  background: linear-gradient(180deg, rgba(17,27,43,.96), rgba(13,20,32,.96));
  border-radius:24px;
  border: 1px solid var(--border);
  padding:20px;
}
.balance-container { text-align:center; margin-bottom:8px; margin-top:4px; }
.balanceTitle{ color:var(--muted); font-size:11px; letter-spacing:2px; text-transform:uppercase; }
.balanceValue{
  margin-top:6px;
  font-size:44px;
  font-weight:900;
  color:white;
  text-shadow: 0 0 20px rgba(0,245,212,.5), 0 0 40px rgba(0,245,212,.25);
  letter-spacing:-1px;
  line-height:1;
}
.passive-badge{
  margin-top:6px;
  display:inline-flex;
  align-items:center;
  gap:5px;
  background: rgba(255,159,67,.12);
  border:1px solid rgba(255,159,67,.25);
  color:var(--orange);
  border-radius:999px;
  padding:4px 10px;
  font-size:11px;
  font-weight:700;
}
.coinArea{
  display:flex;
  flex-direction:column;
  align-items:center;
  justify-content:center;
  padding:8px 0;
  position:relative;
}
.coin{
  width:min(240px, 62vw);
  height:min(240px, 62vw);
  cursor:pointer;
  transition: transform .07s cubic-bezier(.25,.46,.45,.94);
  -webkit-user-select: none;
  user-select: none;
  -webkit-tap-highlight-color: transparent !important;
  outline: none;
  touch-action: manipulation;
  display:block;
  will-change: transform;
}
.coin:active{ transform:scale(.91) !important; }
.energyRow{
  margin-top:16px;
  display:flex;
  gap:10px;
  align-items:center;
}
.energyBox{ flex:1; }
.energyLabel{ font-size:11px; color:var(--muted); margin-bottom:5px; display:flex; justify-content:space-between; }
.energyBar{
  width:100%;
  height:12px;
  border-radius:999px;
  overflow:hidden;
  background:#08111b;
  border: 1px solid rgba(255,255,255,.06);
}
.energyFill{
  width:100%;
  height:100%;
  background: linear-gradient(90deg, var(--green), var(--green2));
  box-shadow: 0 0 12px rgba(0,245,212,.3);
  transition: width 0.15s ease;
}
.btn{
  border:none;
  cursor:pointer;
  padding: 11px 16px;
  border-radius:14px;
  font-weight:800;
  font-size:13px;
  background: linear-gradient(135deg, var(--green), var(--green2));
  color:#04120d;
  box-shadow: 0 0 20px rgba(0,245,212,.2);
  transition: filter .15s, transform .1s;
  white-space:nowrap;
  -webkit-tap-highlight-color: transparent !important;
}
.btn:active{ transform:scale(.96); }
.btn:hover{ filter:brightness(1.07); }
.btn:disabled{ opacity:0.45; cursor:not-allowed; pointer-events:none; }
.btn-secondary{
  background: #1b2a3d;
  color:#a0c4c0;
  box-shadow:none;
  border:1px solid rgba(255,255,255,.07);
}
.btn-danger{
  background: linear-gradient(135deg, #c0392b, #e74c3c);
  color:#fff;
  box-shadow: 0 0 20px rgba(231,76,60,.25);
}
.btn-purple{
  background: linear-gradient(135deg, #7c3aed, #9d4dff);
  color:#fff;
  box-shadow: 0 0 20px rgba(157,77,255,.25);
}
.btn-orange{
  background: linear-gradient(135deg, #e67e22, #ff9f43);
  color:#fff;
  box-shadow: 0 0 20px rgba(255,159,67,.25);
}
.upgrades{
  margin-top:18px;
  display:grid;
  grid-template-columns: repeat(2,1fr);
  gap:10px;
}
.upgradeCard{
  background: rgba(0,0,0,.22);
  border: 1px solid rgba(255,255,255,.06);
  border-radius:18px;
  padding:14px;
}
.upgradeTitle{ font-size:13px; font-weight:800; margin-bottom:4px; }
.upgradeSub{ font-size:11px; color:var(--muted); margin-bottom:10px; }
/* Passive upgrade card full width */
.upgradeCardFull{
  background: rgba(0,0,0,.22);
  border: 1px solid rgba(255,159,67,.15);
  border-radius:18px;
  padding:14px;
  margin-top:10px;
}
.bottomNav{
  position:fixed;
  left:0; right:0; bottom:0;
  padding:8px 12px;
  padding-bottom: max(8px, env(safe-area-inset-bottom));
  z-index:999;
}
.bottomInner{
  max-width:600px;
  margin:auto;
  display:grid;
  grid-template-columns: repeat(5,1fr);
  gap:6px;
  background: rgba(8,14,22,.97);
  border: 1px solid rgba(255,255,255,.06);
  border-radius:22px;
  padding:8px;
  backdrop-filter:blur(20px);
  -webkit-backdrop-filter:blur(20px);
}
.tabBtn{
  background:none;
  border:none;
  color:#87a39d;
  padding:8px 4px;
  border-radius:14px;
  cursor:pointer;
  font-weight:800;
  font-size:10px;
  transition:.2s;
  line-height:1.3;
  -webkit-tap-highlight-color: transparent !important;
}
.tabBtn.active{
  color:#04120d;
  background: linear-gradient(135deg, var(--green), var(--green2));
}
.hidden{ display:none !important; }
.floatGain{
  position:fixed;
  color:white;
  font-weight:900;
  font-size:22px;
  pointer-events:none;
  z-index:99999;
  text-shadow: -1px -1px 0 #000, 1px -1px 0 #000, -1px 1px 0 #000, 1px 1px 0 #000,
               0 0 12px rgba(0,255,135,.9);
  animation: floatUp 0.75s ease-out forwards;
  will-change: transform, opacity;
}
@keyframes floatUp{
  from{ opacity:1; transform: translateY(0) scale(1); }
  to{ opacity:0; transform: translateY(-90px) scale(0.75); }
}
.leaderboardItem{
  display:flex;
  align-items:center;
  padding:12px 0;
  border-bottom:1px solid rgba(255,255,255,.05);
  gap:8px;
}
.leaderboardRank{ width:28px; color:var(--green); font-weight:900; font-size:14px; }
.leaderboardName{ flex:1; font-size:13px; }
.leaderboardBalance{ color:#fff; font-weight:700; font-size:13px; }
.taskCard{
  background: rgba(0,0,0,.2);
  border: 1px solid rgba(255,255,255,.06);
  border-radius:18px;
  padding:16px;
  display:flex;
  justify-content:space-between;
  align-items:center;
  gap:12px;
}
.taskInfo{ flex:1; }
.taskTitle{ font-size:14px; font-weight:800; margin-bottom:3px; }
.taskDesc{ font-size:11px; color:var(--muted); }
.checkinGrid{
  display:grid;
  grid-template-columns:repeat(7,1fr);
  gap:6px;
  margin-top:14px;
}
.checkinDay{
  aspect-ratio:1;
  border-radius:10px;
  display:flex;
  flex-direction:column;
  align-items:center;
  justify-content:center;
  font-size:9px;
  font-weight:700;
  background:rgba(0,0,0,.2);
  border:1px solid rgba(255,255,255,.06);
  gap:2px;
}
.checkinDay.done{
  background:rgba(0,245,212,.15);
  border-color:rgba(0,245,212,.3);
  color:var(--green);
}
.checkinDay.today{
  border-color:var(--orange);
  box-shadow:0 0 10px rgba(255,159,67,.3);
}
.referralBox{
  background:rgba(0,0,0,.2);
  border:1px solid rgba(157,77,255,.2);
  border-radius:18px;
  padding:16px;
}
.referralCode{
  font-size:22px;
  font-weight:900;
  color:var(--purple);
  letter-spacing:3px;
  text-align:center;
  padding:12px;
  background:rgba(157,77,255,.1);
  border-radius:12px;
  margin:10px 0;
  text-shadow:0 0 20px rgba(157,77,255,.5);
}
.sectionTitle{
  font-size:16px;
  font-weight:900;
  margin-bottom:16px;
  color:#d9fff5;
}
.stat-row{
  display:flex;
  justify-content:space-between;
  align-items:center;
  padding:8px 0;
  border-bottom:1px solid rgba(255,255,255,.05);
  font-size:13px;
}
.stat-label{ color:var(--muted); }
.stat-value{ font-weight:700; }
.toast{
  position:fixed;
  top:70px;
  left:50%;
  transform:translateX(-50%);
  background:#1a2940;
  border:1px solid rgba(0,245,212,.3);
  color:var(--green);
  padding:10px 20px;
  border-radius:999px;
  font-weight:700;
  font-size:13px;
  z-index:99999;
  pointer-events:none;
  opacity:0;
  transition:opacity .3s;
  white-space:nowrap;
  max-width:90vw;
  text-align:center;
}
.toast.show{ opacity:1; }
</style>
</head>
<body>
<div id="toastEl" class="toast"></div>
<div class="app">

<!-- HEADER -->
<header class="header">
  <div class="userBox">
    <div class="userLabel">User Core</div>
    <div id="telegramUser" class="userName">Guest</div>
  </div>
  <div class="netBadge">⚡ ZX-NET LIVE</div>
</header>

<!-- TABS CONTENT -->
<div class="page">

<!-- ============ GENERATOR TAB ============ -->
<div id="generatorTab" class="section">
  <div class="balance-container">
    <div class="balanceTitle">Total ZX Tokens</div>
    <div id="balanceDisplay" class="balanceValue">0</div>
    <div id="passiveBadge" class="passive-badge hidden">⚙️ <span id="passiveRate">0</span> ZX/oră</div>
  </div>

  <div class="coinArea">
    <svg id="coin" class="coin" viewBox="0 0 500 500" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <radialGradient id="coinGrad" cx="50%" cy="40%" r="55%">
          <stop offset="0%" stop-color="#7affd8"/>
          <stop offset="55%" stop-color="#00ff87"/>
          <stop offset="100%" stop-color="#00a85a"/>
        </radialGradient>
        <filter id="glow" x="-30%" y="-30%" width="160%" height="160%">
          <feGaussianBlur stdDeviation="5" result="blur"/>
          <feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
        <radialGradient id="innerGlow" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="rgba(0,255,135,0.15)"/>
          <stop offset="100%" stop-color="transparent"/>
        </radialGradient>
      </defs>
      <!-- Outer ring decorative -->
      <circle cx="250" cy="250" r="235" fill="none" stroke="rgba(0,255,135,0.08)" stroke-width="2"/>
      <circle cx="250" cy="250" r="218" fill="none" stroke="#b0c4de" stroke-width="6"/>
      <circle cx="250" cy="250" r="205" fill="none" stroke="url(#coinGrad)" stroke-width="4"/>
      <!-- Coin body -->
      <circle cx="250" cy="250" r="190" fill="#05101c" stroke="#00ff87" stroke-width="1.5"/>
      <circle cx="250" cy="250" r="190" fill="url(#innerGlow)"/>
      <!-- Grid decorations -->
      <rect x="130" y="135" width="240" height="230" rx="8" fill="none" stroke="rgba(0,255,135,0.12)" stroke-width="1"/>
      <!-- Circuit lines top -->
      <g stroke="#00ff87" stroke-width="1" opacity="0.35">
        <line x1="155" y1="155" x2="155" y2="180"/><line x1="175" y1="155" x2="175" y2="180"/>
        <line x1="195" y1="155" x2="195" y2="180"/><line x1="215" y1="155" x2="215" y2="180"/>
        <line x1="235" y1="155" x2="235" y2="180"/><line x1="255" y1="155" x2="255" y2="180"/>
        <line x1="275" y1="155" x2="275" y2="180"/><line x1="295" y1="155" x2="295" y2="180"/>
        <line x1="315" y1="155" x2="315" y2="180"/><line x1="335" y1="155" x2="335" y2="180"/>
      </g>
      <!-- Circuit lines bottom -->
      <g stroke="#00ff87" stroke-width="1" opacity="0.35">
        <line x1="155" y1="345" x2="155" y2="320"/><line x1="175" y1="345" x2="175" y2="320"/>
        <line x1="195" y1="345" x2="195" y2="320"/><line x1="215" y1="345" x2="215" y2="320"/>
        <line x1="235" y1="345" x2="235" y2="320"/><line x1="255" y1="345" x2="255" y2="320"/>
        <line x1="275" y1="345" x2="275" y2="320"/><line x1="295" y1="345" x2="295" y2="320"/>
        <line x1="315" y1="345" x2="315" y2="320"/><line x1="335" y1="345" x2="335" y2="320"/>
      </g>
      <!-- Center square -->
      <rect x="228" y="228" width="44" height="44" fill="none" stroke="#00ff87" stroke-width="1.5" rx="4"/>
      <circle cx="250" cy="250" r="14" fill="#00ff87" opacity="0.18"/>
      <!-- ZX letter mark -->
      <g filter="url(#glow)">
        <path d="M 175 185 L 325 185 L 175 315 L 325 315" fill="none" stroke="#ffffff" stroke-width="20" stroke-linecap="round" stroke-linejoin="round"/>
        <path d="M 175 185 L 325 315 M 325 185 L 175 315" fill="none" stroke="#ffffff" stroke-width="20" stroke-linecap="round" stroke-linejoin="round"/>
      </g>
    </svg>
  </div>

  <div class="energyRow">
    <div class="energyBox">
      <div class="energyLabel">
        <span>⚡ Energie</span>
        <span id="energyText">500 / 500</span>
      </div>
      <div class="energyBar"><div id="energyFill" class="energyFill"></div></div>
    </div>
    <button id="rechargeBtn" class="btn btn-secondary">Încarcă</button>
  </div>

  <div class="upgrades">
    <div class="upgradeCard">
      <div class="upgradeTitle">👆 Multitap</div>
      <div class="upgradeSub" id="tapUpgradeSub">Lv.<span id="tapLevelDisplay">0</span> · Cost: <span id="tapCostDisplay">1000</span></div>
      <button id="buyTapUpgrade" class="btn" style="width:100%;">Upgrade</button>
    </div>
    <div class="upgradeCard">
      <div class="upgradeTitle">⚡ Energie Max</div>
      <div class="upgradeSub" id="energyUpgradeSub">Lv.<span id="energyLevelDisplay">0</span> · Cost: <span id="energyCostDisplay">2500</span></div>
      <button id="buyEnergyUpgrade" class="btn" style="width:100%;">Upgrade</button>
    </div>
  </div>

  <!-- Passive Income upgrade full width -->
  <div class="upgradeCardFull">
    <div class="upgradeTitle">⚙️ Passive Mining — Lv.<span id="passiveLevelDisplay">0</span></div>
    <div class="upgradeSub">
      Generează <strong id="passiveRateDisplay">0</strong> ZX/oră automat (max 8h acumulate)<br>
      Cost upgrade: <span id="passiveCostDisplay">5000</span> ZX
    </div>
    <button id="buyPassiveUpgrade" class="btn btn-orange" style="width:100%; margin-top:8px;">Upgrade Passive Mining</button>
  </div>
</div>

<!-- ============ TASKS TAB ============ -->
<div id="tasksTab" class="section hidden">
  <div class="sectionTitle">💼 Tasks & Misiuni</div>

  <!-- Daily check-in card -->
  <div class="referralBox" style="margin-bottom:14px;">
    <div class="upgradeTitle">📅 Daily Check-in</div>
    <div id="checkinStatus" style="font-size:12px; color:var(--muted); margin:6px 0 10px 0;">Streak: <strong id="streakDisplay">0</strong> zile consecutive</div>
    <div class="checkinGrid" id="checkinGrid"></div>
    <button id="checkinBtn" class="btn" style="width:100%;margin-top:14px;">🎁 Revendică Recompensa Zilnică</button>
  </div>

  <!-- Tasks list -->
  <div style="display:flex;flex-direction:column;gap:10px;">
    <div class="taskCard">
      <div class="taskInfo">
        <div class="taskTitle">📺 Vizionează Reclamă</div>
        <div class="taskDesc">+1000 ZX instant</div>
      </div>
      <button id="watchAdBtn" class="btn" style="min-width:80px;">+1000</button>
    </div>
    <div class="taskCard">
      <div class="taskInfo">
        <div class="taskTitle">📢 Abonare Canal</div>
        <div class="taskDesc">Abonează-te la canalul oficial Telegram</div>
      </div>
      <button id="taskTelegram" class="btn" style="min-width:80px;">+500</button>
    </div>
    <div class="taskCard">
      <div class="taskInfo">
        <div class="taskTitle">🤖 Activează Bot Partner</div>
        <div class="taskDesc">Start partener și revendică recompensa</div>
      </div>
      <button id="taskPartner" class="btn" style="min-width:80px;">+2000</button>
    </div>
    <div class="taskCard">
      <div class="taskInfo">
        <div class="taskTitle">🐦 Follow Twitter/X</div>
        <div class="taskDesc">+750 ZX pentru urmărire</div>
      </div>
      <button id="taskTwitter" class="btn" style="min-width:80px;">+750</button>
    </div>
  </div>
</div>

<!-- ============ REFERRAL TAB ============ -->
<div id="referralTab" class="section hidden">
  <div class="sectionTitle">👥 Sistem Referrals</div>

  <div class="referralBox">
    <div style="font-size:12px;color:var(--muted);text-align:center;">Codul tău unic de invitație</div>
    <div id="myReferralCode" class="referralCode">LOADING</div>
    <button id="copyRefCode" class="btn btn-purple" style="width:100%;">📋 Copiază Link de Invitație</button>
  </div>

  <div class="stat-row" style="margin-top:16px;">
    <span class="stat-label">Jucători invitați</span>
    <span class="stat-value" id="refCount">0</span>
  </div>
  <div class="stat-row">
    <span class="stat-label">Câștiguri totale referral</span>
    <span class="stat-value" id="refEarnings">0 ZX</span>
  </div>
  <div class="stat-row">
    <span class="stat-label">Bonus per invitat</span>
    <span class="stat-value" style="color:var(--green);">500 ZX</span>
  </div>

  <!-- Claim referral code input -->
  <div style="margin-top:18px;">
    <div class="sectionTitle" style="font-size:14px;margin-bottom:10px;">Introdu cod de invitație</div>
    <div style="display:flex;gap:8px;">
      <input id="refInput" type="text" placeholder="ex: ZXABCD12"
        style="flex:1;background:rgba(0,0,0,.25);border:1px solid rgba(255,255,255,.1);border-radius:12px;padding:12px 14px;color:white;font-size:14px;font-weight:700;letter-spacing:2px;outline:none;text-transform:uppercase;" maxlength="8"/>
      <button id="claimRefBtn" class="btn btn-purple">Aplică</button>
    </div>
    <div id="refStatus" style="font-size:12px;margin-top:8px;color:var(--muted);"></div>
  </div>

  <div style="margin-top:18px;font-size:12px;color:var(--muted);line-height:1.6;">
    ℹ️ Invită prietenii cu link-ul tău și primești <strong style="color:var(--green);">500 ZX</strong> pentru fiecare jucător nou înregistrat. Ei primesc <strong style="color:var(--green);">1000 ZX</strong> bonus de start.
  </div>
</div>

<!-- ============ WALLET TAB ============ -->
<div id="walletTab" class="section hidden">
  <div class="sectionTitle">👛 Wallet & Retrageri</div>
  <div style="background:rgba(0,0,0,.18);border-radius:18px;padding:16px;border:1px solid rgba(255,255,255,.05);">
    <div style="font-size:12px;color:var(--muted);">ZX Balance</div>
    <div id="walletBalance" style="margin-top:6px;font-size:32px;font-weight:900;">0</div>
    <button id="connectWallet" class="btn" style="margin-top:14px;width:100%;">🔗 Connect TON Wallet</button>
    <div id="walletAddress" style="display:none;margin-top:12px;color:#9fffe9;word-break:break-all;font-size:12px;background:rgba(0,0,0,.2);padding:10px;border-radius:10px;"></div>
  </div>

  <div style="margin-top:16px;">
    <div class="sectionTitle" style="font-size:14px;margin-bottom:12px;">📋 Istoric Retrageri</div>
    <table style="width:100%;border-collapse:collapse;font-size:12px;">
      <thead>
        <tr style="color:var(--muted);">
          <th style="text-align:left;padding:8px 4px;border-bottom:1px solid rgba(255,255,255,.08);">Data</th>
          <th style="border-bottom:1px solid rgba(255,255,255,.08);">ZX</th>
          <th style="border-bottom:1px solid rgba(255,255,255,.08);">Asset</th>
          <th style="border-bottom:1px solid rgba(255,255,255,.08);">Status</th>
        </tr>
      </thead>
      <tbody id="withdrawTable">
        <tr>
          <td style="padding:10px 4px;">2026-06-01</td>
          <td style="text-align:center;">120.000</td>
          <td style="text-align:center;">TON</td>
          <td style="text-align:center;color:#ffd166;">În așteptare</td>
        </tr>
        <tr>
          <td style="padding:10px 4px;">2026-05-24</td>
          <td style="text-align:center;">50.000</td>
          <td style="text-align:center;">TON</td>
          <td style="text-align:center;color:#00ff87;">Finalizat</td>
        </tr>
      </tbody>
    </table>
  </div>

  <div style="margin-top:20px;">
    <button id="deleteAccountBtn" class="btn btn-danger" style="width:100%;">🗑️ Șterge Datele Locale</button>
  </div>
</div>

<!-- ============ RANK TAB ============ -->
<div id="rankTab" class="section hidden">
  <div class="sectionTitle">🏆 Global Leaderboard</div>
  <div id="leaderboard"><div style="color:var(--muted);text-align:center;padding:20px;">Se încarcă...</div></div>
  <div style="margin-top:20px;height:1px;background:linear-gradient(90deg,transparent,#9d4dff,transparent);box-shadow:0 0 15px #9d4dff;"></div>
  <div style="margin-top:16px;background:rgba(157,77,255,.08);border:1px solid rgba(157,77,255,.2);border-radius:18px;padding:16px;">
    <div style="font-size:12px;color:#b79cff;">Poziția ta</div>
    <div id="myRank" style="margin-top:6px;font-size:28px;font-weight:900;">#-</div>
    <div id="myBalanceRank" style="margin-top:4px;color:#e9dbff;font-size:14px;">0 ZX</div>
  </div>
</div>

</div><!-- end .page -->
</div><!-- end .app -->

<!-- RECHARGE MODAL -->
<div id="rechargeModal" style="display:none;position:fixed;inset:0;background:rgba(0,0,0,.85);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);z-index:8000;justify-content:center;align-items:center;">
  <div style="width:90%;max-width:400px;background:#0f1d2e;border-radius:24px;border:1px solid rgba(0,245,212,.2);padding:24px;">
    <h2 style="margin-bottom:10px;">📺 Reîncarcă Energia</h2>
    <p style="color:var(--muted);font-size:13px;">Vizionează 3 reclame pentru a restaura energia complet.</p>
    <div id="adCounter" style="margin-top:14px;font-size:28px;font-weight:900;text-align:center;">0 / 3</div>
    <div style="width:100%;height:8px;background:#08111b;border-radius:999px;margin-top:10px;overflow:hidden;">
      <div id="adProgress" style="height:100%;width:0%;background:linear-gradient(90deg,var(--green),var(--green2));transition:width .4s;"></div>
    </div>
    <button id="watchRechargeAd" class="btn" style="width:100%;margin-top:18px;">▶ Watch Ad</button>
    <button id="closeRecharge" class="btn btn-secondary" style="width:100%;margin-top:10px;">Închide</button>
  </div>
</div>

<div class="bottomNav">
  <div class="bottomInner">
    <button class="tabBtn active" data-tab="generator">🎮<br>Mine</button>
    <button class="tabBtn" data-tab="tasks">💼<br>Tasks</button>
    <button class="tabBtn" data-tab="referral">👥<br>Referral</button>
    <button class="tabBtn" data-tab="wallet">👛<br>Wallet</button>
    <button class="tabBtn" data-tab="rank">🏆<br>Rank</button>
  </div>
</div>

<script>
(function() {
'use strict';

// ─── Telegram init ───
var tg = (window.Telegram && window.Telegram.WebApp) ? window.Telegram.WebApp : null;
var currentUser = { username: 'guest', firstName: 'Guest', id: 0 };

if (tg) {
  tg.ready();
  tg.expand();
  tg.disableVerticalSwipes && tg.disableVerticalSwipes();
  var u = tg.initDataUnsafe ? tg.initDataUnsafe.user : null;
  if (u) {
    currentUser.username = u.username || ('id' + u.id);
    currentUser.firstName = u.first_name || u.username || 'Player';
    currentUser.id = u.id || 0;
  }
}

document.getElementById('telegramUser').textContent = currentUser.firstName;

// ─── State ───
var STORAGE_KEY = 'zxnet-v2';
var raw = {};
try { raw = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}'); } catch(e){}

var state = {
  balance:       raw.balance       !== undefined ? raw.balance : 0,
  energy:        raw.energy        !== undefined ? raw.energy  : 500,
  maxEnergy:     raw.maxEnergy     !== undefined ? raw.maxEnergy : 500,
  tapLevel:      raw.tapLevel      || 0,
  energyLevel:   raw.energyLevel   || 0,
  passiveLevel:  raw.passiveLevel  || 0,
  walletConnected: raw.walletConnected || false,
  walletAddress: raw.walletAddress || '',
  claimedTasks:  raw.claimedTasks  || {},
  checkinStreak: raw.checkinStreak || 0,
  lastCheckin:   raw.lastCheckin   || '',
  referralCode:  raw.referralCode  || '',
  referredByDone: raw.referredByDone || false
};

function save() {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(state)); } catch(e) {}
}

// ─── Helpers ───
function fmt(v) {
  return Number(v).toLocaleString('ro-RO');
}

function toast(msg, dur) {
  var el = document.getElementById('toastEl');
  el.textContent = msg;
  el.classList.add('show');
  clearTimeout(el._t);
  el._t = setTimeout(function(){ el.classList.remove('show'); }, dur || 2500);
}

// ─── Upgrade costs ───
function tapCost()     { return Math.round(1000 * Math.pow(state.tapLevel + 1, 2.2)); }
function energyCost()  { return Math.round(2500 * Math.pow(state.energyLevel + 1, 2.2)); }
function passiveCost() { return Math.round(5000 * Math.pow(state.passiveLevel + 1, 2.5)); }

function passivePerHour(lvl) {
  if (lvl <= 0) return 0;
  return Math.round(100 * Math.pow(1.8, lvl - 1));
}

// ─── UI update ───
function updateUI() {
  document.getElementById('balanceDisplay').textContent = fmt(state.balance);
  document.getElementById('walletBalance').textContent  = fmt(state.balance);

  var pct = state.maxEnergy > 0 ? (state.energy / state.maxEnergy) * 100 : 0;
  document.getElementById('energyFill').style.width = pct + '%';
  document.getElementById('energyText').textContent  = state.energy + ' / ' + state.maxEnergy;

  // Tap upgrade
  document.getElementById('tapLevelDisplay').textContent  = state.tapLevel;
  document.getElementById('tapCostDisplay').textContent   = fmt(tapCost());
  document.getElementById('buyTapUpgrade').disabled       = state.balance < tapCost();

  // Energy upgrade
  document.getElementById('energyLevelDisplay').textContent = state.energyLevel;
  document.getElementById('energyCostDisplay').textContent  = fmt(energyCost());
  document.getElementById('buyEnergyUpgrade').disabled      = state.balance < energyCost();

  // Passive upgrade
  document.getElementById('passiveLevelDisplay').textContent = state.passiveLevel;
  document.getElementById('passiveCostDisplay').textContent  = fmt(passiveCost());
  document.getElementById('passiveRateDisplay').textContent  = fmt(passivePerHour(state.passiveLevel));
  document.getElementById('buyPassiveUpgrade').disabled      = state.balance < passiveCost();

  // Passive badge
  var pph = passivePerHour(state.passiveLevel);
  var badge = document.getElementById('passiveBadge');
  var rateEl = document.getElementById('passiveRate');
  if (pph > 0) {
    badge.classList.remove('hidden');
    rateEl.textContent = fmt(pph);
  } else {
    badge.classList.add('hidden');
  }

  // Checkin streak
  document.getElementById('streakDisplay').textContent = state.checkinStreak;
  updateCheckinGrid();
}

// ─── Check-in grid visual ───
function updateCheckinGrid() {
  var grid = document.getElementById('checkinGrid');
  grid.innerHTML = '';
  var today = new Date().toISOString().split('T')[0];
  for (var i = 1; i <= 7; i++) {
    var day = document.createElement('div');
    day.className = 'checkinDay';
    var reward = Math.min(500 + 100 * i, 3000);
    day.innerHTML = '<span>D' + i + '</span><span style="font-size:8px;color:var(--green);">+' + (reward >= 1000 ? Math.round(reward/1000) + 'k' : reward) + '</span>';
    if (i <= state.checkinStreak) day.classList.add('done');
    if (i === state.checkinStreak + 1) day.classList.add('today');
    grid.appendChild(day);
  }
  // Check-in button state
  var btn = document.getElementById('checkinBtn');
  if (state.lastCheckin === today) {
    btn.textContent = '✅ Revine mâine';
    btn.disabled = true;
  } else {
    btn.textContent = '🎁 Revendică Recompensa Zilnică';
    btn.disabled = false;
  }
}

// ─── Server sync (debounced) ───
var syncTimer = null;
function syncNow(immediate) {
  if (currentUser.username === 'guest') return;
  if (syncTimer) clearTimeout(syncTimer);
  var delay = immediate ? 200 : 1800;
  syncTimer = setTimeout(function() {
    fetch('/api/sync', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username:     currentUser.username,
        balance:      state.balance,
        tapLevel:     state.tapLevel,
        energyLevel:  state.energyLevel,
        passiveLevel: state.passiveLevel,
        telegramId:   currentUser.id
      })
    })
    .then(function(r){ return r.json(); })
    .then(function(d) {
      if (d.balance !== undefined && d.balance !== state.balance) {
        state.balance = d.balance;
        save();
        updateUI();
      }
      if (d.passiveEarned && d.passiveEarned > 0) {
        toast('⚙️ +' + fmt(d.passiveEarned) + ' ZX passive mining', 3000);
      }
    })
    .catch(function(){});
  }, delay);
}

// ─── COIN TAP ───
var coin = document.getElementById('coin');
var lastTapTime = 0;
var tapCount = 0;

function gainTap(clientX, clientY) {
  if (state.energy <= 0) {
    toast('⚡ Energie epuizată! Reîncarcă.', 2000);
    return;
  }
  var now = Date.now();
  // Soft rate limit: max 25 taps/s client-side
  if (now - lastTapTime < 40) return;
  lastTapTime = now;

  var gain = 1 + state.tapLevel;
  state.balance += gain;
  state.energy  = Math.max(0, state.energy - 1);

  spawnFloat(gain, clientX, clientY);
  save();
  updateUI();
  syncNow(false);
}

function spawnFloat(value, x, y) {
  var el = document.createElement('div');
  el.className = 'floatGain';
  el.textContent = '+' + value;
  el.style.left  = x + 'px';
  el.style.top   = (y - 10) + 'px';
  document.body.appendChild(el);
  setTimeout(function(){ if(el.parentNode) el.parentNode.removeChild(el); }, 800);
}

// Touch handler — supports multi-touch (each finger = 1 tap)
coin.addEventListener('touchstart', function(e) {
  e.preventDefault();
  e.stopPropagation();
  for (var i = 0; i < e.changedTouches.length; i++) {
    var t = e.changedTouches[i];
    gainTap(t.clientX, t.clientY);
  }
}, { passive: false });

// Desktop click
coin.addEventListener('mousedown', function(e) {
  if (e.button !== 0) return;
  gainTap(e.clientX, e.clientY);
});

// Prevent context menu on long press
coin.addEventListener('contextmenu', function(e){ e.preventDefault(); });

// ─── Energy regen (1 per 3s) ───
setInterval(function() {
  if (state.energy < state.maxEnergy) {
    state.energy = Math.min(state.maxEnergy, state.energy + 1);
    save();
    updateUI();
  }
}, 3000);

// ─── Tabs ───
var tabIds = ['generator', 'tasks', 'referral', 'wallet', 'rank'];
document.querySelectorAll('.tabBtn').forEach(function(btn) {
  btn.addEventListener('click', function() {
    var tab = btn.dataset.tab;
    document.querySelectorAll('.tabBtn').forEach(function(b){ b.classList.remove('active'); });
    btn.classList.add('active');
    tabIds.forEach(function(id){
      document.getElementById(id + 'Tab').classList.add('hidden');
    });
    document.getElementById(tab + 'Tab').classList.remove('hidden');

    if (tab === 'rank') updateLeaderboardUI();
    if (tab === 'referral') loadReferralInfo();
    syncNow(true);
  });
});

// ─── Tasks ───
function claimTask(id, reward, btn) {
  if (state.claimedTasks[id]) return;
  state.claimedTasks[id] = true;
  state.balance += reward;
  btn.textContent = '✅ Claimed';
  btn.disabled = true;
  save();
  updateUI();
  syncNow(true);
  toast('+' + fmt(reward) + ' ZX câștigat!');
}

document.getElementById('taskTelegram').addEventListener('click', function(){
  claimTask('tg', 500, this);
});
document.getElementById('taskPartner').addEventListener('click', function(){
  claimTask('partner', 2000, this);
});
document.getElementById('taskTwitter').addEventListener('click', function(){
  claimTask('twitter', 750, this);
});
document.getElementById('watchAdBtn').addEventListener('click', function(){
  state.balance += 1000;
  save(); updateUI(); syncNow(true);
  toast('+1000 ZX din reclamă!');
});

// Restore claimed task buttons
(function(){
  var tasks = { tg: {id:'taskTelegram'}, partner: {id:'taskPartner'}, twitter: {id:'taskTwitter'} };
  for (var key in tasks) {
    if (state.claimedTasks[key]) {
      var el = document.getElementById(tasks[key].id);
      if (el) { el.textContent = '✅ Claimed'; el.disabled = true; }
    }
  }
})();

// ─── Daily Check-in ───
document.getElementById('checkinBtn').addEventListener('click', function() {
  if (currentUser.username === 'guest') {
    toast('⚠️ Autentifică-te prin Telegram pentru check-in!');
    return;
  }
  fetch('/api/checkin', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: currentUser.username, telegramId: currentUser.id })
  })
  .then(function(r){ return r.json(); })
  .then(function(d) {
    if (d.success) {
      state.balance += d.reward;
      state.checkinStreak = d.streak;
      state.lastCheckin = new Date().toISOString().split('T')[0];
      save(); updateUI();
      toast('🎁 ' + d.message);
    } else {
      toast(d.message || 'Deja ai făcut check-in azi!');
    }
  })
  .catch(function(){ toast('❌ Eroare conexiune.'); });
});

// ─── Passive income upgrade ───
document.getElementById('buyPassiveUpgrade').addEventListener('click', function() {
  var cost = passiveCost();
  if (state.balance < cost) return;
  state.balance -= cost;
  state.passiveLevel++;
  save(); updateUI(); syncNow(true);
  toast('⚙️ Passive Mining upgraded! Lv.' + state.passiveLevel + ' → ' + fmt(passivePerHour(state.passiveLevel)) + ' ZX/oră');
});

// ─── Tap upgrade ───
document.getElementById('buyTapUpgrade').addEventListener('click', function() {
  var cost = tapCost();
  if (state.balance < cost) return;
  state.balance -= cost;
  state.tapLevel++;
  save(); updateUI(); syncNow(true);
  toast('👆 Multitap upgraded! Lv.' + state.tapLevel + ' — +' + (1 + state.tapLevel) + ' ZX/tap');
});

// ─── Energy upgrade ───
document.getElementById('buyEnergyUpgrade').addEventListener('click', function() {
  var cost = energyCost();
  if (state.balance < cost) return;
  state.balance -= cost;
  state.energyLevel++;
  state.maxEnergy += 500;
  state.energy = state.maxEnergy;
  save(); updateUI(); syncNow(true);
  toast('⚡ Energie extinsă! Max: ' + state.maxEnergy);
});

// ─── Recharge modal ───
var adCount = 0;
document.getElementById('rechargeBtn').addEventListener('click', function() {
  adCount = 0;
  document.getElementById('adCounter').textContent = '0 / 3';
  document.getElementById('adProgress').style.width = '0%';
  document.getElementById('rechargeModal').style.display = 'flex';
});
document.getElementById('watchRechargeAd').addEventListener('click', function() {
  adCount++;
  document.getElementById('adCounter').textContent = adCount + ' / 3';
  document.getElementById('adProgress').style.width = (adCount / 3 * 100) + '%';
  if (adCount >= 3) {
    state.energy = state.maxEnergy;
    save(); updateUI();
    document.getElementById('rechargeModal').style.display = 'none';
    toast('⚡ Energia restaurată complet!');
    syncNow(true);
  }
});
document.getElementById('closeRecharge').addEventListener('click', function(){
  document.getElementById('rechargeModal').style.display = 'none';
});

// ─── Referral ───
function loadReferralInfo() {
  if (currentUser.username === 'guest') {
    document.getElementById('myReferralCode').textContent = 'LOGIN NEEDED';
    return;
  }
  fetch('/api/referral?username=' + encodeURIComponent(currentUser.username) + '&telegramId=' + currentUser.id)
  .then(function(r){ return r.json(); })
  .then(function(d) {
    state.referralCode = d.code || '';
    document.getElementById('myReferralCode').textContent = d.code || '—';
    document.getElementById('refCount').textContent = (d.referrals || []).length;
    document.getElementById('refEarnings').textContent = fmt(d.earnings || 0) + ' ZX';
    save();
  })
  .catch(function(){});
}

document.getElementById('copyRefCode').addEventListener('click', function() {
  if (!state.referralCode) {
    loadReferralInfo();
    toast('Se generează codul...');
    return;
  }
  var botName = 'ZXNetworkBot'; // update with your bot username
  var link = 'https://t.me/' + botName + '?start=ref_' + state.referralCode;
  if (navigator.clipboard) {
    navigator.clipboard.writeText(link).then(function(){
      toast('📋 Link copiat: ' + link);
    });
  } else {
    toast('Codul tău: ' + state.referralCode);
  }
});

document.getElementById('claimRefBtn').addEventListener('click', function() {
  if (currentUser.username === 'guest') {
    toast('⚠️ Autentifică-te prin Telegram!');
    return;
  }
  if (state.referredByDone) {
    toast('Ai folosit deja un cod de referral.');
    return;
  }
  var code = document.getElementById('refInput').value.trim().toUpperCase();
  if (code.length < 8) {
    toast('Cod invalid — minim 8 caractere.');
    return;
  }
  fetch('/api/referral/claim', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: currentUser.username, referralCode: code, telegramId: currentUser.id })
  })
  .then(function(r){ return r.json(); })
  .then(function(d) {
    document.getElementById('refStatus').textContent = d.message || '';
    document.getElementById('refStatus').style.color = d.success ? '#00ff87' : '#ff6b6b';
    if (d.success) {
      state.balance += (d.reward || 1000);
      state.referredByDone = true;
      save(); updateUI(); syncNow(true);
    }
  })
  .catch(function(){ toast('❌ Eroare la server.'); });
});

// ─── Wallet ───
document.getElementById('connectWallet').addEventListener('click', function() {
  if (state.walletConnected) return;
  var chars = 'ABCDEFGHJKLMNPQRSTUVWXYZ0123456789';
  var addr = 'EQB-';
  for (var i = 0; i < 20; i++) addr += chars[Math.floor(Math.random() * chars.length)];
  state.walletConnected = true;
  state.walletAddress = addr;
  var el = document.getElementById('walletAddress');
  el.style.display = 'block';
  el.textContent = addr;
  this.textContent = '✅ Wallet Conectat';
  this.disabled = true;
  save(); toast('✅ Wallet conectat!');
});

if (state.walletConnected) {
  var wa = document.getElementById('walletAddress');
  wa.style.display = 'block';
  wa.textContent = state.walletAddress;
  var cwb = document.getElementById('connectWallet');
  cwb.textContent = '✅ Wallet Conectat';
  cwb.disabled = true;
}

document.getElementById('deleteAccountBtn').addEventListener('click', function(){
  if (!confirm('Ștergi datele locale? Progresul salvat pe server rămâne.')) return;
  localStorage.removeItem(STORAGE_KEY);
  location.reload();
});

// ─── Leaderboard ───
async function updateLeaderboardUI() {
  var board = document.getElementById('leaderboard');
  board.innerHTML = '<div style="color:var(--muted);text-align:center;padding:16px;">Se încarcă...</div>';
  try {
    var resp = await fetch('/api/leaderboard');
    var entries = await resp.json();
    board.innerHTML = '';
    if (!entries || entries.length === 0) {
      board.innerHTML = '<div style="color:var(--muted);text-align:center;padding:20px;">Nu există jucători înregistrați încă.</div>';
      return;
    }
    var medals = ['🥇','🥈','🥉'];
    var myRank = '-';
    entries.forEach(function(entry, i) {
      var isMe = entry.username === currentUser.username;
      if (isMe) myRank = '#' + (i + 1);
      var div = document.createElement('div');
      div.className = 'leaderboardItem';
      div.innerHTML =
        '<span class="leaderboardRank">' + (medals[i] || '#'+(i+1)) + '</span>' +
        '<span class="leaderboardName" style="' + (isMe ? 'color:#00ff87;font-weight:900;' : '') + '">' +
          (isMe ? entry.username + ' ◀' : entry.username) +
        '</span>' +
        '<span class="leaderboardBalance">' + fmt(entry.balance) + ' ZX</span>';
      board.appendChild(div);
    });
    document.getElementById('myRank').textContent = myRank;
    document.getElementById('myBalanceRank').textContent = fmt(state.balance) + ' ZX';
  } catch(e) {
    board.innerHTML = '<div style="color:#ff6b6b;text-align:center;">Eroare la încărcare.</div>';
  }
}

// ─── Init ───
updateUI();

// Sync on load (fetch authoritative balance from server)
if (currentUser.username !== 'guest') {
  setTimeout(function(){ syncNow(true); }, 500);
  // Also do a passive income check on load
  fetch('/api/passive?username=' + encodeURIComponent(currentUser.username))
  .then(function(r){ return r.json(); })
  .then(function(d){
    if (d.earned && d.earned > 0) {
      toast('⚙️ Mining offline: +' + fmt(d.earned) + ' ZX acumulați!', 4000);
    }
  }).catch(function(){});
}

})();
</script>
</body>
</html>`

func main() {
	// Load persisted database
	db = loadDatabase()
	startPeriodicSave()

	// Init Telegram Bot
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		// Fallback for local dev — remove before production or use env var
		token = ""
	}

	if token != "" {
		var err error
		bot, err = tgbotapi.NewBotAPI(token)
		if err != nil {
			log.Println("⚠️ Telegram bot error:", err)
		} else {
			log.Printf("✅ Telegram bot conectat: @%s", bot.Self.UserName)
			// Set webhook
			webhookURL := os.Getenv("WEBHOOK_URL")
			if webhookURL != "" {
				wh, err2 := tgbotapi.NewWebhook(webhookURL + "/webhook")
				if err2 == nil {
					bot.Request(wh)
					log.Println("✅ Webhook setat la:", webhookURL+"/webhook")
				}
			}
		}
	} else {
		log.Println("⚠️ TELEGRAM_TOKEN nu e setat — bot dezactivat.")
	}

	// Routes
	http.HandleFunc("/api/sync",            handleSync)
	http.HandleFunc("/api/leaderboard",     handleLeaderboard)
	http.HandleFunc("/api/checkin",         handleCheckin)
	http.HandleFunc("/api/referral",        handleReferral)
	http.HandleFunc("/api/referral/claim",  handleClaimReferral)
	http.HandleFunc("/api/passive",         handlePassiveInfo)
	http.HandleFunc("/webhook",             handleWebhook)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write([]byte(webAppHTML))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 ZX Network pornit pe portul %s | %d jucători în DB", port, len(db.Players))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
