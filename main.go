package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ═══════════════════════════════════════════════════════════════════════════════
// CONFIG
// ═══════════════════════════════════════════════════════════════════════════════

const (
	TG_CHANNEL       = "@ZXchatofficial"
	ADSGRAM_BLOCK_ID = "34749"
	APP_URL          = "https://zx-elite-core.onrender.com"

	REWARD_WATCH_AD     int64 = 1000
	REWARD_JOIN_CHANNEL int64 = 10000
	REWARD_CHANNEL2     int64 = 1500
	REWARD_TWITTER      int64 = 5000
	REWARD_PARTNER      int64 = 20000
	REWARD_DAILY_COMBO  int64 = 5000000

	LINK_CHANNEL  = "https://t.me/Swordstarsibot?start=_tgr_6ZeW5DBkNTli"
	LINK_CHANNEL2 = "https://t.me/CandyAIOfficialbot?start=_tgr_92TSa084ODcy"
	LINK_TWITTER  = "https://t.me/StarsiFotBot?start=_tgr_KvKAi-5hZDQy"
	LINK_PARTNER  = "https://t.me/wen_Lambo_1212bot?start=_tgr_fBveixRhNzQy"
	TG_CHANNEL2   = ""

	MAX_TAP_LEVEL     = 20
	MAX_ENERGY_LEVEL  = 20
	MAX_PASSIVE_LEVEL = 20

	// Staking APY rates
	STAKE_7D_APY  = 0.08  // 8%
	STAKE_30D_APY = 0.20  // 20%
	STAKE_90D_APY = 0.50  // 50%
)

// Daily combo — poate fi setat din env sau hardcodat zilnic
func getDailyComboCards() []string {
	v := os.Getenv("DAILY_COMBO_CARDS")
	if v != "" {
		return strings.Split(v, ",")
	}
	return []string{"pr_viral", "web3_dao", "legal_compliance"}
}
func getDailyCipherAnswer() string {
	v := os.Getenv("DAILY_CIPHER")
	if v != "" {
		return strings.ToUpper(v)
	}
	return "ZXNET"
}

// ═══════════════════════════════════════════════════════════════════════════════
// RANK SYSTEM
// ═══════════════════════════════════════════════════════════════════════════════

type RankInfo struct {
	Level int
	Name  string
	Min   int64
}

var ranks = []RankInfo{
	{1, "Bronze", 0},
	{2, "Silver", 50_000},
	{3, "Gold", 250_000},
	{4, "Platinum", 1_000_000},
	{5, "Diamond", 5_000_000},
	{6, "Elite", 20_000_000},
	{7, "Master", 75_000_000},
	{8, "GrandMaster", 200_000_000},
	{9, "Legend", 500_000_000},
	{10, "Grandmaster+", 1_000_000_000},
}

func getRank(balance int64) RankInfo {
	r := ranks[0]
	for _, rk := range ranks {
		if balance >= rk.Min {
			r = rk
		}
	}
	return r
}

// ═══════════════════════════════════════════════════════════════════════════════
// CARD CATALOG
// ═══════════════════════════════════════════════════════════════════════════════

type CardDef struct {
	ID          string
	Category    string
	Name        string
	Emoji       string
	BasePrice   int64
	PriceScale  float64
	BasePPH     int64
	PPHScale    float64
	MaxLevel    int
}

var cardCatalog = []CardDef{
	// PR & Marketing
	{"pr_viral",       "PR",        "Viral Campaign",       "📢", 5_000,    2.0, 200,  1.5, 20},
	{"pr_influencer",  "PR",        "Influencer Deal",      "🌟", 12_000,   2.1, 500,  1.6, 20},
	{"pr_press",       "PR",        "Press Release",        "📰", 8_000,    2.0, 350,  1.5, 20},
	{"mkt_ads",        "Marketing", "Ad Campaign",          "📊", 10_000,   2.1, 400,  1.5, 20},
	{"mkt_seo",        "Marketing", "SEO Boost",            "🔍", 15_000,   2.2, 700,  1.6, 20},
	{"mkt_community",  "Marketing", "Community Manager",    "👥", 7_000,    2.0, 280,  1.5, 20},
	// Web3
	{"web3_dao",       "Web3",      "DAO Integration",      "🏛️", 25_000,   2.3, 1_200, 1.7, 20},
	{"web3_nft",       "Web3",      "NFT Collection",       "🎨", 50_000,   2.4, 2_500, 1.8, 20},
	{"web3_defi",      "Web3",      "DeFi Protocol",        "💱", 80_000,   2.5, 4_000, 1.9, 20},
	{"web3_bridge",    "Web3",      "Chain Bridge",         "🌉", 120_000,  2.6, 6_000, 2.0, 20},
	// Legal
	{"legal_kycaml",   "Legal",     "KYC/AML System",       "🔏", 30_000,   2.2, 1_500, 1.7, 20},
	{"legal_compliance","Legal",    "Compliance Officer",   "⚖️", 45_000,   2.3, 2_200, 1.8, 20},
	{"legal_audit",    "Legal",     "Security Audit",       "🛡️", 60_000,   2.4, 3_000, 1.9, 20},
	// Team
	{"team_dev",       "Team",      "Lead Developer",       "💻", 20_000,   2.1, 900,  1.6, 20},
	{"team_cto",       "Team",      "CTO Hire",             "🧠", 100_000,  2.5, 5_000, 2.0, 20},
	{"team_cmo",       "Team",      "CMO Hire",             "📈", 90_000,   2.5, 4_500, 1.9, 20},
}

func getCardDef(id string) *CardDef {
	for i := range cardCatalog {
		if cardCatalog[i].ID == id {
			return &cardCatalog[i]
		}
	}
	return nil
}

func cardPrice(def *CardDef, level int) int64 {
	if level >= def.MaxLevel {
		return math.MaxInt64
	}
	return int64(float64(def.BasePrice) * math.Pow(def.PriceScale, float64(level)))
}

func cardPPH(def *CardDef, level int) int64 {
	if level <= 0 {
		return 0
	}
	return int64(float64(def.BasePPH) * math.Pow(def.PPHScale, float64(level-1)))
}

// ═══════════════════════════════════════════════════════════════════════════════
// RATE LIMITER
// ═══════════════════════════════════════════════════════════════════════════════

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
}

func newRateLimiter() *RateLimiter {
	rl := &RateLimiter{buckets: make(map[string][]time.Time)}
	go func() {
		for range time.NewTicker(30 * time.Second).C {
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) Allow(key string, maxPerSec int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-time.Second)
	prev := rl.buckets[key]
	fresh := prev[:0]
	for _, t := range prev {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	rl.buckets[key] = fresh
	if len(fresh) >= maxPerSec {
		return false
	}
	rl.buckets[key] = append(rl.buckets[key], now)
	return true
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-5 * time.Second)
	for k, times := range rl.buckets {
		fresh := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				fresh = append(fresh, t)
			}
		}
		if len(fresh) == 0 {
			delete(rl.buckets, k)
		} else {
			rl.buckets[k] = fresh
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// DATABASE LAYER (PostgreSQL)
// ═══════════════════════════════════════════════════════════════════════════════

var db *sql.DB

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("❌ DATABASE_URL env var lipsă!")
	}
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("❌ DB open:", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err = db.Ping(); err != nil {
		log.Fatal("❌ DB ping:", err)
	}
	migrateDB()
	log.Println("✅ PostgreSQL conectat și migrat.")
}

func migrateDB() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS players (
			telegram_id        BIGINT PRIMARY KEY,
			username           TEXT NOT NULL DEFAULT '',
			first_name         TEXT NOT NULL DEFAULT '',
			photo_url          TEXT NOT NULL DEFAULT '',
			balance            BIGINT NOT NULL DEFAULT 0,
			tap_level          INT NOT NULL DEFAULT 0,
			energy_level       INT NOT NULL DEFAULT 0,
			passive_level      INT NOT NULL DEFAULT 0,
			last_sync          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_passive       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			checkin_streak     INT NOT NULL DEFAULT 0,
			last_checkin       TEXT NOT NULL DEFAULT '',
			referred_by        BIGINT NOT NULL DEFAULT 0,
			wallet_address     TEXT NOT NULL DEFAULT '',
			bonus_claimed      BOOLEAN NOT NULL DEFAULT FALSE,
			level              INT NOT NULL DEFAULT 1,
			points_per_hour    BIGINT NOT NULL DEFAULT 0,
			spin_cooldown      TIMESTAMPTZ,
			daily_combo_claimed TEXT NOT NULL DEFAULT '',
			referral_code      TEXT NOT NULL DEFAULT '',
			is_pro             BOOLEAN NOT NULL DEFAULT FALSE,
			pro_expires        TIMESTAMPTZ,
			last_balance       BIGINT NOT NULL DEFAULT 0,
			created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS player_cards (
			telegram_id BIGINT NOT NULL REFERENCES players(telegram_id) ON DELETE CASCADE,
			card_id     TEXT NOT NULL,
			level       INT NOT NULL DEFAULT 1,
			bought_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (telegram_id, card_id)
		)`,
		`CREATE TABLE IF NOT EXISTS player_tasks (
			telegram_id BIGINT NOT NULL REFERENCES players(telegram_id) ON DELETE CASCADE,
			task_id     TEXT NOT NULL,
			claimed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (telegram_id, task_id)
		)`,
		`CREATE TABLE IF NOT EXISTS referrals (
			referrer_id BIGINT NOT NULL REFERENCES players(telegram_id) ON DELETE CASCADE,
			referee_id  BIGINT NOT NULL REFERENCES players(telegram_id) ON DELETE CASCADE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (referee_id)
		)`,
		`CREATE TABLE IF NOT EXISTS stakes (
			id          SERIAL PRIMARY KEY,
			telegram_id BIGINT NOT NULL REFERENCES players(telegram_id) ON DELETE CASCADE,
			amount      BIGINT NOT NULL,
			duration_days INT NOT NULL,
			apy         FLOAT NOT NULL,
			started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			unlocks_at  TIMESTAMPTZ NOT NULL,
			claimed     BOOLEAN NOT NULL DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS sync_history (
			id          SERIAL PRIMARY KEY,
			telegram_id BIGINT NOT NULL,
			balance     BIGINT NOT NULL,
			ts          TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_players_balance ON players(balance DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_history_tid ON sync_history(telegram_id, ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_stakes_player ON stakes(telegram_id, claimed)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			log.Fatalf("migrate error:\n%s\n%v", s, err)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// PLAYER MODEL
// ═══════════════════════════════════════════════════════════════════════════════

type Player struct {
	TelegramID       int64
	Username         string
	FirstName        string
	PhotoURL         string
	Balance          int64
	TapLevel         int
	EnergyLevel      int
	PassiveLevel     int
	LastSync         time.Time
	LastPassive      time.Time
	CheckinStreak    int
	LastCheckin      string
	ReferredBy       int64
	WalletAddress    string
	BonusClaimed     bool
	Level            int
	PointsPerHour    int64
	SpinCooldown     *time.Time
	DailyComboClaimed string
	ReferralCode     string
	IsPro            bool
	ProExpires       *time.Time
	LastBalance      int64
}

type PlayerCard struct {
	TelegramID int64
	CardID     string
	Level      int
}

func getOrCreatePlayer(telegramID int64, username, firstName, photoURL string) (*Player, error) {
	p := &Player{}
	err := db.QueryRow(`
		SELECT telegram_id, username, first_name, photo_url, balance, tap_level, energy_level,
		       passive_level, last_sync, last_passive, checkin_streak, last_checkin,
		       referred_by, wallet_address, bonus_claimed, level, points_per_hour,
		       spin_cooldown, daily_combo_claimed, referral_code, is_pro, pro_expires,
		       last_balance
		FROM players WHERE telegram_id = $1
	`, telegramID).Scan(
		&p.TelegramID, &p.Username, &p.FirstName, &p.PhotoURL,
		&p.Balance, &p.TapLevel, &p.EnergyLevel, &p.PassiveLevel,
		&p.LastSync, &p.LastPassive, &p.CheckinStreak, &p.LastCheckin,
		&p.ReferredBy, &p.WalletAddress, &p.BonusClaimed,
		&p.Level, &p.PointsPerHour, &p.SpinCooldown, &p.DailyComboClaimed,
		&p.ReferralCode, &p.IsPro, &p.ProExpires, &p.LastBalance,
	)
	if err == sql.ErrNoRows {
		code := generateReferralCode(telegramID)
		_, err2 := db.Exec(`
			INSERT INTO players (telegram_id, username, first_name, photo_url, referral_code)
			VALUES ($1, $2, $3, $4, $5)
		`, telegramID, username, firstName, photoURL, code)
		if err2 != nil {
			return nil, fmt.Errorf("insert player: %w", err2)
		}
		return getOrCreatePlayer(telegramID, username, firstName, photoURL)
	}
	if err != nil {
		return nil, err
	}
	// Update meta if changed
	if username != "" && username != p.Username ||
		firstName != "" && firstName != p.FirstName ||
		photoURL != "" && photoURL != p.PhotoURL {
		db.Exec(`UPDATE players SET username=$1, first_name=$2, photo_url=$3 WHERE telegram_id=$4`,
			username, firstName, photoURL, telegramID)
		p.Username = username
		p.FirstName = firstName
		p.PhotoURL = photoURL
	}
	return p, nil
}

func savePlayer(p *Player) error {
	rank := getRank(p.Balance)
	p.Level = rank.Level
	_, err := db.Exec(`
		UPDATE players SET
			username=$1, first_name=$2, photo_url=$3, balance=$4,
			tap_level=$5, energy_level=$6, passive_level=$7,
			last_sync=$8, last_passive=$9, checkin_streak=$10, last_checkin=$11,
			wallet_address=$12, level=$13, points_per_hour=$14,
			spin_cooldown=$15, daily_combo_claimed=$16, is_pro=$17, pro_expires=$18,
			last_balance=$19
		WHERE telegram_id=$20
	`,
		p.Username, p.FirstName, p.PhotoURL, p.Balance,
		p.TapLevel, p.EnergyLevel, p.PassiveLevel,
		p.LastSync, p.LastPassive, p.CheckinStreak, p.LastCheckin,
		p.WalletAddress, p.Level, p.PointsPerHour,
		p.SpinCooldown, p.DailyComboClaimed, p.IsPro, p.ProExpires,
		p.LastBalance,
		p.TelegramID,
	)
	return err
}

func getPlayerCards(telegramID int64) ([]PlayerCard, error) {
	rows, err := db.Query(`SELECT card_id, level FROM player_cards WHERE telegram_id=$1`, telegramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []PlayerCard
	for rows.Next() {
		var c PlayerCard
		c.TelegramID = telegramID
		if err := rows.Scan(&c.CardID, &c.Level); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}

func computeTotalPPH(cards []PlayerCard) int64 {
	var total int64
	for _, c := range cards {
		def := getCardDef(c.CardID)
		if def == nil {
			continue
		}
		total += cardPPH(def, c.Level)
	}
	return total
}

func hasClaimedTask(telegramID int64, taskID string) bool {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM player_tasks WHERE telegram_id=$1 AND task_id=$2`, telegramID, taskID).Scan(&n)
	return n > 0
}

func claimTaskDB(telegramID int64, taskID string) {
	db.Exec(`INSERT INTO player_tasks (telegram_id, task_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, telegramID, taskID)
}

func generateReferralCode(telegramID int64) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	rng := rand.New(rand.NewSource(telegramID + time.Now().UnixNano()))
	code := "ZX"
	for i := 0; i < 6; i++ {
		code += string(chars[rng.Intn(len(chars))])
	}
	return code
}

// ═══════════════════════════════════════════════════════════════════════════════
// TELEGRAM INIT DATA VALIDATION (HMAC-SHA256)
// ═══════════════════════════════════════════════════════════════════════════════

func validateTelegramInitData(initData string) (map[string]string, bool) {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		// Dev mode: skip validation
		return nil, true
	}
	parsed, err := url.ParseQuery(initData)
	if err != nil {
		return nil, false
	}
	receivedHash := parsed.Get("hash")
	if receivedHash == "" {
		return nil, false
	}

	// Build data-check-string: all params except hash, sorted alphabetically
	var parts []string
	for k, v := range parsed {
		if k == "hash" {
			continue
		}
		parts = append(parts, k+"="+v[0])
	}
	sort.Strings(parts)
	dataCheckString := strings.Join(parts, "\n")

	// secret = HMAC-SHA256("WebAppData", bot_token)
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(token))
	secret := mac.Sum(nil)

	// signature = HMAC-SHA256(secret, data_check_string)
	mac2 := hmac.New(sha256.New, secret)
	mac2.Write([]byte(dataCheckString))
	expected := hex.EncodeToString(mac2.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(receivedHash)) {
		return nil, false
	}

	// Parse auth_date — reject if older than 24h
	if adStr, ok := parsed["auth_date"]; ok && len(adStr) > 0 {
		authDate, err := strconv.ParseInt(adStr[0], 10, 64)
		if err == nil {
			age := time.Now().Unix() - authDate
			if age > 86400 {
				return nil, false
			}
		}
	}

	result := make(map[string]string)
	for k, v := range parsed {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result, true
}

// extractTelegramID reads telegramId from the request JSON body (already parsed)
// and validates the initData header if present.
// Returns (telegramID, ok).
func validateRequest(r *http.Request, telegramID int64) bool {
	initData := r.Header.Get("X-Telegram-Init-Data")
	if initData == "" {
		// No initData header — allow only in dev (no token set)
		return os.Getenv("TELEGRAM_TOKEN") == ""
	}
	_, ok := validateTelegramInitData(initData)
	return ok
}

// ═══════════════════════════════════════════════════════════════════════════════
// PASSIVE INCOME
// ═══════════════════════════════════════════════════════════════════════════════

func passivePerHour(p *Player) int64 {
	base := p.PointsPerHour
	lvl := p.PassiveLevel
	if lvl > 0 {
		base += int64(math.Round(100 * math.Pow(1.8, float64(lvl-1))))
	}
	if p.IsPro {
		base *= 2
	}
	return base
}

func computePassiveEarned(p *Player) int64 {
	pph := passivePerHour(p)
	if pph == 0 {
		return 0
	}
	elapsed := time.Since(p.LastPassive)
	if elapsed > 8*time.Hour {
		elapsed = 8 * time.Hour
	}
	return int64(elapsed.Hours() * float64(pph))
}

// ═══════════════════════════════════════════════════════════════════════════════
// ANTI-CHEAT — SERVER-SIDE TAP PROCESSING
// ═══════════════════════════════════════════════════════════════════════════════

const (
	MAX_TAPS_PER_SEC = 25
	MAX_ENERGY_BASE  = 500
)

func calcMaxEnergy(energyLevel int) int64 {
	return int64(MAX_ENERGY_BASE + energyLevel*500)
}

// processServerTaps verifies tap count is physically possible and updates player.
// Returns (newBalance, energyLeft, error)
func processServerTaps(p *Player, taps int64, clientBalance int64) (int64, int64, error) {
	if taps < 0 || taps > 10000 {
		return p.Balance, 0, fmt.Errorf("tap count out of range")
	}

	// Verify elapsed time allows this many taps
	now := time.Now()
	elapsed := now.Sub(p.LastSync).Seconds()
	if elapsed < 0.1 {
		elapsed = 0.1
	}
	maxTaps := int64(elapsed * MAX_TAPS_PER_SEC)
	if maxTaps < 1 {
		maxTaps = 1
	}
	if taps > maxTaps {
		taps = maxTaps
	}

	// Energy check
	maxEnergy := calcMaxEnergy(p.EnergyLevel)
	// Reconstruct current energy (server doesn't store energy, derive from last sync)
	var energyCurrent int64
	db.QueryRow(`
		SELECT COALESCE(
			(SELECT $1 - SUM(taps_used) FROM (
				SELECT 0 AS taps_used -- placeholder; real impl tracks energy in DB
			) t), $1
		)`, maxEnergy).Scan(&energyCurrent)
	// Simpler: energy is stored in a lightweight row; for this implementation
	// we trust taps ≤ maxEnergy check on the client, server validates rate only.
	if taps > maxEnergy {
		taps = maxEnergy
	}

	tapGain := int64(1 + p.TapLevel)
	earned := taps * tapGain
	passive := computePassiveEarned(p)
	p.LastPassive = now
	p.LastSync = now

	newBalance := p.Balance + earned + passive

	// Sliding window cross-check with sync_history
	rows, err := db.Query(`
		SELECT balance, ts FROM sync_history
		WHERE telegram_id=$1 AND ts > $2
		ORDER BY ts ASC LIMIT 60
	`, p.TelegramID, now.Add(-60*time.Second))
	if err == nil {
		defer rows.Close()
		type snap struct{ bal int64; t time.Time }
		var history []snap
		for rows.Next() {
			var s snap
			rows.Scan(&s.bal, &s.t)
			history = append(history, s)
		}
		if len(history) >= 2 {
			oldest := history[0]
			windowSec := now.Sub(oldest.t).Seconds()
			if windowSec > 0 {
				maxRate := float64(tapGain)*float64(MAX_TAPS_PER_SEC) + float64(passivePerHour(p))/3600.0 + 100
				maxFromWindow := oldest.bal + int64(windowSec*maxRate*1.2)
				if newBalance > maxFromWindow {
					newBalance = maxFromWindow
				}
			}
		}
	}

	p.Balance = newBalance
	p.LastBalance = newBalance

	// Record snapshot
	db.Exec(`INSERT INTO sync_history (telegram_id, balance, ts) VALUES ($1,$2,$3)`,
		p.TelegramID, newBalance, now)
	// Prune old snapshots
	db.Exec(`DELETE FROM sync_history WHERE telegram_id=$1 AND ts < $2`,
		p.TelegramID, now.Add(-120*time.Second))

	return newBalance, maxEnergy, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// GLOBAL STATE
// ═══════════════════════════════════════════════════════════════════════════════

var (
	rateLimiter = newRateLimiter()
	bot         *tgbotapi.BotAPI
)

// ═══════════════════════════════════════════════════════════════════════════════
// HTTP HELPERS
// ═══════════════════════════════════════════════════════════════════════════════

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Telegram-Init-Data")
}

func jsonResp(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func getIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	return r.RemoteAddr
}

// ═══════════════════════════════════════════════════════════════════════════════
// HANDLERS
// ═══════════════════════════════════════════════════════════════════════════════

// POST /api/sync
// Body: { telegramId, username, firstName, photoUrl, taps, tapLevel, energyLevel, passiveLevel }
func handleSync(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID   int64  `json:"telegramId"`
		Username     string `json:"username"`
		FirstName    string `json:"firstName"`
		PhotoURL     string `json:"photoUrl"`
		Taps         int64  `json:"taps"`
		TapLevel     int    `json:"tapLevel"`
		EnergyLevel  int    `json:"energyLevel"`
		PassiveLevel int    `json:"passiveLevel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400); return
	}
	if req.TelegramID == 0 { jsonError(w, "telegramId required", 400); return }
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	// Rate limit: 2 req/s per user
	key := fmt.Sprintf("sync:%d", req.TelegramID)
	if !rateLimiter.Allow(key, 2) { jsonError(w, "rate limit exceeded", 429); return }

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, req.FirstName, req.PhotoURL)
	if err != nil { jsonError(w, "db error", 500); return }

	// Clamp levels from client (client hints, server authoritative)
	if req.TapLevel > MAX_TAP_LEVEL       { req.TapLevel = MAX_TAP_LEVEL }
	if req.EnergyLevel > MAX_ENERGY_LEVEL { req.EnergyLevel = MAX_ENERGY_LEVEL }
	if req.PassiveLevel > MAX_PASSIVE_LEVEL { req.PassiveLevel = MAX_PASSIVE_LEVEL }

	newBal, _, err2 := processServerTaps(p, req.Taps, p.Balance)
	if err2 != nil {
		// Don't fail, just use current balance
		newBal = p.Balance
	}

	passive := computePassiveEarned(p)
	p.LastPassive = time.Now()
	p.Balance = newBal + passive

	if err := savePlayer(p); err != nil {
		jsonError(w, "save error", 500); return
	}

	jsonResp(w, map[string]interface{}{
		"status":        "ok",
		"balance":       p.Balance,
		"passiveEarned": passive,
		"rank":          getRank(p.Balance),
	})
}

// POST /api/ad-reward
func handleAdReward(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		BlockID    string `json:"blockId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400); return
	}
	if req.TelegramID == 0 { jsonError(w, "unauthorized", 401); return }
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	// Rate limit: 2 req/s per user+IP
	key := fmt.Sprintf("ad:%d:%s", req.TelegramID, getIP(r))
	if !rateLimiter.Allow(key, 2) { jsonError(w, "rate limit exceeded", 429); return }

	if req.BlockID != ADSGRAM_BLOCK_ID {
		jsonError(w, "invalid block id", 400); return
	}

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	p.Balance += REWARD_WATCH_AD
	p.LastBalance = p.Balance
	savePlayer(p)

	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  REWARD_WATCH_AD,
		"balance": p.Balance,
		"message": fmt.Sprintf("+%s ZX din reclamă!", formatInt(REWARD_WATCH_AD)),
	})
}

// GET /api/leaderboard
func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	rows, err := db.Query(`
		SELECT telegram_id, username, first_name, photo_url, balance, level
		FROM players ORDER BY balance DESC LIMIT 50
	`)
	if err != nil { jsonError(w, "db error", 500); return }
	defer rows.Close()

	type Entry struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		FirstName  string `json:"firstName"`
		PhotoURL   string `json:"photoUrl"`
		Balance    int64  `json:"balance"`
		Level      int    `json:"level"`
		RankName   string `json:"rankName"`
		Rank       int    `json:"rank"`
	}
	var list []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(&e.TelegramID, &e.Username, &e.FirstName, &e.PhotoURL, &e.Balance, &e.Level)
		e.RankName = getRank(e.Balance).Name
		list = append(list, e)
	}
	for i := range list { list[i].Rank = i + 1 }
	jsonResp(w, list)
}

// POST /api/checkin
func handleCheckin(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	today := time.Now().Format("2006-01-02")
	if p.LastCheckin == today {
		jsonResp(w, map[string]interface{}{
			"success": false,
			"message": "Ai deja check-in-ul de azi!",
			"streak":  p.CheckinStreak,
		})
		return
	}

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if p.LastCheckin == yesterday {
		p.CheckinStreak++
	} else {
		p.CheckinStreak = 1
	}
	p.LastCheckin = today

	reward := int64(500 + 100*p.CheckinStreak)
	if reward > 3000 { reward = 3000 }
	if p.CheckinStreak == 7  { reward += 5000 }
	if p.CheckinStreak == 30 { reward += 25000 }

	p.Balance += reward
	p.LastBalance = p.Balance
	savePlayer(p)

	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  reward,
		"streak":  p.CheckinStreak,
		"balance": p.Balance,
		"message": fmt.Sprintf("🎁 Zi %d! +%s ZX", p.CheckinStreak, formatInt(reward)),
	})
}

// GET /api/referral?telegramId=
func handleReferral(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 { jsonError(w, "telegramId required", 400); return }

	p, err := getOrCreatePlayer(tid, r.URL.Query().Get("username"), "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	rows, _ := db.Query(`SELECT r.referee_id, p.username FROM referrals r JOIN players p ON p.telegram_id=r.referee_id WHERE r.referrer_id=$1`, tid)
	var refs []map[string]interface{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var refID int64
			var refUsername string
			rows.Scan(&refID, &refUsername)
			refs = append(refs, map[string]interface{}{"id": refID, "username": refUsername})
		}
	}

	jsonResp(w, map[string]interface{}{
		"code":     p.ReferralCode,
		"refs":     refs,
		"count":    len(refs),
		"earnings": int64(len(refs)) * 500,
	})
}

// POST /api/referral/claim
func handleClaimReferral(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID   int64  `json:"telegramId"`
		Username     string `json:"username"`
		ReferralCode string `json:"referralCode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	// Find owner
	var ownerID int64
	err := db.QueryRow(`SELECT telegram_id FROM players WHERE referral_code=$1`, req.ReferralCode).Scan(&ownerID)
	if err == sql.ErrNoRows { jsonResp(w, map[string]interface{}{"success": false, "message": "Cod invalid."}); return }
	if err != nil { jsonError(w, "db error", 500); return }
	if ownerID == req.TelegramID { jsonResp(w, map[string]interface{}{"success": false, "message": "Nu poți folosi propriul cod!"}); return }

	// Check already referred
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM referrals WHERE referee_id=$1`, req.TelegramID).Scan(&n)
	if n > 0 { jsonResp(w, map[string]interface{}{"success": false, "message": "Ai folosit deja un cod."}); return }

	p, _ := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	_, err2 := db.Exec(`INSERT INTO referrals (referrer_id, referee_id) VALUES ($1,$2)`, ownerID, req.TelegramID)
	if err2 != nil { jsonResp(w, map[string]interface{}{"success": false, "message": "Eroare BD."}); return }

	p.Balance += 1000
	p.LastBalance = p.Balance
	savePlayer(p)

	// Reward owner
	db.Exec(`UPDATE players SET balance=balance+500, last_balance=balance+500 WHERE telegram_id=$1`, ownerID)

	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  1000,
		"balance": p.Balance,
		"message": "✅ +1000 ZX pentru tine, +500 ZX pentru invitant!",
	})
}

// GET /api/passive?telegramId=
func handlePassiveInfo(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 { jsonResp(w, map[string]interface{}{"earned": 0, "perHour": 0}); return }

	p := &Player{}
	err := db.QueryRow(`SELECT telegram_id, balance, passive_level, last_passive, points_per_hour, is_pro FROM players WHERE telegram_id=$1`, tid).
		Scan(&p.TelegramID, &p.Balance, &p.PassiveLevel, &p.LastPassive, &p.PointsPerHour, &p.IsPro)
	if err != nil { jsonResp(w, map[string]interface{}{"earned": 0, "perHour": 0}); return }

	jsonResp(w, map[string]interface{}{
		"earned":  computePassiveEarned(p),
		"perHour": passivePerHour(p),
		"balance": p.Balance,
	})
}

// POST /api/task/claim
func handleTaskClaim(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		TaskID     string `json:"taskId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	if hasClaimedTask(req.TelegramID, req.TaskID) {
		jsonResp(w, map[string]interface{}{"success": false, "message": "Task deja revendicat."})
		return
	}

	var reward int64
	var msg string
	switch req.TaskID {
	case "channel1":
		if !checkChannelMember(TG_CHANNEL, req.TelegramID) {
			jsonResp(w, map[string]interface{}{"success": false, "message": "⚠️ Nu ești abonat la canal!"})
			return
		}
		reward = REWARD_JOIN_CHANNEL
		msg = fmt.Sprintf("✅ Abonat! +%s ZX", formatInt(reward))
	case "channel2":
		if TG_CHANNEL2 == "" {
			jsonResp(w, map[string]interface{}{"success": false, "message": "Task indisponibil."})
			return
		}
		if !checkChannelMember(TG_CHANNEL2, req.TelegramID) {
			jsonResp(w, map[string]interface{}{"success": false, "message": "⚠️ Nu ești abonat!"})
			return
		}
		reward = REWARD_CHANNEL2
		msg = fmt.Sprintf("✅ Abonat! +%s ZX", formatInt(reward))
	case "twitter":
		reward = REWARD_TWITTER
		msg = fmt.Sprintf("✅ Urmărit! +%s ZX", formatInt(reward))
	case "partner":
		reward = REWARD_PARTNER
		msg = fmt.Sprintf("✅ Activat! +%s ZX", formatInt(reward))
	default:
		jsonResp(w, map[string]interface{}{"success": false, "message": "Task necunoscut."})
		return
	}

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	claimTaskDB(req.TelegramID, req.TaskID)
	p.Balance += reward
	p.LastBalance = p.Balance
	savePlayer(p)

	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  reward,
		"balance": p.Balance,
		"message": msg,
	})
}

// ─── CARDS ────────────────────────────────────────────────────────────────────

// GET /api/cards?telegramId=
func handleGetCards(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)

	var owned map[string]int
	if tid != 0 {
		cards, _ := getPlayerCards(tid)
		owned = make(map[string]int)
		for _, c := range cards {
			owned[c.CardID] = c.Level
		}
	}

	type CardResp struct {
		ID          string  `json:"id"`
		Category    string  `json:"category"`
		Name        string  `json:"name"`
		Emoji       string  `json:"emoji"`
		Level       int     `json:"level"`
		MaxLevel    int     `json:"maxLevel"`
		Price       int64   `json:"price"`
		PPH         int64   `json:"pph"`
		NextPPH     int64   `json:"nextPph"`
	}
	var resp []CardResp
	for _, def := range cardCatalog {
		lvl := 0
		if owned != nil {
			lvl = owned[def.ID]
		}
		resp = append(resp, CardResp{
			ID:       def.ID,
			Category: def.Category,
			Name:     def.Name,
			Emoji:    def.Emoji,
			Level:    lvl,
			MaxLevel: def.MaxLevel,
			Price:    cardPrice(&def, lvl),
			PPH:      cardPPH(&def, lvl),
			NextPPH:  cardPPH(&def, lvl+1),
		})
	}
	jsonResp(w, resp)
}

// POST /api/cards/buy
func handleBuyCard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		CardID     string `json:"cardId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	def := getCardDef(req.CardID)
	if def == nil { jsonError(w, "card not found", 404); return }

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	// Get current level
	var currentLevel int
	err2 := db.QueryRow(`SELECT level FROM player_cards WHERE telegram_id=$1 AND card_id=$2`, req.TelegramID, req.CardID).Scan(&currentLevel)
	if err2 == sql.ErrNoRows { currentLevel = 0 }

	if currentLevel >= def.MaxLevel {
		jsonResp(w, map[string]interface{}{"success": false, "message": "Card la nivel maxim!"}); return
	}

	price := cardPrice(def, currentLevel)
	if p.Balance < price {
		jsonResp(w, map[string]interface{}{"success": false, "message": "Fonduri insuficiente!", "need": price}); return
	}

	// BURN: deduct from balance (token burn)
	p.Balance -= price
	p.LastBalance = p.Balance

	newLevel := currentLevel + 1
	if currentLevel == 0 {
		db.Exec(`INSERT INTO player_cards (telegram_id, card_id, level) VALUES ($1,$2,$3)`, req.TelegramID, req.CardID, newLevel)
	} else {
		db.Exec(`UPDATE player_cards SET level=$1 WHERE telegram_id=$2 AND card_id=$3`, newLevel, req.TelegramID, req.CardID)
	}

	// Recalculate total PPH
	allCards, _ := getPlayerCards(req.TelegramID)
	p.PointsPerHour = computeTotalPPH(allCards)
	savePlayer(p)

	jsonResp(w, map[string]interface{}{
		"success":      true,
		"cardId":       req.CardID,
		"newLevel":     newLevel,
		"pph":          cardPPH(def, newLevel),
		"totalPPH":     p.PointsPerHour,
		"balance":      p.Balance,
		"message":      fmt.Sprintf("✅ %s Lv.%d! +%s ZX/oră", def.Name, newLevel, formatInt(cardPPH(def, newLevel))),
	})
}

// ─── WHEEL ────────────────────────────────────────────────────────────────────

// POST /api/wheel/spin
func handleWheelSpin(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	now := time.Now()
	if p.SpinCooldown != nil && now.Before(*p.SpinCooldown) {
		remaining := p.SpinCooldown.Sub(now)
		jsonResp(w, map[string]interface{}{
			"success":     false,
			"cooldownSec": int(remaining.Seconds()),
			"message":     fmt.Sprintf("⏳ Revino în %dh %dm", int(remaining.Hours()), int(remaining.Minutes())%60),
		})
		return
	}

	// Weighted prize table
	type Prize struct {
		Label    string
		Amount   int64
		IsPremium bool
		Weight   int
	}
	prizes := []Prize{
		{"1.000 ZX",   1_000,  false, 30},
		{"2.500 ZX",   2_500,  false, 25},
		{"5.000 ZX",   5_000,  false, 20},
		{"10.000 ZX",  10_000, false, 12},
		{"25.000 ZX",  25_000, false, 7},
		{"50.000 ZX",  50_000, false, 4},
		{"1 PREMIUM",  0,      true,  2},
	}
	totalWeight := 0
	for _, pr := range prizes {
		totalWeight += pr.Weight
	}
	rng := rand.New(rand.NewSource(now.UnixNano()))
	roll := rng.Intn(totalWeight)
	var won Prize
	acc := 0
	for _, pr := range prizes {
		acc += pr.Weight
		if roll < acc {
			won = pr
			break
		}
	}

	// Apply reward
	if won.IsPremium {
		expiry := now.Add(7 * 24 * time.Hour)
		p.IsPro = true
		p.ProExpires = &expiry
	} else {
		p.Balance += won.Amount
		p.LastBalance = p.Balance
	}

	// Set 24h cooldown
	cooldown := now.Add(24 * time.Hour)
	p.SpinCooldown = &cooldown
	savePlayer(p)

	msg := fmt.Sprintf("🎉 Ai câștigat %s!", won.Label)
	if won.IsPremium { msg = "🌟 Ai câștigat 7 zile PRO ACCOUNT!" }

	jsonResp(w, map[string]interface{}{
		"success":      true,
		"prize":        won.Label,
		"amount":       won.Amount,
		"isPremium":    won.IsPremium,
		"balance":      p.Balance,
		"message":      msg,
		"nextSpinSec":  86400,
	})
}

// GET /api/wheel/status?telegramId=
func handleWheelStatus(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 { jsonResp(w, map[string]interface{}{"canSpin": true, "cooldownSec": 0}); return }

	var spinCooldown *time.Time
	db.QueryRow(`SELECT spin_cooldown FROM players WHERE telegram_id=$1`, tid).Scan(&spinCooldown)

	now := time.Now()
	if spinCooldown == nil || now.After(*spinCooldown) {
		jsonResp(w, map[string]interface{}{"canSpin": true, "cooldownSec": 0})
	} else {
		jsonResp(w, map[string]interface{}{
			"canSpin":     false,
			"cooldownSec": int(spinCooldown.Sub(now).Seconds()),
		})
	}
}

// ─── DAILY COMBO ─────────────────────────────────────────────────────────────

// POST /api/combo/claim
func handleDailyCombo(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		Cipher     string `json:"cipher"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	today := time.Now().Format("2006-01-02")
	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	if p.DailyComboClaimed == today {
		jsonResp(w, map[string]interface{}{"success": false, "message": "Combo deja revendicat azi!"}); return
	}

	// Check cipher
	if strings.ToUpper(strings.TrimSpace(req.Cipher)) != getDailyCipherAnswer() {
		jsonResp(w, map[string]interface{}{"success": false, "message": "Cifru incorect!"}); return
	}

	// Check player owns all 3 combo cards
	comboCards := getDailyComboCards()
	playerCards, _ := getPlayerCards(req.TelegramID)
	ownedMap := make(map[string]bool)
	for _, c := range playerCards { ownedMap[c.CardID] = true }
	for _, required := range comboCards {
		if !ownedMap[required] {
			def := getCardDef(required)
			name := required
			if def != nil { name = def.Emoji + " " + def.Name }
			jsonResp(w, map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Îți lipsește cardul: %s", name),
			})
			return
		}
	}

	p.Balance += REWARD_DAILY_COMBO
	p.LastBalance = p.Balance
	p.DailyComboClaimed = today
	savePlayer(p)

	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  REWARD_DAILY_COMBO,
		"balance": p.Balance,
		"message": fmt.Sprintf("🔥 COMBO ZILNIC! +%s ZX!", formatInt(REWARD_DAILY_COMBO)),
	})
}

// GET /api/combo/status?telegramId=
func handleComboStatus(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	today := time.Now().Format("2006-01-02")

	comboCards := getDailyComboCards()
	var cardNames []map[string]string
	for _, id := range comboCards {
		def := getCardDef(id)
		if def != nil {
			cardNames = append(cardNames, map[string]string{"id": id, "name": def.Name, "emoji": def.Emoji})
		}
	}

	claimed := false
	if tid != 0 {
		var claimed_date string
		db.QueryRow(`SELECT daily_combo_claimed FROM players WHERE telegram_id=$1`, tid).Scan(&claimed_date)
		claimed = claimed_date == today
	}

	jsonResp(w, map[string]interface{}{
		"date":        today,
		"cards":       cardNames,
		"claimed":     claimed,
		"reward":      REWARD_DAILY_COMBO,
		"cipherHint":  "Introduce codul Morse zilnic",
	})
}

// ─── STAKING ─────────────────────────────────────────────────────────────────

// POST /api/stake/create
func handleStakeCreate(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID   int64  `json:"telegramId"`
		Username     string `json:"username"`
		Amount       int64  `json:"amount"`
		DurationDays int    `json:"durationDays"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	var apy float64
	switch req.DurationDays {
	case 7:  apy = STAKE_7D_APY
	case 30: apy = STAKE_30D_APY
	case 90: apy = STAKE_90D_APY
	default:
		jsonResp(w, map[string]interface{}{"success": false, "message": "Durata invalidă. Alege 7, 30 sau 90 zile."}); return
	}

	if req.Amount < 1000 {
		jsonResp(w, map[string]interface{}{"success": false, "message": "Minimum 1.000 ZX pentru staking."}); return
	}

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	if p.Balance < req.Amount {
		jsonResp(w, map[string]interface{}{"success": false, "message": "Fonduri insuficiente!"}); return
	}

	// BURN / lock: deduct from balance
	p.Balance -= req.Amount
	p.LastBalance = p.Balance
	savePlayer(p)

	now := time.Now()
	unlocks := now.Add(time.Duration(req.DurationDays) * 24 * time.Hour)
	db.Exec(`INSERT INTO stakes (telegram_id, amount, duration_days, apy, started_at, unlocks_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		req.TelegramID, req.Amount, req.DurationDays, apy, now, unlocks)

	projectedReward := int64(float64(req.Amount) * apy * float64(req.DurationDays) / 365)
	jsonResp(w, map[string]interface{}{
		"success":           true,
		"stakedAmount":      req.Amount,
		"durationDays":      req.DurationDays,
		"apy":               apy * 100,
		"projectedReward":   projectedReward,
		"unlocksAt":         unlocks.Format(time.RFC3339),
		"balance":           p.Balance,
		"message":           fmt.Sprintf("✅ %s ZX staked pentru %d zile (APY %.0f%%)", formatInt(req.Amount), req.DurationDays, apy*100),
	})
}

// GET /api/stake/list?telegramId=
func handleStakeList(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 { jsonResp(w, []interface{}{}); return }

	rows, err := db.Query(`
		SELECT id, amount, duration_days, apy, started_at, unlocks_at, claimed
		FROM stakes WHERE telegram_id=$1 ORDER BY started_at DESC
	`, tid)
	if err != nil { jsonError(w, "db error", 500); return }
	defer rows.Close()

	type Stake struct {
		ID           int     `json:"id"`
		Amount       int64   `json:"amount"`
		DurationDays int     `json:"durationDays"`
		APY          float64 `json:"apy"`
		StartedAt    string  `json:"startedAt"`
		UnlocksAt    string  `json:"unlocksAt"`
		Claimed      bool    `json:"claimed"`
		CanClaim     bool    `json:"canClaim"`
		Reward       int64   `json:"reward"`
	}
	var list []Stake
	now := time.Now()
	for rows.Next() {
		var s Stake
		var startedAt, unlocksAt time.Time
		rows.Scan(&s.ID, &s.Amount, &s.DurationDays, &s.APY, &startedAt, &unlocksAt, &s.Claimed)
		s.StartedAt = startedAt.Format(time.RFC3339)
		s.UnlocksAt = unlocksAt.Format(time.RFC3339)
		s.CanClaim = !s.Claimed && now.After(unlocksAt)
		s.Reward = int64(float64(s.Amount) * s.APY * float64(s.DurationDays) / 365)
		list = append(list, s)
	}
	if list == nil { list = []Stake{} }
	jsonResp(w, list)
}

// POST /api/stake/claim
func handleStakeClaim(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64 `json:"telegramId"`
		StakeID    int   `json:"stakeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	var amount int64
	var durationDays int
	var apy float64
	var claimed bool
	var unlocksAt time.Time
	err := db.QueryRow(`SELECT amount, duration_days, apy, claimed, unlocks_at FROM stakes WHERE id=$1 AND telegram_id=$2`,
		req.StakeID, req.TelegramID).Scan(&amount, &durationDays, &apy, &claimed, &unlocksAt)
	if err == sql.ErrNoRows { jsonResp(w, map[string]interface{}{"success": false, "message": "Stake negăsit."}); return }
	if claimed { jsonResp(w, map[string]interface{}{"success": false, "message": "Deja revendicat."}); return }
	if time.Now().Before(unlocksAt) {
		remaining := time.Until(unlocksAt)
		jsonResp(w, map[string]interface{}{"success": false, "message": fmt.Sprintf("Deblochează în %dh %dm", int(remaining.Hours()), int(remaining.Minutes())%60)})
		return
	}

	reward := int64(float64(amount)*apy*float64(durationDays)/365)
	total := amount + reward

	db.Exec(`UPDATE stakes SET claimed=TRUE WHERE id=$1`, req.StakeID)

	p, _ := getOrCreatePlayer(req.TelegramID, "", "", "")
	p.Balance += total
	p.LastBalance = p.Balance
	savePlayer(p)

	jsonResp(w, map[string]interface{}{
		"success":  true,
		"returned": amount,
		"reward":   reward,
		"total":    total,
		"balance":  p.Balance,
		"message":  fmt.Sprintf("💰 +%s ZX (stake + dobândă)", formatInt(total)),
	})
}

// ─── PRO ACCOUNT ─────────────────────────────────────────────────────────────

// POST /api/pro/activate  (simulated payment — integrate Telegram Stars / TON in prod)
func handleProActivate(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		Plan       string `json:"plan"` // "monthly" | "yearly"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	var costZX int64
	var days int
	switch req.Plan {
	case "monthly":
		costZX = 500_000
		days = 30
	case "yearly":
		costZX = 4_000_000
		days = 365
	default:
		jsonResp(w, map[string]interface{}{"success": false, "message": "Plan invalid."}); return
	}

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }

	if p.Balance < costZX {
		jsonResp(w, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Insuficient. Ai nevoie de %s ZX.", formatInt(costZX)),
		}); return
	}

	// BURN the PRO cost
	p.Balance -= costZX
	p.LastBalance = p.Balance
	p.IsPro = true
	now := time.Now()
	// Extend if already pro
	base := now
	if p.ProExpires != nil && p.ProExpires.After(now) { base = *p.ProExpires }
	expiry := base.Add(time.Duration(days) * 24 * time.Hour)
	p.ProExpires = &expiry
	savePlayer(p)

	jsonResp(w, map[string]interface{}{
		"success":    true,
		"plan":       req.Plan,
		"expiresAt":  expiry.Format(time.RFC3339),
		"balance":    p.Balance,
		"message":    fmt.Sprintf("🌟 PRO activat! Expiră: %s", expiry.Format("02 Jan 2006")),
	})
}

// ─── WALLET ───────────────────────────────────────────────────────────────────

func handleWalletSave(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		Address    string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil { jsonError(w, "db error", 500); return }
	p.WalletAddress = req.Address
	savePlayer(p)

	jsonResp(w, map[string]interface{}{"success": true, "address": req.Address})
}

// ─── ACCOUNT DELETE ───────────────────────────────────────────────────────────

func handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions { w.WriteHeader(200); return }
	if r.Method != http.MethodPost { jsonError(w, "method not allowed", 405); return }

	var req struct {
		TelegramID int64 `json:"telegramId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "bad request", 400); return
	}
	if !validateRequest(r, req.TelegramID) { jsonError(w, "unauthorized", 401); return }

	db.Exec(`DELETE FROM players WHERE telegram_id=$1`, req.TelegramID)
	jsonResp(w, map[string]interface{}{"success": true})
}

// ─── CONFIG + MANIFEST ────────────────────────────────────────────────────────

func handleAppConfig(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	jsonResp(w, map[string]interface{}{
		"adsgramBlockId":  ADSGRAM_BLOCK_ID,
		"linkChannel":     LINK_CHANNEL,
		"linkChannel2":    LINK_CHANNEL2,
		"linkTwitter":     LINK_TWITTER,
		"linkPartner":     LINK_PARTNER,
		"channel2Enabled": TG_CHANNEL2 != "",
		"rewardChannel":   REWARD_JOIN_CHANNEL,
		"rewardChannel2":  REWARD_CHANNEL2,
		"rewardTwitter":   REWARD_TWITTER,
		"rewardPartner":   REWARD_PARTNER,
		"rewardAd":        REWARD_WATCH_AD,
		"rewardDailyCombo": REWARD_DAILY_COMBO,
		"appUrl":          APP_URL,
		"maxTapLevel":     MAX_TAP_LEVEL,
		"maxEnergyLevel":  MAX_ENERGY_LEVEL,
		"maxPassiveLevel": MAX_PASSIVE_LEVEL,
	})
}

func handleTonManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":              APP_URL,
		"name":             "ZX Network",
		"iconUrl":          APP_URL + "/icon.png",
		"termsOfUseUrl":    APP_URL + "/terms",
		"privacyPolicyUrl": APP_URL + "/privacy",
	})
}

// ─── TELEGRAM BOT ─────────────────────────────────────────────────────────────

func checkChannelMember(channel string, userID int64) bool {
	if bot == nil || userID == 0 { return false }
	cfg := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			SuperGroupUsername: channel,
			UserID:             userID,
		},
	}
	member, err := bot.GetChatMember(cfg)
	if err != nil { return false }
	s := member.Status
	return s == "member" || s == "administrator" || s == "creator" || s == "restricted"
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if bot == nil { http.Error(w, "bot not initialized", 500); return }
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil { return }
	if update.Message == nil { return }

	msg := update.Message
	webAppURL := os.Getenv("WEBAPP_URL")
	if webAppURL == "" { webAppURL = APP_URL }

	switch {
	case msg.Text == "/start" || strings.HasPrefix(msg.Text, "/start ref_"):
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonWebApp("🎮 Joacă ZX Network", tgbotapi.WebAppInfo{URL: webAppURL}),
			),
		)
		reply := tgbotapi.NewMessage(msg.Chat.ID,
			"🧬 Bine ai venit în nucleul *ZX Network*!\n\n"+
			"⚡ Tap → Earn → Upgrade → Dominate\n"+
			"💎 Carduri · Staking · Daily Combo · Fortune Wheel")
		reply.ParseMode = "Markdown"
		reply.ReplyMarkup = keyboard
		bot.Send(reply)

		// Handle referral from deep link
		if strings.HasPrefix(msg.Text, "/start ref_") {
			code := strings.TrimPrefix(msg.Text, "/start ref_")
			u := msg.From
			username := u.UserName
			if username == "" { username = fmt.Sprintf("id%d", u.ID) }
			var ownerID int64
			err := db.QueryRow(`SELECT telegram_id FROM players WHERE referral_code=$1`, code).Scan(&ownerID)
			if err == nil && ownerID != u.ID {
				var n int
				db.QueryRow(`SELECT COUNT(*) FROM referrals WHERE referee_id=$1`, u.ID).Scan(&n)
				if n == 0 {
					getOrCreatePlayer(u.ID, username, u.FirstName, "")
					db.Exec(`INSERT INTO referrals (referrer_id, referee_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, ownerID, u.ID)
					db.Exec(`UPDATE players SET balance=balance+1000 WHERE telegram_id=$1`, u.ID)
					db.Exec(`UPDATE players SET balance=balance+500 WHERE telegram_id=$1`, ownerID)
				}
			}
		}

	case msg.Text == "/balance":
		u := msg.From
		username := u.UserName
		if username == "" { username = fmt.Sprintf("id%d", u.ID) }
		var bal int64
		db.QueryRow(`SELECT balance FROM players WHERE telegram_id=$1`, u.ID).Scan(&bal)
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("💰 Balanța ta: *%s ZX*\n🏆 Rank: *%s*", formatInt(bal), getRank(bal).Name))
		reply.ParseMode = "Markdown"
		bot.Send(reply)

	case msg.Text == "/leaderboard":
		rows, err := db.Query(`SELECT first_name, balance FROM players ORDER BY balance DESC LIMIT 5`)
		if err != nil { return }
		defer rows.Close()
		text := "🏆 *Top 5 ZX Network*\n\n"
		medals := []string{"🥇","🥈","🥉","4️⃣","5️⃣"}
		i := 0
		for rows.Next() {
			var name string
			var bal int64
			rows.Scan(&name, &bal)
			if name == "" { name = "Player" }
			text += fmt.Sprintf("%s %s — *%s ZX*\n", medals[i], name, formatInt(bal))
			i++
		}
		reply := tgbotapi.NewMessage(msg.Chat.ID, text)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
	}
}

// ─── UTIL ─────────────────────────────────────────────────────────────────────

func formatInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 { result += "." }
		result += string(c)
	}
	return result
}

// ═══════════════════════════════════════════════════════════════════════════════
// FRONTEND HTML
// ═══════════════════════════════════════════════════════════════════════════════

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
<script src="https://unpkg.com/@tonconnect/ui@latest/dist/tonconnect-ui.min.js"></script>
<script src="https://sad.adsgram.ai/js/sad.min.js"></script>
<style>
:root{
  --bg:#070b10;--panel:#0d1420;--panel2:#111b2b;
  --green:#00f5d4;--green2:#00ff87;--text:#f3fffc;
  --muted:#87a39d;--purple:#9d4dff;--orange:#ff9f43;
  --red:#e74c3c;--ton:#0098EA;--border:rgba(255,255,255,.08);
}
*{margin:0;padding:0;box-sizing:border-box;-webkit-tap-highlight-color:transparent!important;}
html,body{overscroll-behavior:none;-webkit-text-size-adjust:none;}
body{
  background:radial-gradient(circle at top left,rgba(0,245,212,.12),transparent 40%),
             radial-gradient(circle at top right,rgba(0,255,135,.08),transparent 40%),#070b10;
  color:var(--text);font-family:-apple-system,'Inter','Segoe UI',sans-serif;
  min-height:100vh;overflow-x:hidden;-webkit-user-select:none;user-select:none;touch-action:pan-y;
}
body::before{content:"";position:fixed;inset:0;pointer-events:none;z-index:0;
  background-image:linear-gradient(rgba(255,255,255,.025) 1px,transparent 1px),
                   linear-gradient(90deg,rgba(255,255,255,.025) 1px,transparent 1px);
  background-size:40px 40px;}
.app{position:relative;z-index:1;max-width:600px;margin:auto;padding-bottom:130px;}
.header{position:sticky;top:0;display:flex;justify-content:space-between;align-items:center;
  padding:12px 16px;backdrop-filter:blur(15px);-webkit-backdrop-filter:blur(15px);
  background:rgba(7,11,16,.92);border-bottom:1px solid var(--border);z-index:100;}
.userBox{display:flex;align-items:center;gap:8px;}
.headerText{display:flex;flex-direction:column;gap:1px;}
.userLabel{font-size:10px;color:var(--muted);text-transform:uppercase;letter-spacing:.5px;}
.userName{font-weight:800;font-size:14px;color:#fff;}
.rankBadge{font-size:10px;color:var(--orange);font-weight:700;}
.netBadge{background:rgba(0,245,212,.15);color:var(--green);border:1px solid rgba(0,245,212,.3);
  border-radius:999px;padding:7px 11px;font-size:11px;font-weight:800;box-shadow:0 0 18px rgba(0,245,212,.18);}
.page{padding:14px;display:flex;flex-direction:column;gap:12px;}
.section{background:linear-gradient(180deg,rgba(17,27,43,.96),rgba(13,20,32,.96));
  border-radius:22px;border:1px solid var(--border);padding:18px;}
.balance-container{text-align:center;margin-bottom:6px;}
.balanceTitle{color:var(--muted);font-size:10px;letter-spacing:2px;text-transform:uppercase;}
.balanceValue{margin-top:4px;font-size:42px;font-weight:900;color:white;
  text-shadow:0 0 20px rgba(0,245,212,.45),0 0 40px rgba(0,245,212,.2);letter-spacing:-1px;line-height:1;}
.pph-row{margin-top:5px;display:flex;justify-content:center;gap:8px;flex-wrap:wrap;}
.badge{display:inline-flex;align-items:center;gap:4px;border-radius:999px;padding:3px 9px;font-size:11px;font-weight:700;}
.badge-orange{background:rgba(255,159,67,.12);border:1px solid rgba(255,159,67,.25);color:var(--orange);}
.badge-purple{background:rgba(157,77,255,.12);border:1px solid rgba(157,77,255,.25);color:#c4a8ff;}
.badge-green{background:rgba(0,245,212,.1);border:1px solid rgba(0,245,212,.25);color:var(--green);}
.coinArea{display:flex;flex-direction:column;align-items:center;justify-content:center;padding:6px 0;}
.coin{width:min(230px,60vw);height:min(230px,60vw);cursor:pointer;
  transition:transform .07s cubic-bezier(.25,.46,.45,.94);
  -webkit-user-select:none;user-select:none;touch-action:manipulation;outline:none;will-change:transform;}
.coin:active{transform:scale(.9)!important;}
.energyRow{margin-top:14px;display:flex;gap:10px;align-items:center;}
.energyBox{flex:1;}
.energyLabel{font-size:11px;color:var(--muted);margin-bottom:5px;display:flex;justify-content:space-between;}
.energyBar{width:100%;height:10px;border-radius:999px;overflow:hidden;background:#08111b;border:1px solid rgba(255,255,255,.05);}
.energyFill{width:100%;height:100%;background:linear-gradient(90deg,var(--green),var(--green2));box-shadow:0 0 10px rgba(0,245,212,.25);transition:width .15s ease;}
.btn{border:none;cursor:pointer;padding:10px 15px;border-radius:13px;font-weight:800;font-size:13px;
  background:linear-gradient(135deg,var(--green),var(--green2));color:#04120d;
  box-shadow:0 0 18px rgba(0,245,212,.18);transition:filter .15s,transform .1s;white-space:nowrap;
  -webkit-tap-highlight-color:transparent!important;}
.btn:active{transform:scale(.95);}
.btn:hover{filter:brightness(1.07);}
.btn:disabled{opacity:.4;cursor:not-allowed;pointer-events:none;}
.btn-secondary{background:#1b2a3d;color:#a0c4c0;box-shadow:none;border:1px solid rgba(255,255,255,.07);}
.btn-danger{background:linear-gradient(135deg,#c0392b,#e74c3c);color:#fff;}
.btn-purple{background:linear-gradient(135deg,#7c3aed,#9d4dff);color:#fff;}
.btn-orange{background:linear-gradient(135deg,#e67e22,#ff9f43);color:#fff;}
.btn-ton{background:linear-gradient(135deg,#0077c2,#0098EA);color:#fff;}
.btn-sm{padding:7px 12px;font-size:11px;border-radius:10px;}
.upgrades{margin-top:16px;display:grid;grid-template-columns:repeat(2,1fr);gap:10px;}
.upgradeCard{background:rgba(0,0,0,.22);border:1px solid rgba(255,255,255,.06);border-radius:16px;padding:13px;}
.upgradeCardFull{background:rgba(0,0,0,.22);border:1px solid rgba(255,159,67,.15);border-radius:16px;padding:13px;margin-top:10px;}
.upgradeTitle{font-size:12px;font-weight:800;margin-bottom:3px;}
.upgradeSub{font-size:10px;color:var(--muted);margin-bottom:8px;}
.levelBar{width:100%;height:5px;background:rgba(255,255,255,.07);border-radius:999px;margin:5px 0 9px;overflow:hidden;}
.levelBarFill{height:100%;background:linear-gradient(90deg,var(--green),var(--green2));border-radius:999px;transition:width .3s;}
/* Bottom Nav */
.bottomNav{position:fixed;left:0;right:0;bottom:0;padding:7px 10px;padding-bottom:max(7px,env(safe-area-inset-bottom));z-index:999;}
.bottomInner{max-width:600px;margin:auto;display:grid;grid-template-columns:repeat(7,1fr);gap:3px;
  background:rgba(8,14,22,.97);border:1px solid rgba(255,255,255,.06);
  border-radius:20px;padding:7px;backdrop-filter:blur(20px);-webkit-backdrop-filter:blur(20px);}
.tabBtn{background:none;border:none;color:#87a39d;padding:6px 2px;border-radius:11px;cursor:pointer;font-weight:800;font-size:8.5px;transition:.2s;line-height:1.3;-webkit-tap-highlight-color:transparent!important;}
.tabBtn.active{color:#04120d;background:linear-gradient(135deg,var(--green),var(--green2));}
.hidden{display:none!important;}
/* Float gain */
.floatGain{position:fixed;color:white;font-weight:900;font-size:20px;pointer-events:none;z-index:99999;
  text-shadow:-1px -1px 0 #000,1px -1px 0 #000,-1px 1px 0 #000,1px 1px 0 #000,0 0 10px rgba(0,255,135,.9);
  animation:floatUp .75s ease-out forwards;will-change:transform,opacity;}
@keyframes floatUp{from{opacity:1;transform:translateY(0) scale(1)}to{opacity:0;transform:translateY(-80px) scale(.75)}}
/* Cards */
.cardGrid{display:grid;grid-template-columns:repeat(2,1fr);gap:9px;margin-top:12px;}
.cardItem{background:rgba(0,0,0,.25);border:1px solid rgba(255,255,255,.07);border-radius:16px;padding:12px;}
.cardItem.maxed{border-color:rgba(0,245,212,.2);}
.cardEmoji{font-size:22px;margin-bottom:6px;}
.cardName{font-size:12px;font-weight:800;margin-bottom:2px;}
.cardLevel{font-size:10px;color:var(--muted);margin-bottom:5px;}
.cardPPH{font-size:10px;color:var(--orange);font-weight:700;margin-bottom:8px;}
.cardPrice{font-size:10px;color:var(--muted);margin-bottom:8px;}
.catFilter{display:flex;gap:6px;overflow-x:auto;padding-bottom:4px;margin-bottom:2px;scrollbar-width:none;}
.catFilter::-webkit-scrollbar{display:none;}
.catBtn{background:rgba(255,255,255,.06);border:1px solid rgba(255,255,255,.08);color:var(--muted);
  border-radius:999px;padding:5px 12px;font-size:11px;font-weight:700;white-space:nowrap;cursor:pointer;}
.catBtn.active{background:rgba(0,245,212,.15);border-color:rgba(0,245,212,.3);color:var(--green);}
/* Leaderboard */
.lbItem{display:flex;align-items:center;padding:10px 0;border-bottom:1px solid rgba(255,255,255,.05);gap:8px;}
.lbRank{width:26px;color:var(--green);font-weight:900;font-size:13px;}
.lbName{flex:1;font-size:13px;}
.lbBal{color:#fff;font-weight:700;font-size:12px;}
.lbRankLabel{font-size:9px;color:var(--orange);}
/* Task cards */
.taskCard{background:rgba(0,0,0,.2);border:1px solid rgba(255,255,255,.06);border-radius:16px;
  padding:13px 14px;display:flex;justify-content:space-between;align-items:center;gap:10px;margin-bottom:9px;}
.taskInfo{flex:1;}
.taskTitle{font-size:13px;font-weight:800;margin-bottom:2px;}
.taskDesc{font-size:10px;color:var(--muted);}
.taskReward{font-size:11px;color:var(--green);font-weight:700;margin-top:2px;}
.verify-btn{display:none;}
/* Check-in grid */
.checkinGrid{display:grid;grid-template-columns:repeat(7,1fr);gap:5px;margin-top:12px;}
.checkinDay{aspect-ratio:1;border-radius:9px;display:flex;flex-direction:column;align-items:center;justify-content:center;
  font-size:8px;font-weight:700;background:rgba(0,0,0,.2);border:1px solid rgba(255,255,255,.06);gap:1px;}
.checkinDay.done{background:rgba(0,245,212,.12);border-color:rgba(0,245,212,.28);color:var(--green);}
.checkinDay.today{border-color:var(--orange);box-shadow:0 0 8px rgba(255,159,67,.28);}
/* Referral */
.referralBox{background:rgba(0,0,0,.2);border:1px solid rgba(157,77,255,.2);border-radius:16px;padding:14px;}
.referralCode{font-size:20px;font-weight:900;color:var(--purple);letter-spacing:3px;text-align:center;
  padding:10px;background:rgba(157,77,255,.1);border-radius:10px;margin:8px 0;text-shadow:0 0 18px rgba(157,77,255,.45);}
/* Fortune wheel */
.wheelSection{text-align:center;}
.wheel-canvas{width:min(260px,68vw);height:min(260px,68vw);margin:10px auto;cursor:pointer;border-radius:50%;
  box-shadow:0 0 30px rgba(0,245,212,.2);display:block;}
.wheelTimer{font-size:13px;color:var(--muted);margin-top:6px;}
/* Staking */
.stakeOption{background:rgba(0,0,0,.2);border:1px solid rgba(255,255,255,.07);border-radius:14px;padding:13px;margin-bottom:9px;}
.stakeHeader{display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;}
.stakeApy{font-size:18px;font-weight:900;color:var(--green);}
.stakeDays{font-size:12px;color:var(--muted);}
.stakeItem{background:rgba(0,0,0,.2);border:1px solid rgba(255,255,255,.06);border-radius:12px;padding:11px;margin-bottom:7px;}
/* Stat rows */
.stat-row{display:flex;justify-content:space-between;align-items:center;padding:8px 0;border-bottom:1px solid rgba(255,255,255,.05);font-size:12px;}
.stat-label{color:var(--muted);}
.stat-value{font-weight:700;}
/* Profile avatar */
.profile-avatar{width:72px;height:72px;border-radius:50%;background:linear-gradient(135deg,var(--green),var(--purple));
  display:flex;align-items:center;justify-content:center;font-size:28px;font-weight:900;
  margin:0 auto 10px;box-shadow:0 0 28px rgba(0,245,212,.25);}
.sectionTitle{font-size:15px;font-weight:900;margin-bottom:14px;color:#d9fff5;}
/* Toast */
.toast{position:fixed;top:66px;left:50%;transform:translateX(-50%);background:#1a2940;
  border:1px solid rgba(0,245,212,.3);color:var(--green);padding:9px 18px;border-radius:999px;
  font-weight:700;font-size:12px;z-index:99999;pointer-events:none;opacity:0;transition:opacity .3s;
  white-space:nowrap;max-width:92vw;text-align:center;}
.toast.show{opacity:1;}
/* Ad spinner */
.ad-loading{display:none;position:fixed;inset:0;background:rgba(0,0,0,.7);backdrop-filter:blur(8px);
  z-index:9000;align-items:center;justify-content:center;flex-direction:column;gap:14px;}
.ad-loading.show{display:flex;}
.ad-spinner{width:36px;height:36px;border:3px solid rgba(0,245,212,.2);border-top-color:var(--green);border-radius:50%;animation:spin .8s linear infinite;}
@keyframes spin{to{transform:rotate(360deg)}}
/* Ton wallet */
.ton-wallet-card{background:rgba(0,152,234,.08);border:1px solid rgba(0,152,234,.25);border-radius:16px;padding:18px;margin-bottom:12px;text-align:center;}
.ton-addr{font-size:11px;color:#7dd3fc;word-break:break-all;background:rgba(0,0,0,.2);padding:9px;border-radius:9px;margin-top:9px;font-family:monospace;}
.wallet-status{display:flex;align-items:center;justify-content:center;gap:7px;font-size:12px;margin-bottom:14px;}
.wallet-dot{width:7px;height:7px;border-radius:50%;background:#555;}
.wallet-dot.connected{background:var(--green);box-shadow:0 0 5px var(--green);}
</style>
</head>
<body>
<div id="toastEl" class="toast"></div>
<div id="adLoading" class="ad-loading">
  <div class="ad-spinner"></div>
  <div style="color:var(--muted);font-size:12px">Se încarcă reclama...</div>
</div>
<div class="app">

<header class="header">
  <div class="userBox">
    <div id="headerAvatarWrap"></div>
    <div class="headerText">
      <div class="userLabel">ZX Network</div>
      <div id="headerName" class="userName">Guest</div>
      <div id="headerRank" class="rankBadge">Bronze · Lv.1</div>
    </div>
  </div>
  <div class="netBadge">⚡ ZX-NET</div>
</header>

<div class="page">

<!-- ════ MINE TAB ════ -->
<div id="generatorTab" class="section">
  <div class="balance-container">
    <div class="balanceTitle">Total ZX Tokens</div>
    <div id="balanceDisplay" class="balanceValue">0</div>
    <div class="pph-row">
      <span class="badge badge-orange" id="passiveBadgeRow" style="display:none">⚙️ <span id="passiveRate">0</span>/hr</span>
      <span class="badge badge-purple" id="proBadge" style="display:none">🌟 PRO</span>
    </div>
  </div>
  <div class="coinArea">
    <svg id="coin" class="coin" viewBox="0 0 500 500" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <radialGradient id="cg" cx="50%" cy="40%" r="55%">
          <stop offset="0%" stop-color="#7affd8"/><stop offset="55%" stop-color="#00ff87"/><stop offset="100%" stop-color="#00a85a"/>
        </radialGradient>
        <filter id="glow" x="-30%" y="-30%" width="160%" height="160%">
          <feGaussianBlur stdDeviation="5" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
      </defs>
      <circle cx="250" cy="250" r="218" fill="none" stroke="#b0c4de" stroke-width="5"/>
      <circle cx="250" cy="250" r="205" fill="none" stroke="url(#cg)" stroke-width="3.5"/>
      <circle cx="250" cy="250" r="190" fill="#05101c" stroke="#00ff87" stroke-width="1.5"/>
      <g filter="url(#glow)">
        <path d="M175 185 L325 185 L175 315 L325 315" fill="none" stroke="#fff" stroke-width="19" stroke-linecap="round" stroke-linejoin="round"/>
        <path d="M175 185 L325 315 M325 185 L175 315" fill="none" stroke="#fff" stroke-width="19" stroke-linecap="round" stroke-linejoin="round"/>
      </g>
    </svg>
  </div>
  <div class="energyRow">
    <div class="energyBox">
      <div class="energyLabel"><span>⚡ Energie</span><span id="energyText">500/500</span></div>
      <div class="energyBar"><div id="energyFill" class="energyFill"></div></div>
    </div>
    <button id="rechargeBtn" class="btn btn-secondary btn-sm">Reîncarcă</button>
  </div>
  <div class="upgrades">
    <div class="upgradeCard">
      <div class="upgradeTitle">👆 Multitap</div>
      <div class="upgradeSub">Lv.<span id="tapLvl">0</span>/20 · +<span id="tapGain">1</span>/tap</div>
      <div class="levelBar"><div id="tapLvlBar" class="levelBarFill" style="width:0%"></div></div>
      <div class="upgradeSub">Cost: <span id="tapCost">1.000</span> ZX</div>
      <button id="buyTap" class="btn" style="width:100%;font-size:11px">Upgrade</button>
    </div>
    <div class="upgradeCard">
      <div class="upgradeTitle">⚡ Max Energie</div>
      <div class="upgradeSub">Lv.<span id="energyLvl">0</span>/20 · <span id="energyMax">500</span></div>
      <div class="levelBar"><div id="energyLvlBar" class="levelBarFill" style="width:0%"></div></div>
      <div class="upgradeSub">Cost: <span id="energyCost">2.500</span> ZX</div>
      <button id="buyEnergy" class="btn" style="width:100%;font-size:11px">Upgrade</button>
    </div>
  </div>
  <div class="upgradeCardFull">
    <div class="upgradeTitle">⚙️ Passive Mining — Lv.<span id="passiveLvl">0</span>/20</div>
    <div class="levelBar"><div id="passiveLvlBar" class="levelBarFill" style="width:0%"></div></div>
    <div class="upgradeSub"><span id="passiveRateDisp">0</span> ZX/oră · Cost: <span id="passiveCost">5.000</span> ZX</div>
    <button id="buyPassive" class="btn btn-orange" style="width:100%;margin-top:7px;font-size:11px">Upgrade Passive Mining</button>
  </div>
</div>

<!-- ════ CARDS TAB ════ -->
<div id="cardsTab" class="section hidden">
  <div class="sectionTitle">💎 Mine — Carduri Pasive</div>
  <div style="font-size:12px;color:var(--muted);margin-bottom:10px">Total PPH: <strong id="cardsTotalPPH" style="color:var(--orange)">0</strong> ZX/oră</div>
  <div class="catFilter" id="catFilter">
    <button class="catBtn active" data-cat="all">🌐 Toate</button>
    <button class="catBtn" data-cat="PR">📢 PR</button>
    <button class="catBtn" data-cat="Marketing">📊 Marketing</button>
    <button class="catBtn" data-cat="Web3">🌐 Web3</button>
    <button class="catBtn" data-cat="Legal">⚖️ Legal</button>
    <button class="catBtn" data-cat="Team">👥 Team</button>
  </div>
  <div class="cardGrid" id="cardGrid"></div>
</div>

<!-- ════ DAILY TAB ════ -->
<div id="dailyTab" class="section hidden">
  <div class="sectionTitle">📅 Daily & Roata Norocului</div>

  <!-- Fortune Wheel -->
  <div class="stakeOption wheelSection">
    <div style="font-size:14px;font-weight:900;margin-bottom:4px">🎰 Fortune Wheel</div>
    <div style="font-size:11px;color:var(--muted);margin-bottom:10px">1 spin per 24 ore · Premii: 1k–50k ZX sau PRO</div>
    <canvas id="wheelCanvas" class="wheel-canvas" width="260" height="260"></canvas>
    <div id="wheelTimer" class="wheelTimer">Disponibil acum!</div>
    <button id="spinBtn" class="btn btn-purple" style="width:100%;margin-top:10px">🎰 Rotește Roata</button>
  </div>

  <!-- Daily Combo -->
  <div class="stakeOption" style="margin-top:4px">
    <div style="font-size:14px;font-weight:900;margin-bottom:4px">🔐 Daily Combo</div>
    <div style="font-size:11px;color:var(--muted);margin-bottom:8px">Deține cardurile zilei + introdu cifrul → <strong style="color:var(--orange)">+5.000.000 ZX</strong></div>
    <div id="comboCards" style="margin-bottom:10px;font-size:12px;color:var(--muted)">Se încarcă...</div>
    <input id="cipherInput" type="text" placeholder="Introdu cifrul Morse..." maxlength="20"
      style="width:100%;background:rgba(0,0,0,.25);border:1px solid rgba(255,255,255,.1);border-radius:10px;padding:10px 12px;color:white;font-size:13px;outline:none;margin-bottom:9px;text-transform:uppercase"/>
    <button id="comboBtn" class="btn btn-orange" style="width:100%;font-size:12px">🔓 Revendică Combo</button>
    <div id="comboStatus" style="font-size:11px;margin-top:7px;color:var(--muted)"></div>
  </div>

  <!-- Daily Check-in -->
  <div class="referralBox" style="margin-top:4px">
    <div style="font-size:13px;font-weight:800;margin-bottom:4px">📅 Daily Check-in</div>
    <div style="font-size:11px;color:var(--muted);margin:4px 0 8px">Streak: <strong id="streakDisp">0</strong> zile</div>
    <div class="checkinGrid" id="checkinGrid"></div>
    <button id="checkinBtn" class="btn" style="width:100%;margin-top:12px;font-size:12px">🎁 Revendică Recompensa Zilnică</button>
  </div>
</div>

<!-- ════ TASKS TAB ════ -->
<div id="tasksTab" class="section hidden">
  <div class="sectionTitle">💼 Tasks & Misiuni</div>
  <div class="taskCard">
    <div class="taskInfo">
      <div class="taskTitle">📺 Vizionează Reclamă</div>
      <div class="taskDesc">Reclamă Adsgram, fără cooldown</div>
      <div class="taskReward">+<span id="rewardAdDisp">1.000</span> ZX</div>
    </div>
    <button id="watchAdBtn" class="btn btn-sm">▶ Watch</button>
  </div>
  <div class="taskCard" id="taskCh1Card">
    <div class="taskInfo">
      <div class="taskTitle">📢 Abonare Canal Telegram</div>
      <div class="taskDesc">Abonează-te la canalul oficial</div>
      <div class="taskReward">+<span id="rewardCh1Disp">10.000</span> ZX</div>
    </div>
    <div style="display:flex;flex-direction:column;gap:5px;align-items:flex-end">
      <a id="ch1Link" href="#" target="_blank" class="btn btn-sm btn-secondary" onclick="showVerifyBtn('channel1')">Deschide</a>
      <button id="taskBtn_channel1" class="btn btn-sm verify-btn" onclick="claimTask('channel1',this)">Verifică ✓</button>
    </div>
  </div>
  <div class="taskCard hidden" id="taskCh2Card">
    <div class="taskInfo">
      <div class="taskTitle">📢 Al doilea Canal</div>
      <div class="taskDesc">Abonează-te la canalul secundar</div>
      <div class="taskReward">+<span id="rewardCh2Disp">1.500</span> ZX</div>
    </div>
    <div style="display:flex;flex-direction:column;gap:5px;align-items:flex-end">
      <a id="ch2Link" href="#" target="_blank" class="btn btn-sm btn-secondary" onclick="showVerifyBtn('channel2')">Deschide</a>
      <button id="taskBtn_channel2" class="btn btn-sm verify-btn" onclick="claimTask('channel2',this)">Verifică ✓</button>
    </div>
  </div>
  <div class="taskCard">
    <div class="taskInfo">
      <div class="taskTitle">🐦 Follow Twitter / X</div>
      <div class="taskDesc">Urmărește contul oficial</div>
      <div class="taskReward">+<span id="rewardTwDisp">5.000</span> ZX</div>
    </div>
    <div style="display:flex;flex-direction:column;gap:5px;align-items:flex-end">
      <a id="twLink" href="#" target="_blank" class="btn btn-sm btn-secondary" onclick="showVerifyBtn('twitter')">Deschide</a>
      <button id="taskBtn_twitter" class="btn btn-sm verify-btn" onclick="claimTask('twitter',this)">Revendică ✓</button>
    </div>
  </div>
  <div class="taskCard">
    <div class="taskInfo">
      <div class="taskTitle">🤖 Start Bot Partener</div>
      <div class="taskDesc">Activează botul partener</div>
      <div class="taskReward">+<span id="rewardPartnerDisp">20.000</span> ZX</div>
    </div>
    <div style="display:flex;flex-direction:column;gap:5px;align-items:flex-end">
      <a id="partnerLink" href="#" target="_blank" class="btn btn-sm btn-secondary" onclick="showVerifyBtn('partner')">Deschide</a>
      <button id="taskBtn_partner" class="btn btn-sm verify-btn" onclick="claimTask('partner',this)">Revendică ✓</button>
    </div>
  </div>
</div>

<!-- ════ REFERRAL TAB ════ -->
<div id="referralTab" class="section hidden">
  <div class="sectionTitle">👥 Referrals</div>
  <div class="referralBox">
    <div style="font-size:11px;color:var(--muted);text-align:center">Codul tău unic</div>
    <div id="myRefCode" class="referralCode">LOADING</div>
    <button id="copyRefLink" class="btn btn-purple" style="width:100%;font-size:12px">📋 Copiază Link Invitație</button>
  </div>
  <div class="stat-row" style="margin-top:14px">
    <span class="stat-label">Jucători invitați</span><span class="stat-value" id="refCount">0</span>
  </div>
  <div class="stat-row">
    <span class="stat-label">Câștiguri referral</span><span class="stat-value" id="refEarnings">0 ZX</span>
  </div>
  <div class="stat-row">
    <span class="stat-label">Bonus per invitat</span><span class="stat-value" style="color:var(--green)">500 ZX</span>
  </div>
  <div style="margin-top:16px">
    <div style="font-size:13px;font-weight:800;margin-bottom:9px">Introdu cod primit</div>
    <div style="display:flex;gap:7px">
      <input id="refInput" type="text" placeholder="ZXABCD12" maxlength="8"
        style="flex:1;background:rgba(0,0,0,.25);border:1px solid rgba(255,255,255,.1);border-radius:10px;padding:10px 12px;color:white;font-size:13px;font-weight:700;letter-spacing:2px;outline:none;text-transform:uppercase"/>
      <button id="claimRef" class="btn btn-purple btn-sm">Aplică</button>
    </div>
    <div id="refStatus" style="font-size:11px;margin-top:6px;color:var(--muted)"></div>
  </div>
</div>

<!-- ════ STAKING TAB ════ -->
<div id="stakingTab" class="section hidden">
  <div class="sectionTitle">🏦 Staking ZX</div>
  <div style="font-size:12px;color:var(--muted);margin-bottom:14px">Blochează ZX și câștigă dobândă. Tokenii sunt arși din circulație temporar.</div>
  <div class="stakeOption">
    <div class="stakeHeader"><span style="font-weight:800">7 Zile</span><span class="stakeApy">8% APY</span></div>
    <div class="upgradeSub">Câștig estimat: 0.15% din suma stakată</div>
    <div style="display:flex;gap:7px;margin-top:8px">
      <input id="stake7Input" type="number" min="1000" placeholder="Min 1.000 ZX"
        style="flex:1;background:rgba(0,0,0,.25);border:1px solid rgba(255,255,255,.1);border-radius:9px;padding:9px 11px;color:white;font-size:13px;outline:none"/>
      <button class="btn btn-sm btn-purple" onclick="stakeCreate(7,document.getElementById('stake7Input').value)">Stake</button>
    </div>
  </div>
  <div class="stakeOption">
    <div class="stakeHeader"><span style="font-weight:800">30 Zile</span><span class="stakeApy">20% APY</span></div>
    <div class="upgradeSub">Câștig estimat: 1.64% din suma stakată</div>
    <div style="display:flex;gap:7px;margin-top:8px">
      <input id="stake30Input" type="number" min="1000" placeholder="Min 1.000 ZX"
        style="flex:1;background:rgba(0,0,0,.25);border:1px solid rgba(255,255,255,.1);border-radius:9px;padding:9px 11px;color:white;font-size:13px;outline:none"/>
      <button class="btn btn-sm btn-purple" onclick="stakeCreate(30,document.getElementById('stake30Input').value)">Stake</button>
    </div>
  </div>
  <div class="stakeOption">
    <div class="stakeHeader"><span style="font-weight:800">90 Zile</span><span class="stakeApy">50% APY</span></div>
    <div class="upgradeSub">Câștig estimat: 12.3% din suma stakată</div>
    <div style="display:flex;gap:7px;margin-top:8px">
      <input id="stake90Input" type="number" min="1000" placeholder="Min 1.000 ZX"
        style="flex:1;background:rgba(0,0,0,.25);border:1px solid rgba(255,255,255,.1);border-radius:9px;padding:9px 11px;color:white;font-size:13px;outline:none"/>
      <button class="btn btn-sm btn-purple" onclick="stakeCreate(90,document.getElementById('stake90Input').value)">Stake</button>
    </div>
  </div>
  <div style="margin-top:16px">
    <div style="font-size:13px;font-weight:800;margin-bottom:9px">Stake-uri Active</div>
    <div id="stakeList"><div style="color:var(--muted);font-size:12px">Se încarcă...</div></div>
  </div>

  <!-- PRO Account -->
  <div style="margin-top:16px;background:rgba(157,77,255,.08);border:1px solid rgba(157,77,255,.2);border-radius:14px;padding:14px">
    <div style="font-size:14px;font-weight:900;margin-bottom:4px">🌟 PRO Account</div>
    <div style="font-size:11px;color:var(--muted);margin-bottom:10px">2× income pasiv · +5 energy refills/zi · Acces funcții exclusive</div>
    <div style="display:flex;gap:7px">
      <button class="btn btn-purple btn-sm" style="flex:1" onclick="activatePro('monthly')">30 Zile — 500K ZX</button>
      <button class="btn btn-purple btn-sm" style="flex:1" onclick="activatePro('yearly')">365 Zile — 4M ZX</button>
    </div>
    <div id="proStatus" style="font-size:11px;margin-top:7px;color:var(--muted)"></div>
  </div>
</div>

<!-- ════ RANK TAB ════ -->
<div id="rankTab" class="section hidden">
  <div class="sectionTitle">🏆 Global Leaderboard</div>
  <div id="leaderboard"><div style="color:var(--muted);text-align:center;padding:18px">Se încarcă...</div></div>
  <div style="margin-top:16px;background:rgba(157,77,255,.08);border:1px solid rgba(157,77,255,.2);border-radius:16px;padding:14px">
    <div style="font-size:11px;color:#b79cff">Poziția ta</div>
    <div id="myRank" style="margin-top:4px;font-size:26px;font-weight:900">#-</div>
    <div id="myRankBal" style="margin-top:3px;color:#e9dbff;font-size:13px">0 ZX</div>
  </div>
  <!-- Rank progression -->
  <div style="margin-top:14px">
    <div style="font-size:13px;font-weight:900;margin-bottom:10px">🎖️ Progresie Rank</div>
    <div id="rankProgress"></div>
  </div>
</div>

<!-- ════ WALLET TAB ════ -->
<div id="walletTab" class="section hidden">
  <div class="sectionTitle">👛 TON Wallet</div>
  <div class="ton-wallet-card">
    <div class="wallet-status">
      <div class="wallet-dot" id="walletDot"></div>
      <span id="walletStatusTxt">Wallet neconectat</span>
    </div>
    <div id="ton-connect-btn"></div>
    <div id="tonAddrDisplay" class="ton-addr hidden"></div>
    <div id="tonSavedRow" class="hidden" style="margin-top:8px;font-size:11px;color:var(--muted);text-align:center">✅ Adresa salvată</div>
  </div>
  <div style="background:rgba(0,0,0,.18);border-radius:16px;padding:14px;border:1px solid rgba(255,255,255,.05)">
    <div style="font-size:11px;color:var(--muted)">ZX Balance</div>
    <div id="walletBalance" style="margin-top:4px;font-size:30px;font-weight:900">0</div>
  </div>
</div>

<!-- ════ PROFILE TAB ════ -->
<div id="profileTab" class="section hidden">
  <div class="sectionTitle">👤 Profil</div>
  <div style="text-align:center;margin-bottom:20px">
    <div class="profile-avatar" id="profileAvatar">?</div>
    <div style="font-size:20px;font-weight:900" id="profileName">—</div>
    <div style="font-size:12px;color:var(--muted)" id="profileUsername">@—</div>
    <div style="font-size:12px;color:var(--orange);font-weight:700;margin-top:4px" id="profileRankLabel">Bronze · Lv.1</div>
  </div>
  <div class="stat-row"><span class="stat-label">💰 Balanță</span><span class="stat-value" id="profileBalance">0 ZX</span></div>
  <div class="stat-row"><span class="stat-label">📈 PPH Total</span><span class="stat-value" id="profilePPH">0 ZX/oră</span></div>
  <div class="stat-row"><span class="stat-label">👆 Nivel Multitap</span><span class="stat-value" id="profileTapLvl">0/20</span></div>
  <div class="stat-row"><span class="stat-label">⚡ Nivel Energie</span><span class="stat-value" id="profileEnergyLvl">0/20</span></div>
  <div class="stat-row"><span class="stat-label">⚙️ Passive Mining</span><span class="stat-value" id="profilePassiveLvl">0/20</span></div>
  <div class="stat-row"><span class="stat-label">📅 Streak Check-in</span><span class="stat-value" id="profileStreak">0 zile</span></div>
  <div class="stat-row"><span class="stat-label">👥 Referrali</span><span class="stat-value" id="profileReferrals">0</span></div>
  <div class="stat-row"><span class="stat-label">🔗 Telegram ID</span><span class="stat-value" id="profileTgId" style="font-size:10px;color:var(--muted)">—</span></div>
  <div class="stat-row"><span class="stat-label">🌟 PRO Account</span><span class="stat-value" id="profilePro">Inactiv</span></div>
  <div style="margin-top:20px;padding:14px;background:rgba(231,76,60,.06);border:1px solid rgba(231,76,60,.2);border-radius:16px">
    <div style="font-size:12px;font-weight:700;margin-bottom:7px;color:#ff8080">⚠️ Zona Periculoasă</div>
    <div style="font-size:11px;color:var(--muted);margin-bottom:10px">Ștergerea este permanentă și elimină tot progresul.</div>
    <button id="deleteAccountBtn" class="btn btn-danger" style="width:100%;font-size:12px">🗑️ Șterge Contul</button>
  </div>
</div>

</div><!-- .page -->
</div><!-- .app -->

<!-- Recharge Modal -->
<div id="rechargeModal" style="display:none;position:fixed;inset:0;background:rgba(0,0,0,.88);backdrop-filter:blur(12px);z-index:8000;justify-content:center;align-items:center">
  <div style="width:90%;max-width:380px;background:#0f1d2e;border-radius:22px;border:1px solid rgba(0,245,212,.2);padding:22px">
    <h2 style="margin-bottom:9px;font-size:17px">📺 Reîncarcă Energia</h2>
    <p style="color:var(--muted);font-size:12px">Vizionează 3 reclame pentru energie completă.</p>
    <div id="adCounter" style="margin-top:12px;font-size:26px;font-weight:900;text-align:center">0/3</div>
    <div style="width:100%;height:7px;background:#08111b;border-radius:999px;margin-top:9px;overflow:hidden">
      <div id="adProgress" style="height:100%;width:0%;background:linear-gradient(90deg,var(--green),var(--green2));transition:width .4s"></div>
    </div>
    <button id="watchRechargeAd" class="btn" style="width:100%;margin-top:16px;font-size:12px">▶ Watch Ad</button>
    <button id="closeRecharge" class="btn btn-secondary" style="width:100%;margin-top:8px;font-size:12px">Închide</button>
  </div>
</div>

<div class="bottomNav">
  <div class="bottomInner">
    <button class="tabBtn active" data-tab="generator">🎮<br>Mine</button>
    <button class="tabBtn" data-tab="cards">💎<br>Cards</button>
    <button class="tabBtn" data-tab="daily">🎰<br>Daily</button>
    <button class="tabBtn" data-tab="tasks">💼<br>Tasks</button>
    <button class="tabBtn" data-tab="referral">👥<br>Ref</button>
    <button class="tabBtn" data-tab="staking">🏦<br>Stake</button>
    <button class="tabBtn" data-tab="rank">🏆<br>Rank</button>
  </div>
</div>

<script>
(function(){
'use strict';

// ── Telegram ──────────────────────────────────────────────────────────────────
var tg = (window.Telegram && window.Telegram.WebApp) ? window.Telegram.WebApp : null;
var cu = { id:0, username:'guest', firstName:'Guest', photoUrl:'', initData:'' };
if(tg){
  tg.ready(); tg.expand();
  if(tg.disableVerticalSwipes) tg.disableVerticalSwipes();
  var u = tg.initDataUnsafe ? tg.initDataUnsafe.user : null;
  if(u){
    cu.id        = u.id || 0;
    cu.username  = u.username || ('id'+u.id);
    cu.firstName = u.first_name || u.username || 'Player';
    cu.photoUrl  = u.photo_url || '';
  }
  cu.initData = tg.initData || '';
}
document.getElementById('headerName').textContent = cu.firstName;

function apiHeaders(){
  var h = { 'Content-Type':'application/json' };
  if(cu.initData) h['X-Telegram-Init-Data'] = cu.initData;
  return h;
}

// Set header avatar
(function(){
  var wrap = document.getElementById('headerAvatarWrap');
  if(cu.photoUrl && wrap){
    var img = document.createElement('img');
    img.src = cu.photoUrl;
    img.style.cssText = 'width:30px;height:30px;border-radius:50%;object-fit:cover;border:1.5px solid rgba(0,245,212,.35);flex-shrink:0;';
    img.onerror = function(){ this.style.display='none'; };
    wrap.appendChild(img);
  }
})();

// ── Config from server ────────────────────────────────────────────────────────
var cfg = { adsgramBlockId:'', linkChannel:'#', linkChannel2:'#', linkTwitter:'#', linkPartner:'#',
  channel2Enabled:false, rewardChannel:10000, rewardChannel2:1500, rewardTwitter:5000,
  rewardPartner:20000, rewardAd:1000, appUrl:'' };

fetch('/api/config').then(function(r){ return r.json(); }).then(function(d){
  cfg = d;
  setEl('ch1Link','href',cfg.linkChannel); setEl('ch2Link','href',cfg.linkChannel2);
  setEl('twLink','href',cfg.linkTwitter); setEl('partnerLink','href',cfg.linkPartner);
  if(cfg.channel2Enabled){ var e=document.getElementById('taskCh2Card'); if(e) e.classList.remove('hidden'); }
  setTxt('rewardAdDisp',fmt(cfg.rewardAd)); setTxt('rewardCh1Disp',fmt(cfg.rewardChannel));
  setTxt('rewardCh2Disp',fmt(cfg.rewardChannel2)); setTxt('rewardTwDisp',fmt(cfg.rewardTwitter));
  setTxt('rewardPartnerDisp',fmt(cfg.rewardPartner));
  initAdsgram(); initTonConnect(); restoreVerifyBtns();
}).catch(function(){ initAdsgram(); initTonConnect(); restoreVerifyBtns(); });

function setEl(id,attr,val){ var e=document.getElementById(id); if(e) e[attr]=val; }
function setTxt(id,val){ var e=document.getElementById(id); if(e) e.textContent=val; }
function fmt(v){ return Number(v).toLocaleString('ro-RO'); }

var _tt;
function toast(m,d){
  var el=document.getElementById('toastEl');
  el.textContent=m; el.classList.add('show');
  clearTimeout(_tt); _tt=setTimeout(function(){ el.classList.remove('show'); },d||2500);
}

// ── Local State ────────────────────────────────────────────────────────────────
var SK='zxnet-v5';
var raw={}; try{ raw=JSON.parse(localStorage.getItem(SK)||'{}'); }catch(e){}
var MAX=20;
var state={
  balance:   raw.balance   !== undefined ? raw.balance  : 0,
  energy:    raw.energy    !== undefined ? raw.energy   : 500,
  maxEnergy: raw.maxEnergy !== undefined ? raw.maxEnergy: 500,
  tapLevel:  Math.min(raw.tapLevel   ||0, MAX),
  energyLevel: Math.min(raw.energyLevel ||0, MAX),
  passiveLevel:Math.min(raw.passiveLevel||0, MAX),
  claimedTasks: raw.claimedTasks  ||{},
  checkinStreak:raw.checkinStreak ||0,
  lastCheckin:  raw.lastCheckin   ||'',
  referralCode: raw.referralCode  ||'',
  referredByDone: raw.referredByDone||false,
  walletAddress:  raw.walletAddress ||'',
  linkClicked:    raw.linkClicked   ||{},
  refCount:       raw.refCount      ||0,
  isPro:          raw.isPro         ||false,
  pendingTaps:    0,
  lastSyncBal:    raw.balance||0
};
function save(){ try{ localStorage.setItem(SK,JSON.stringify(state)); }catch(e){} }

// ── Level formulas ─────────────────────────────────────────────────────────────
function tapGain()     { return 1 + state.tapLevel; }
function tapCost()     { if(state.tapLevel>=MAX) return Infinity; return Math.round(1000*Math.pow(state.tapLevel+1,2.5)); }
function energyCost()  { if(state.energyLevel>=MAX) return Infinity; return Math.round(2500*Math.pow(state.energyLevel+1,2.5)); }
function passiveCost() { if(state.passiveLevel>=MAX) return Infinity; return Math.round(5000*Math.pow(state.passiveLevel+1,2.8)); }
function pph(lvl)      { return lvl<=0?0:Math.round(100*Math.pow(1.8,lvl-1)); }
function calcMax(lvl)  { return 500+lvl*500; }

// ── Ranks ──────────────────────────────────────────────────────────────────────
var RANKS=[
  {n:'Bronze',min:0},{n:'Silver',min:50000},{n:'Gold',min:250000},
  {n:'Platinum',min:1e6},{n:'Diamond',min:5e6},{n:'Elite',min:2e7},
  {n:'Master',min:7.5e7},{n:'GrandMaster',min:2e8},{n:'Legend',min:5e8},{n:'Grandmaster+',min:1e9}
];
function getRankName(bal){
  var r=RANKS[0];
  for(var i=0;i<RANKS.length;i++){ if(bal>=RANKS[i].min) r=RANKS[i]; }
  return r.n;
}
function getRankLevel(bal){
  var lv=1;
  for(var i=0;i<RANKS.length;i++){ if(bal>=RANKS[i].min) lv=i+1; }
  return lv;
}

// ── UI ─────────────────────────────────────────────────────────────────────────
function updateUI(){
  var bal=state.balance;
  setTxt('balanceDisplay',fmt(bal));
  setTxt('walletBalance',fmt(bal));

  var rnk=getRankName(bal), lv=getRankLevel(bal);
  setTxt('headerRank',rnk+' · Lv.'+lv);

  var pct=state.maxEnergy>0?(state.energy/state.maxEnergy)*100:0;
  document.getElementById('energyFill').style.width=pct+'%';
  setTxt('energyText',state.energy+'/'+state.maxEnergy);

  // Tap
  var tl=state.tapLevel;
  setTxt('tapLvl',tl); setTxt('tapGain',tapGain());
  document.getElementById('tapLvlBar').style.width=(tl/MAX*100)+'%';
  if(tl>=MAX){ setTxt('tapCost','MAX'); disBtn('buyTap','✅ MAX'); }
  else{ setTxt('tapCost',fmt(tapCost())); enBtn('buyTap','Upgrade',state.balance<tapCost()); }

  // Energy
  var el=state.energyLevel;
  setTxt('energyLvl',el); setTxt('energyMax',fmt(calcMax(el)));
  document.getElementById('energyLvlBar').style.width=(el/MAX*100)+'%';
  if(el>=MAX){ setTxt('energyCost','MAX'); disBtn('buyEnergy','✅ MAX'); }
  else{ setTxt('energyCost',fmt(energyCost())); enBtn('buyEnergy','Upgrade',state.balance<energyCost()); }

  // Passive
  var pl=state.passiveLevel;
  setTxt('passiveLvl',pl); setTxt('passiveRateDisp',fmt(pph(pl)));
  document.getElementById('passiveLvlBar').style.width=(pl/MAX*100)+'%';
  var totalPPH=pph(pl); // cards PPH added server-side
  setTxt('passiveRate',fmt(totalPPH));
  if(pl>=MAX){ setTxt('passiveCost','MAX'); disBtn('buyPassive','✅ MAX'); }
  else{ setTxt('passiveCost',fmt(passiveCost())); enBtn('buyPassive','Upgrade Passive Mining',state.balance<passiveCost()); }

  var pb=document.getElementById('passiveBadgeRow');
  if(totalPPH>0) pb.style.display=''; else pb.style.display='none';
  var proBadge=document.getElementById('proBadge');
  if(state.isPro) proBadge.style.display=''; else proBadge.style.display='none';

  setTxt('streakDisp',state.checkinStreak);
  updateCheckinGrid();
  restoreTaskBtns();
}

function disBtn(id,txt){ var b=document.getElementById(id); if(b){ b.disabled=true; b.textContent=txt; } }
function enBtn(id,txt,disabled){ var b=document.getElementById(id); if(b){ b.disabled=!!disabled; b.textContent=txt; } }

function updateCheckinGrid(){
  var grid=document.getElementById('checkinGrid'); if(!grid) return;
  grid.innerHTML='';
  var today=new Date().toISOString().split('T')[0];
  for(var i=1;i<=7;i++){
    var d=document.createElement('div'); d.className='checkinDay';
    var rw=Math.min(500+100*i,3000);
    d.innerHTML='<span>D'+i+'</span><span style="font-size:7px;color:var(--green)">'+(rw>=1000?Math.round(rw/1000)+'k':rw)+'</span>';
    if(i<=state.checkinStreak) d.classList.add('done');
    if(i===state.checkinStreak+1) d.classList.add('today');
    grid.appendChild(d);
  }
  var btn=document.getElementById('checkinBtn');
  if(state.lastCheckin===today){ btn.textContent='✅ Revine mâine'; btn.disabled=true; }
  else { btn.textContent='🎁 Revendică Recompensa Zilnică'; btn.disabled=false; }
}

function restoreTaskBtns(){
  var map={channel1:'taskBtn_channel1',channel2:'taskBtn_channel2',twitter:'taskBtn_twitter',partner:'taskBtn_partner'};
  for(var id in map){
    var btn=document.getElementById(map[id]);
    if(btn&&state.claimedTasks[id]){ btn.textContent='✅ Revendicat'; btn.disabled=true; btn.style.display='inline-block'; }
  }
}

function updateProfileTab(){
  var avatarEl=document.getElementById('profileAvatar');
  if(cu.photoUrl){
    avatarEl.innerHTML=''; avatarEl.style.background='none'; avatarEl.style.padding='0';
    var img=document.createElement('img');
    img.src=cu.photoUrl;
    img.style.cssText='width:100%;height:100%;border-radius:50%;object-fit:cover;';
    img.onerror=function(){ avatarEl.style.background='linear-gradient(135deg,var(--green),var(--purple))'; avatarEl.textContent=cu.firstName.charAt(0).toUpperCase()||'?'; };
    avatarEl.appendChild(img);
  } else {
    avatarEl.style.background='linear-gradient(135deg,var(--green),var(--purple))';
    avatarEl.textContent=cu.firstName?cu.firstName.charAt(0).toUpperCase():'?';
  }
  setTxt('profileName',cu.firstName||'—');
  setTxt('profileUsername','@'+cu.username);
  var rnk=getRankName(state.balance),lv=getRankLevel(state.balance);
  setTxt('profileRankLabel',rnk+' · Lv.'+lv);
  setTxt('profileBalance',fmt(state.balance)+' ZX');
  setTxt('profilePPH',fmt(pph(state.passiveLevel))+' ZX/oră');
  setTxt('profileTapLvl',state.tapLevel+'/20');
  setTxt('profileEnergyLvl',state.energyLevel+'/20');
  setTxt('profilePassiveLvl',state.passiveLevel+'/20');
  setTxt('profileStreak',state.checkinStreak+' zile');
  setTxt('profileReferrals',state.refCount||0);
  setTxt('profileTgId',cu.id||'—');
  setTxt('profilePro',state.isPro?'🌟 Activ':'Inactiv');
}

function updateRankProgress(){
  var div=document.getElementById('rankProgress'); if(!div) return;
  div.innerHTML='';
  var bal=state.balance;
  for(var i=0;i<RANKS.length;i++){
    var r=RANKS[i];
    var next=RANKS[i+1];
    var pct=next?Math.min(100,Math.max(0,(bal-r.min)/(next.min-r.min)*100)):100;
    var isActive=bal>=r.min&&(!next||bal<next.min);
    var row=document.createElement('div');
    row.style.cssText='margin-bottom:8px;';
    row.innerHTML='<div style="display:flex;justify-content:space-between;font-size:11px;margin-bottom:4px;">'
      +'<span style="font-weight:800;color:'+(isActive?'var(--green)':'var(--muted)')+'">Lv.'+(i+1)+' '+r.n+'</span>'
      +'<span style="color:var(--muted)">'+(next?fmt(next.min)+' ZX':'MAX')+'</span></div>'
      +'<div style="height:5px;background:rgba(255,255,255,.06);border-radius:999px;overflow:hidden;">'
      +'<div style="height:100%;width:'+pct+'%;background:'+(isActive?'linear-gradient(90deg,var(--green),var(--green2))':'rgba(255,255,255,.15)')+';border-radius:999px;"></div></div>';
    div.appendChild(row);
  }
}

// ── Verify btn for tasks ───────────────────────────────────────────────────────
window.showVerifyBtn=function(id){
  var btn=document.getElementById('taskBtn_'+id);
  if(btn&&!state.claimedTasks[id]){ btn.style.display='inline-block'; state.linkClicked[id]=true; save(); }
};
function restoreVerifyBtns(){
  ['channel1','channel2','twitter','partner'].forEach(function(id){
    var btn=document.getElementById('taskBtn_'+id);
    if(btn&&state.linkClicked[id]&&!state.claimedTasks[id]) btn.style.display='inline-block';
  });
}

// ── Server Sync ────────────────────────────────────────────────────────────────
var syncTimer=null;
var tapBuffer=0;
function syncNow(immediate){
  if(!cu.id) return;
  clearTimeout(syncTimer);
  var taps=tapBuffer; tapBuffer=0;
  syncTimer=setTimeout(function(){
    fetch('/api/sync',{
      method:'POST', headers:apiHeaders(),
      body:JSON.stringify({
        telegramId:cu.id, username:cu.username, firstName:cu.firstName, photoUrl:cu.photoUrl,
        taps:taps, tapLevel:state.tapLevel, energyLevel:state.energyLevel, passiveLevel:state.passiveLevel
      })
    }).then(function(r){ return r.json(); }).then(function(d){
      if(d.balance!==undefined){ state.balance=d.balance; state.lastSyncBal=d.balance; save(); updateUI(); }
      if(d.passiveEarned&&d.passiveEarned>0) toast('⚙️ +'+fmt(d.passiveEarned)+' ZX passive',3000);
      if(d.rank) setTxt('headerRank',d.rank.name+' · Lv.'+d.rank.level);
    }).catch(function(){});
  }, immediate?300:2000);
}

// ── Coin tap ──────────────────────────────────────────────────────────────────
var coin=document.getElementById('coin'), lastTap=0;
function doTap(x,y){
  if(state.energy<=0){ toast('⚡ Energie epuizată!'); return; }
  var now=Date.now(); if(now-lastTap<40) return; lastTap=now;
  var gain=tapGain();
  state.balance+=gain; state.energy=Math.max(0,state.energy-1);
  tapBuffer+=1;
  spawnFloat(gain,x,y); save(); updateUI();
  syncNow(false);
}
function spawnFloat(v,x,y){
  var el=document.createElement('div');
  el.className='floatGain'; el.textContent='+'+v;
  el.style.left=x+'px'; el.style.top=(y-10)+'px';
  document.body.appendChild(el);
  setTimeout(function(){ el.parentNode&&el.parentNode.removeChild(el); },800);
}
coin.addEventListener('touchstart',function(e){
  e.preventDefault(); e.stopPropagation();
  for(var i=0;i<e.changedTouches.length;i++){ var t=e.changedTouches[i]; doTap(t.clientX,t.clientY); }
},{passive:false});
coin.addEventListener('mousedown',function(e){ if(e.button===0) doTap(e.clientX,e.clientY); });
coin.addEventListener('contextmenu',function(e){ e.preventDefault(); });

// Energy regen
setInterval(function(){
  if(state.energy<state.maxEnergy){ state.energy=Math.min(state.maxEnergy,state.energy+1); save(); updateUI(); }
},3000);

// ── Tabs ──────────────────────────────────────────────────────────────────────
var TABS=['generator','cards','daily','tasks','referral','staking','rank','wallet','profile'];
document.querySelectorAll('.tabBtn').forEach(function(btn){
  btn.addEventListener('click',function(){
    var tab=btn.dataset.tab;
    document.querySelectorAll('.tabBtn').forEach(function(b){ b.classList.remove('active'); });
    btn.classList.add('active');
    TABS.forEach(function(id){ var el=document.getElementById(id+'Tab'); if(el) el.classList.add('hidden'); });
    var target=document.getElementById(tab+'Tab');
    if(target) target.classList.remove('hidden');
    if(tab==='rank')    { loadLeaderboard(); updateRankProgress(); }
    if(tab==='referral') loadReferral();
    if(tab==='profile')  updateProfileTab();
    if(tab==='cards')    loadCards();
    if(tab==='staking')  loadStakes();
    if(tab==='daily')    { loadComboStatus(); loadWheelStatus(); drawWheel(); }
    syncNow(true);
  });
});

// Also register wallet + profile tab (not in bottom nav but accessible)
document.querySelectorAll('[data-tab="wallet"],[data-tab="profile"]').forEach(function(btn){
  btn.addEventListener('click',function(){
    var tab=btn.dataset.tab;
    document.querySelectorAll('.tabBtn').forEach(function(b){ b.classList.remove('active'); });
    btn.classList.add('active');
    TABS.forEach(function(id){ var el=document.getElementById(id+'Tab'); if(el) el.classList.add('hidden'); });
    var target=document.getElementById(tab+'Tab');
    if(target) target.classList.remove('hidden');
    if(tab==='profile') updateProfileTab();
  });
});

// ── Upgrades ──────────────────────────────────────────────────────────────────
document.getElementById('buyTap').addEventListener('click',function(){
  if(state.tapLevel>=MAX) return;
  var c=tapCost(); if(state.balance<c) return;
  state.balance-=c; state.tapLevel++; save(); updateUI(); syncNow(true);
  toast('👆 Multitap Lv.'+state.tapLevel+' → +'+tapGain()+' ZX/tap');
});
document.getElementById('buyEnergy').addEventListener('click',function(){
  if(state.energyLevel>=MAX) return;
  var c=energyCost(); if(state.balance<c) return;
  state.balance-=c; state.energyLevel++;
  state.maxEnergy=calcMax(state.energyLevel);
  state.energy=state.maxEnergy;
  save(); updateUI(); syncNow(true);
  toast('⚡ Max Energie: '+state.maxEnergy);
});
document.getElementById('buyPassive').addEventListener('click',function(){
  if(state.passiveLevel>=MAX) return;
  var c=passiveCost(); if(state.balance<c) return;
  state.balance-=c; state.passiveLevel++; save(); updateUI(); syncNow(true);
  toast('⚙️ Passive Mining Lv.'+state.passiveLevel);
});

// ── Recharge Modal ────────────────────────────────────────────────────────────
var adCount=0;
document.getElementById('rechargeBtn').addEventListener('click',function(){
  adCount=0; setTxt('adCounter','0/3');
  document.getElementById('adProgress').style.width='0%';
  document.getElementById('rechargeModal').style.display='flex';
});
document.getElementById('watchRechargeAd').addEventListener('click',function(){
  if(window._adsgram){
    document.getElementById('rechargeModal').style.display='none';
    window._adsgram.show().then(function(){
      adCount++;
      if(adCount>=3){ state.energy=state.maxEnergy; save(); updateUI(); toast('⚡ Energie restaurată!'); syncNow(true); }
      else{ document.getElementById('rechargeModal').style.display='flex'; setTxt('adCounter',adCount+'/3'); document.getElementById('adProgress').style.width=(adCount/3*100)+'%'; }
    }).catch(function(){ document.getElementById('rechargeModal').style.display='flex'; });
  } else {
    adCount++;
    setTxt('adCounter',adCount+'/3');
    document.getElementById('adProgress').style.width=(adCount/3*100)+'%';
    if(adCount>=3){ state.energy=state.maxEnergy; save(); updateUI(); document.getElementById('rechargeModal').style.display='none'; toast('⚡ Energie restaurată!'); syncNow(true); }
  }
});
document.getElementById('closeRecharge').addEventListener('click',function(){ document.getElementById('rechargeModal').style.display='none'; });

// ── Adsgram ───────────────────────────────────────────────────────────────────
function initAdsgram(){
  if(!cfg.adsgramBlockId) return;
  if(typeof window.Adsgram==='undefined'){ return; }
  window._adsgram=window.Adsgram.init({ blockId:cfg.adsgramBlockId });
}
document.getElementById('watchAdBtn').addEventListener('click',function(){
  if(!cu.id){ toast('⚠️ Autentifică-te prin Telegram!'); return; }
  if(!window._adsgram){
    state.balance+=(cfg.rewardAd||1000); save(); updateUI(); syncNow(true);
    toast('+'+fmt(cfg.rewardAd||1000)+' ZX (dev)'); return;
  }
  document.getElementById('adLoading').classList.add('show');
  window._adsgram.show().then(function(){
    document.getElementById('adLoading').classList.remove('show');
    fetch('/api/ad-reward',{ method:'POST', headers:apiHeaders(),
      body:JSON.stringify({ telegramId:cu.id, username:cu.username, blockId:cfg.adsgramBlockId })
    }).then(function(r){ return r.json(); }).then(function(d){
      if(d.success){ state.balance=d.balance; save(); updateUI(); toast(d.message||'+'+fmt(d.reward)+' ZX!'); }
      else toast(d.message||'Eroare.');
    }).catch(function(){ state.balance+=(cfg.rewardAd||1000); save(); updateUI(); toast('+'+fmt(cfg.rewardAd||1000)+' ZX!'); });
  }).catch(function(err){
    document.getElementById('adLoading').classList.remove('show');
    if(err&&err.error) toast('ℹ️ Reclame indisponibile.');
  });
});

// ── Task Claim ────────────────────────────────────────────────────────────────
window.claimTask=function(taskId,btn){
  if(!cu.id){ toast('⚠️ Autentifică-te prin Telegram!'); return; }
  if(state.claimedTasks[taskId]){ toast('Deja revendicat.'); return; }
  btn.disabled=true; btn.textContent='⏳...';
  fetch('/api/task/claim',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, username:cu.username, taskId:taskId })
  }).then(function(r){ return r.json(); }).then(function(d){
    if(d.success){ state.claimedTasks[taskId]=true; state.balance=d.balance; save(); updateUI(); btn.textContent='✅ Revendicat'; btn.disabled=true; toast(d.message||'+'+fmt(d.reward)+' ZX!'); }
    else{ btn.textContent='Verifică ✓'; btn.disabled=false; toast(d.message||'Eroare.'); }
  }).catch(function(){ btn.textContent='Verifică ✓'; btn.disabled=false; toast('❌ Eroare.'); });
};

// ── Check-in ──────────────────────────────────────────────────────────────────
document.getElementById('checkinBtn').addEventListener('click',function(){
  if(!cu.id){ toast('⚠️ Autentifică-te prin Telegram!'); return; }
  fetch('/api/checkin',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, username:cu.username })
  }).then(function(r){ return r.json(); }).then(function(d){
    if(d.success){ state.balance=d.balance; state.checkinStreak=d.streak;
      state.lastCheckin=new Date().toISOString().split('T')[0];
      save(); updateUI(); toast('🎁 '+d.message); }
    else toast(d.message||'Deja ai check-in!');
  }).catch(function(){ toast('❌ Eroare.'); });
});

// ── Cards ─────────────────────────────────────────────────────────────────────
var cardsCat='all';
function loadCards(){
  var tid=cu.id||0;
  fetch('/api/cards?telegramId='+tid).then(function(r){ return r.json(); }).then(function(cards){
    renderCards(cards);
  }).catch(function(){});
}
function renderCards(cards){
  var grid=document.getElementById('cardGrid'); if(!grid) return;
  var totalPPH=0;
  grid.innerHTML='';
  cards.forEach(function(c){
    if(cardsCat!=='all'&&c.category!==cardsCat) return;
    totalPPH+=c.pph;
    var isMax=c.level>=c.maxLevel;
    var div=document.createElement('div');
    div.className='cardItem'+(isMax?' maxed':'');
    div.innerHTML=
      '<div class="cardEmoji">'+c.emoji+'</div>'+
      '<div class="cardName">'+c.name+'</div>'+
      '<div class="cardLevel">'+c.category+' · Lv.'+c.level+'/'+c.maxLevel+'</div>'+
      '<div class="cardPPH">⚙️ '+fmt(c.pph)+' ZX/oră'+(c.level>0?' → '+fmt(c.nextPph):'')+'</div>'+
      '<div class="cardPrice">'+(isMax?'MAX LEVEL':'Preț: '+fmt(c.price)+' ZX')+'</div>'+
      (isMax?'<button class="btn btn-sm" disabled style="width:100%;font-size:10px">✅ Max</button>'
            :'<button class="btn btn-sm" style="width:100%;font-size:10px" onclick="buyCard(\''+c.id+'\',this)">Cumpără Lv.'+(c.level+1)+'</button>');
    grid.appendChild(div);
  });
  setTxt('cardsTotalPPH',fmt(totalPPH));
}
window.buyCard=function(cardId,btn){
  if(!cu.id){ toast('⚠️ Autentifică-te!'); return; }
  btn.disabled=true; btn.textContent='⏳...';
  fetch('/api/cards/buy',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, username:cu.username, cardId:cardId })
  }).then(function(r){ return r.json(); }).then(function(d){
    if(d.success){ state.balance=d.balance; save(); updateUI(); toast(d.message); loadCards(); }
    else{ btn.disabled=false; btn.textContent='Cumpără'; toast(d.message||'Eroare.'); }
  }).catch(function(){ btn.disabled=false; btn.textContent='Cumpără'; toast('❌ Eroare.'); });
};
document.getElementById('catFilter').addEventListener('click',function(e){
  if(!e.target.classList.contains('catBtn')) return;
  document.querySelectorAll('.catBtn').forEach(function(b){ b.classList.remove('active'); });
  e.target.classList.add('active');
  cardsCat=e.target.dataset.cat;
  loadCards();
});

// ── Fortune Wheel ─────────────────────────────────────────────────────────────
var wheelPrizes=[
  {label:'1K ZX',   color:'#00f5d4'},{label:'2.5K ZX', color:'#9d4dff'},
  {label:'5K ZX',   color:'#ff9f43'},{label:'10K ZX',  color:'#00ff87'},
  {label:'25K ZX',  color:'#e74c3c'},{label:'50K ZX',  color:'#0098EA'},
  {label:'1 PREMIUM',color:'#ffd700'}
];
var wheelAngle=0, wheelSpinning=false;

function drawWheel(){
  var canvas=document.getElementById('wheelCanvas'); if(!canvas) return;
  var ctx=canvas.getContext('2d');
  var W=canvas.width, H=canvas.height, cx=W/2, cy=H/2, r=cx-10;
  var slice=Math.PI*2/wheelPrizes.length;
  ctx.clearRect(0,0,W,H);
  ctx.save(); ctx.translate(cx,cy); ctx.rotate(wheelAngle);
  for(var i=0;i<wheelPrizes.length;i++){
    var start=i*slice-Math.PI/2, end=start+slice;
    ctx.beginPath(); ctx.moveTo(0,0);
    ctx.arc(0,0,r,start,end);
    ctx.closePath();
    ctx.fillStyle=wheelPrizes[i].color; ctx.fill();
    ctx.strokeStyle='rgba(0,0,0,.3)'; ctx.lineWidth=2; ctx.stroke();
    ctx.save(); ctx.rotate(start+slice/2);
    ctx.translate(r*0.6,0); ctx.rotate(Math.PI/2);
    ctx.fillStyle='#fff'; ctx.font='bold '+(W>200?'11':'9')+'px sans-serif';
    ctx.textAlign='center'; ctx.fillText(wheelPrizes[i].label,0,0);
    ctx.restore();
  }
  ctx.restore();
  // Pointer
  ctx.beginPath(); ctx.moveTo(cx,12); ctx.lineTo(cx-8,28); ctx.lineTo(cx+8,28); ctx.closePath();
  ctx.fillStyle='#fff'; ctx.fill();
}
drawWheel();

var wheelCooldown=0;
function loadWheelStatus(){
  if(!cu.id) return;
  fetch('/api/wheel/status?telegramId='+cu.id).then(function(r){ return r.json(); }).then(function(d){
    var btn=document.getElementById('spinBtn');
    if(d.canSpin){ btn.disabled=false; setTxt('wheelTimer','Disponibil acum!'); wheelCooldown=0; }
    else{
      btn.disabled=true; wheelCooldown=d.cooldownSec;
      updateWheelTimer();
    }
  }).catch(function(){});
}
function updateWheelTimer(){
  if(wheelCooldown<=0){ setTxt('wheelTimer','Disponibil acum!'); var btn=document.getElementById('spinBtn'); if(btn) btn.disabled=false; return; }
  var h=Math.floor(wheelCooldown/3600), m=Math.floor((wheelCooldown%3600)/60), s=wheelCooldown%60;
  setTxt('wheelTimer','⏳ Revino în '+h+'h '+m+'m '+s+'s'); wheelCooldown--;
  setTimeout(updateWheelTimer,1000);
}

document.getElementById('spinBtn').addEventListener('click',function(){
  if(!cu.id){ toast('⚠️ Autentifică-te prin Telegram!'); return; }
  if(wheelSpinning) return;
  wheelSpinning=true; this.disabled=true;
  fetch('/api/wheel/spin',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, username:cu.username })
  }).then(function(r){ return r.json(); }).then(function(d){
    if(!d.success){ wheelSpinning=false; var btn=document.getElementById('spinBtn'); if(btn) btn.disabled=false; toast(d.message); return; }
    // Animate wheel
    var fullSpins=5+Math.floor(Math.random()*3);
    var targetAngle=(Math.PI*2*fullSpins)+(Math.random()*Math.PI*2);
    var duration=3000, start=null, startAngle=wheelAngle;
    function animate(ts){
      if(!start) start=ts;
      var prog=Math.min((ts-start)/duration,1);
      var ease=1-Math.pow(1-prog,4);
      wheelAngle=startAngle+targetAngle*ease;
      drawWheel();
      if(prog<1){ requestAnimationFrame(animate); }
      else{
        wheelSpinning=false;
        if(d.balance!==undefined){ state.balance=d.balance; save(); updateUI(); }
        if(d.isPremium){ state.isPro=true; save(); updateUI(); }
        toast(d.message,4000);
        wheelCooldown=d.nextSpinSec||86400;
        updateWheelTimer();
      }
    }
    requestAnimationFrame(animate);
  }).catch(function(){ wheelSpinning=false; document.getElementById('spinBtn').disabled=false; toast('❌ Eroare.'); });
});

// ── Daily Combo ────────────────────────────────────────────────────────────────
function loadComboStatus(){
  fetch('/api/combo/status'+(cu.id?'?telegramId='+cu.id:'')).then(function(r){ return r.json(); }).then(function(d){
    var div=document.getElementById('comboCards');
    if(div){
      if(d.cards&&d.cards.length){
        div.innerHTML='Carduri necesare: '+d.cards.map(function(c){ return '<strong style="color:var(--green)">'+c.emoji+' '+c.name+'</strong>'; }).join(', ');
      } else { div.innerHTML='Nu sunt carduri combo astăzi.'; }
    }
    var btn=document.getElementById('comboBtn');
    var status=document.getElementById('comboStatus');
    if(d.claimed){ if(btn) btn.disabled=true; btn.textContent='✅ Revendicat Astăzi'; if(status) { status.textContent='Revino mâine pentru combo nou!'; status.style.color='var(--green)'; } }
    else { if(btn){ btn.disabled=false; btn.textContent='🔓 Revendică Combo'; } }
  }).catch(function(){});
}
document.getElementById('comboBtn').addEventListener('click',function(){
  if(!cu.id){ toast('⚠️ Autentifică-te!'); return; }
  var cipher=document.getElementById('cipherInput').value;
  this.disabled=true; this.textContent='⏳...';
  var me=this;
  fetch('/api/combo/claim',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, username:cu.username, cipher:cipher })
  }).then(function(r){ return r.json(); }).then(function(d){
    var status=document.getElementById('comboStatus');
    if(d.success){
      state.balance=d.balance; save(); updateUI();
      me.textContent='✅ Revendicat'; me.disabled=true;
      if(status){ status.textContent='🔥 '+d.message; status.style.color='var(--orange)'; }
      toast(d.message,5000);
    } else {
      me.disabled=false; me.textContent='🔓 Revendică Combo';
      if(status){ status.textContent=d.message; status.style.color='#ff6b6b'; }
      toast(d.message);
    }
  }).catch(function(){ me.disabled=false; me.textContent='🔓 Revendică Combo'; toast('❌ Eroare.'); });
});

// ── Staking ───────────────────────────────────────────────────────────────────
window.stakeCreate=function(days,amountStr){
  if(!cu.id){ toast('⚠️ Autentifică-te!'); return; }
  var amount=parseInt(amountStr)||0;
  if(amount<1000){ toast('Minimum 1.000 ZX.'); return; }
  if(state.balance<amount){ toast('Fonduri insuficiente!'); return; }
  fetch('/api/stake/create',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, username:cu.username, amount:amount, durationDays:days })
  }).then(function(r){ return r.json(); }).then(function(d){
    if(d.success){ state.balance=d.balance; save(); updateUI(); toast(d.message,4000); loadStakes(); }
    else toast(d.message||'Eroare.');
  }).catch(function(){ toast('❌ Eroare.'); });
};
function loadStakes(){
  if(!cu.id){ document.getElementById('stakeList').innerHTML='<div style="color:var(--muted);font-size:12px">Autentifică-te pentru a vedea stake-urile.</div>'; return; }
  fetch('/api/stake/list?telegramId='+cu.id).then(function(r){ return r.json(); }).then(function(list){
    var div=document.getElementById('stakeList');
    if(!div) return;
    if(!list||!list.length){ div.innerHTML='<div style="color:var(--muted);font-size:12px">Niciun stake activ.</div>'; return; }
    div.innerHTML='';
    list.forEach(function(s){
      var item=document.createElement('div'); item.className='stakeItem';
      var unlockDate=new Date(s.unlocksAt).toLocaleDateString('ro-RO');
      item.innerHTML='<div style="display:flex;justify-content:space-between;margin-bottom:5px;">'
        +'<span style="font-weight:800">'+fmt(s.amount)+' ZX × '+s.durationDays+'d</span>'
        +'<span style="color:var(--green);font-size:12px">+('+fmt(s.reward)+' ZX)</span></div>'
        +'<div style="font-size:10px;color:var(--muted);margin-bottom:7px">Deblochează: '+unlockDate
        +(s.claimed?' · <span style="color:var(--green)">✅ Revendicat</span>':'')+'</div>'
        +(s.canClaim?'<button class="btn btn-sm btn-purple" style="width:100%;font-size:11px" onclick="stakeClaimFn('+s.id+',this)">💰 Retrage +'+fmt(s.reward)+' ZX</button>'
          :s.claimed?'':('<div style="font-size:10px;color:var(--muted)">⏳ Blocat</div>'));
      div.appendChild(item);
    });
  }).catch(function(){});
}
window.stakeClaimFn=function(id,btn){
  btn.disabled=true; btn.textContent='⏳...';
  fetch('/api/stake/claim',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, stakeId:id })
  }).then(function(r){ return r.json(); }).then(function(d){
    if(d.success){ state.balance=d.balance; save(); updateUI(); toast(d.message,4000); loadStakes(); }
    else{ btn.disabled=false; btn.textContent='Retrage'; toast(d.message); }
  }).catch(function(){ btn.disabled=false; btn.textContent='Retrage'; toast('❌ Eroare.'); });
};

// ── PRO Activate ─────────────────────────────────────────────────────────────
window.activatePro=function(plan){
  if(!cu.id){ toast('⚠️ Autentifică-te!'); return; }
  fetch('/api/pro/activate',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, username:cu.username, plan:plan })
  }).then(function(r){ return r.json(); }).then(function(d){
    var s=document.getElementById('proStatus');
    if(d.success){
      state.balance=d.balance; state.isPro=true; save(); updateUI();
      if(s){ s.textContent=d.message; s.style.color='var(--green)'; }
      toast(d.message,4000);
    } else {
      if(s){ s.textContent=d.message; s.style.color='#ff6b6b'; }
      toast(d.message);
    }
  }).catch(function(){ toast('❌ Eroare.'); });
};

// ── Check-in ──────────────────────────────────────────────────────────────────
// (already bound above)

// ── Referral ──────────────────────────────────────────────────────────────────
function loadReferral(){
  if(!cu.id){ setTxt('myRefCode','GUEST'); return; }
  fetch('/api/referral?telegramId='+cu.id+'&username='+encodeURIComponent(cu.username)).then(function(r){ return r.json(); }).then(function(d){
    state.referralCode=d.code||''; state.refCount=d.count||0;
    setTxt('myRefCode',d.code||'—');
    setTxt('refCount',d.count||0);
    setTxt('refEarnings',fmt(d.earnings||0)+' ZX');
    save();
  }).catch(function(){});
}
document.getElementById('copyRefLink').addEventListener('click',function(){
  if(!state.referralCode){ loadReferral(); toast('Se generează...'); return; }
  var link='https://t.me/ZXNetworkBot?start=ref_'+state.referralCode;
  if(navigator.clipboard){ navigator.clipboard.writeText(link).then(function(){ toast('📋 Link copiat!'); }); }
  else toast('Cod: '+state.referralCode);
});
document.getElementById('claimRef').addEventListener('click',function(){
  if(!cu.id){ toast('⚠️ Autentifică-te!'); return; }
  if(state.referredByDone){ toast('Ai folosit deja un cod.'); return; }
  var code=document.getElementById('refInput').value.trim().toUpperCase();
  if(code.length<8){ toast('Cod invalid.'); return; }
  fetch('/api/referral/claim',{ method:'POST', headers:apiHeaders(),
    body:JSON.stringify({ telegramId:cu.id, username:cu.username, referralCode:code })
  }).then(function(r){ return r.json(); }).then(function(d){
    var el=document.getElementById('refStatus');
    el.textContent=d.message||''; el.style.color=d.success?'#00ff87':'#ff6b6b';
    if(d.success){ state.balance=d.balance; state.referredByDone=true; save(); updateUI(); syncNow(true); }
  }).catch(function(){ toast('❌ Eroare.'); });
});

// ── Leaderboard ───────────────────────────────────────────────────────────────
function loadLeaderboard(){
  var board=document.getElementById('leaderboard');
  board.innerHTML='<div style="color:var(--muted);text-align:center;padding:14px">Se încarcă...</div>';
  fetch('/api/leaderboard').then(function(r){ return r.json(); }).then(function(entries){
    board.innerHTML='';
    if(!entries||!entries.length){ board.innerHTML='<div style="color:var(--muted);text-align:center;padding:18px">Niciun jucător.</div>'; return; }
    var medals=['🥇','🥈','🥉'];
    var myRank='-';
    entries.forEach(function(e,i){
      var isMe=e.telegramId===cu.id;
      if(isMe) myRank='#'+(i+1);
      var displayName=e.firstName||e.username||'Player';
      var avatarHtml;
      if(e.photoUrl){
        avatarHtml='<img src="'+e.photoUrl+'" style="width:34px;height:34px;border-radius:50%;object-fit:cover;flex-shrink:0;border:1.5px solid '+(isMe?'var(--green)':'rgba(255,255,255,.12)')+'" onerror="this.style.display=\'none\'">';
      } else {
        var clrs=['135deg,#00f5d4,#9d4dff','135deg,#ff9f43,#e74c3c','135deg,#3b82f6,#9d4dff','135deg,#00ff87,#0098EA'];
        avatarHtml='<div style="width:34px;height:34px;border-radius:50%;background:linear-gradient('+clrs[i%4]+');display:flex;align-items:center;justify-content:center;font-weight:900;font-size:13px;flex-shrink:0;">'+displayName.charAt(0).toUpperCase()+'</div>';
      }
      var row=document.createElement('div'); row.className='lbItem';
      row.innerHTML=(i<3?'<span style="font-size:18px;width:26px;text-align:center;">'+medals[i]+'</span>':'<span class="lbRank">#'+(i+1)+'</span>')
        +avatarHtml
        +'<span class="lbName" style="'+(isMe?'color:#00ff87;font-weight:900;':'')+'">'+displayName+(isMe?' ◀':'')+'<br><span class="lbRankLabel">'+e.rankName+'</span></span>'
        +'<span class="lbBal">'+fmt(e.balance)+' ZX</span>';
      board.appendChild(row);
    });
    setTxt('myRank',myRank);
    setTxt('myRankBal',fmt(state.balance)+' ZX');
  }).catch(function(){ document.getElementById('leaderboard').innerHTML='<div style="color:#ff6b6b;text-align:center">Eroare.</div>'; });
}

// ── TON Connect ───────────────────────────────────────────────────────────────
function initTonConnect(){
  if(typeof TonConnectUI==='undefined') return;
  var manifestUrl=(cfg.appUrl||'')+'/tonconnect-manifest.json';
  var tc;
  try{ tc=new TonConnectUI({ manifestUrl:manifestUrl, buttonRootId:'ton-connect-btn', actionsConfiguration:{ returnStrategy:'back' } }); }
  catch(e){ return; }
  tc.onStatusChange(function(wallet){
    if(wallet){
      var addr=wallet.account.address;
      document.getElementById('tonAddrDisplay').textContent=addr;
      document.getElementById('tonAddrDisplay').classList.remove('hidden');
      document.getElementById('walletDot').classList.add('connected');
      setTxt('walletStatusTxt','Wallet conectat ✅');
      document.getElementById('tonSavedRow').classList.remove('hidden');
      state.walletAddress=addr; save();
      if(cu.id) fetch('/api/wallet/save',{ method:'POST', headers:apiHeaders(), body:JSON.stringify({ telegramId:cu.id, username:cu.username, address:addr }) }).catch(function(){});
    } else {
      document.getElementById('walletDot').classList.remove('connected');
      setTxt('walletStatusTxt','Wallet neconectat');
      document.getElementById('tonAddrDisplay').classList.add('hidden');
      document.getElementById('tonSavedRow').classList.add('hidden');
      state.walletAddress=''; save();
    }
  });
  if(state.walletAddress){
    document.getElementById('tonAddrDisplay').textContent=state.walletAddress;
    document.getElementById('tonAddrDisplay').classList.remove('hidden');
    document.getElementById('walletDot').classList.add('connected');
    setTxt('walletStatusTxt','Wallet conectat ✅');
    document.getElementById('tonSavedRow').classList.remove('hidden');
  }
}

// ── Delete Account ────────────────────────────────────────────────────────────
document.getElementById('deleteAccountBtn').addEventListener('click',function(){
  if(!confirm('Ești sigur? Aceasta șterge COMPLET contul de pe server!')) return;
  if(!confirm('Confirmi ștergerea permanentă?')) return;
  if(cu.id){
    fetch('/api/account/delete',{ method:'POST', headers:apiHeaders(), body:JSON.stringify({ telegramId:cu.id }) })
      .then(function(r){ return r.json(); }).then(function(d){
        if(d.success){ localStorage.removeItem(SK); toast('Cont șters.',3000); setTimeout(function(){ location.reload(); },3000); }
      }).catch(function(){ localStorage.removeItem(SK); location.reload(); });
  } else { localStorage.removeItem(SK); location.reload(); }
});

// ── INIT ──────────────────────────────────────────────────────────────────────
updateUI();
drawWheel();
if(cu.id){
  setTimeout(function(){ syncNow(true); },600);
  fetch('/api/passive?telegramId='+cu.id).then(function(r){ return r.json(); }).then(function(d){
    if(d.earned&&d.earned>0) toast('⚙️ Offline: +'+fmt(d.earned)+' ZX!',4000);
  }).catch(function(){});
}

})();
</script>
</body>
</html>`

// ═══════════════════════════════════════════════════════════════════════════════
// MAIN
// ═══════════════════════════════════════════════════════════════════════════════

func main() {
	initDB()

	token := os.Getenv("TELEGRAM_TOKEN")
	if token != "" {
		var err error
		bot, err = tgbotapi.NewBotAPI(token)
		if err != nil {
			log.Println("⚠️ Bot error:", err)
		} else {
			log.Printf("✅ Bot: @%s", bot.Self.UserName)
			if wurl := os.Getenv("WEBHOOK_URL"); wurl != "" {
				if wh, err2 := tgbotapi.NewWebhook(wurl + "/webhook"); err2 == nil {
					bot.Request(wh)
					log.Println("✅ Webhook set:", wurl+"/webhook")
				}
			}
		}
	} else {
		log.Println("⚠️ TELEGRAM_TOKEN lipsă — bot dezactivat. HMAC validation disabled.")
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/sync",            handleSync)
	mux.HandleFunc("/api/ad-reward",       handleAdReward)
	mux.HandleFunc("/api/leaderboard",     handleLeaderboard)
	mux.HandleFunc("/api/checkin",         handleCheckin)
	mux.HandleFunc("/api/referral",        handleReferral)
	mux.HandleFunc("/api/referral/claim",  handleClaimReferral)
	mux.HandleFunc("/api/passive",         handlePassiveInfo)
	mux.HandleFunc("/api/task/claim",      handleTaskClaim)
	mux.HandleFunc("/api/wallet/save",     handleWalletSave)
	mux.HandleFunc("/api/account/delete",  handleDeleteAccount)
	mux.HandleFunc("/api/config",          handleAppConfig)
	// Cards
	mux.HandleFunc("/api/cards",           handleGetCards)
	mux.HandleFunc("/api/cards/buy",       handleBuyCard)
	// Wheel
	mux.HandleFunc("/api/wheel/spin",      handleWheelSpin)
	mux.HandleFunc("/api/wheel/status",    handleWheelStatus)
	// Daily combo
	mux.HandleFunc("/api/combo/claim",     handleDailyCombo)
	mux.HandleFunc("/api/combo/status",    handleComboStatus)
	// Staking
	mux.HandleFunc("/api/stake/create",    handleStakeCreate)
	mux.HandleFunc("/api/stake/list",      handleStakeList)
	mux.HandleFunc("/api/stake/claim",     handleStakeClaim)
	// PRO
	mux.HandleFunc("/api/pro/activate",    handleProActivate)
	// Static
	mux.HandleFunc("/tonconnect-manifest.json", handleTonManifest)
	mux.HandleFunc("/webhook",             handleWebhook)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(webAppHTML))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 ZX Network pornit pe portul %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
