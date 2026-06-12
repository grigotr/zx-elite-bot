package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
// CONFIGURAȚIE GLOBALĂ ȘI CONSTANTE ALE SISTEMULUI
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

	STAKE_7D_APY  = 0.08
	STAKE_30D_APY = 0.20
	STAKE_90D_APY = 0.50

	MAX_TAPS_PER_SEC = 25
	MAX_ENERGY_BASE  = 500
	WHEEL_COOLDOWN_H = 24
)

// ═══════════════════════════════════════════════════════════════════════════════
// STRUCTURI DE DATE ȘI MODELE
// ═══════════════════════════════════════════════════════════════════════════════

type RankInfo struct {
	Level int    `json:"level"`
	Name  string `json:"name"`
	Min   int64  `json:"min"`
}

type CardDef struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	Name        string  `json:"name"`
	Emoji       string  `json:"emoji"`
	BasePrice   int64   `json:"basePrice"`
	PriceScale  float64 `json:"priceScale"`
	BasePPH     int64   `json:"basePph"`
	PPHScale    float64 `json:"pphScale"`
	MaxLevel    int     `json:"maxLevel"`
}

type Player struct {
	TelegramID        int64      `json:"telegramId"`
	Username          string     `json:"username"`
	FirstName         string     `json:"firstName"`
	PhotoURL          string     `json:"photoUrl"`
	Balance           int64      `json:"balance"`
	TapLevel          int        `json:"tapLevel"`
	EnergyLevel       int        `json:"energyLevel"`
	PassiveLevel      int        `json:"passiveLevel"`
	LastSync          time.Time  `json:"lastSync"`
	LastPassive       time.Time  `json:"lastPassive"`
	CheckinStreak     int        `json:"checkinStreak"`
	LastCheckin       string     `json:"lastCheckin"`
	ReferredBy        int64      `json:"referredBy"`
	WalletAddress     string     `json:"walletAddress"`
	BonusClaimed      bool       `json:"bonusClaimed"`
	Level             int        `json:"level"`
	PointsPerHour     int64      `json:"pointsPerHour"`
	SpinCooldown      *time.Time `json:"spinCooldown"`
	DailyComboClaimed string     `json:"dailyComboClaimed"`
	ReferralCode      string     `json:"referralCode"`
	IsPro             bool       `json:"isPro"`
	ProExpires        *time.Time `json:"proExpires"`
	LastBalance       int64      `json:"lastBalance"`
}

type PlayerCard struct {
	TelegramID int64  `json:"telegramId"`
	CardID     string `json:"cardId"`
	Level      int    `json:"level"`
}

type Stake struct {
	ID           int       `json:"id"`
	TelegramID   int64     `json:"telegramId"`
	Amount       int64     `json:"amount"`
	DurationDays int       `json:"durationDays"`
	APY          float64   `json:"apy"`
	StartedAt    time.Time `json:"startedAt"`
	UnlocksAt    time.Time `json:"unlocksAt"`
	Claimed      bool      `json:"claimed"`
}

type TaskDef struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Reward      int64  `json:"reward"`
	Type        string `json:"type"`
	Link        string `json:"link"`
	TargetValue string `json:"targetValue"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// CATALOGUL DE CARDURI ȘI CLASAMENTE
// ═══════════════════════════════════════════════════════════════════════════════

var ranks = []RankInfo{
	{1, "Bronze", 0},
	{2, "Silver", 50000},
	{3, "Gold", 250000},
	{4, "Platinum", 1000000},
	{5, "Diamond", 5000000},
	{6, "Elite", 20000000},
	{7, "Master", 75000000},
	{8, "GrandMaster", 200000000},
	{9, "Legend", 500000000},
	{10, "Grandmaster+", 1000000000},
}

var cardCatalog = []CardDef{
	{"pr_viral", "PR", "Viral Campaign", "📢", 5000, 2.0, 200, 1.5, 20},
	{"pr_influencer", "PR", "Influencer Deal", "🌟", 12000, 2.1, 500, 1.6, 20},
	{"pr_press", "PR", "Press Release", "📰", 8000, 2.0, 350, 1.5, 20},
	{"mkt_ads", "Marketing", "Ad Campaign", "📊", 10000, 2.1, 400, 1.5, 20},
	{"mkt_seo", "Marketing", "SEO Boost", "🔍", 15000, 2.2, 700, 1.6, 20},
	{"mkt_community", "Marketing", "Community Manager", "👥", 7000, 2.0, 280, 1.5, 20},
	{"web3_dao", "Web3", "DAO Integration", "🏛️", 25000, 2.3, 1200, 1.7, 20},
	{"web3_nft", "Web3", "NFT Collection", "🎨", 50000, 2.4, 2500, 1.8, 20},
	{"web3_defi", "Web3", "DeFi Protocol", "💱", 80000, 2.5, 4000, 1.9, 20},
	{"web3_bridge", "Web3", "Chain Bridge", "🌉", 120000, 2.6, 6000, 2.0, 20},
	{"legal_kycaml", "Legal", "KYC/AML System", "🔏", 30000, 2.2, 1500, 1.7, 20},
	{"legal_compliance", "Legal", "Compliance Officer", "⚖️", 45000, 2.3, 2200, 1.8, 20},
	{"legal_audit", "Legal", "Security Audit", "🛡️", 60000, 2.4, 3000, 1.9, 20},
	{"team_dev", "Team", "Lead Developer", "💻", 20000, 2.1, 900, 1.6, 20},
	{"team_cto", "Team", "CTO Hire", "🧠", 100000, 2.5, 5000, 2.0, 20},
	{"team_cmo", "Team", "CMO Hire", "📈", 90000, 2.5, 4500, 1.9, 20},
}

var taskCatalog = []TaskDef{
	{"task_channel1", "Join ZX Chat Official Channel", REWARD_JOIN_CHANNEL, "telegram_join", LINK_CHANNEL, TG_CHANNEL},
	{"task_channel2", "Join Partner Candy AIO", REWARD_CHANNEL2, "telegram_join", LINK_CHANNEL2, "@CandyAIOfficial"},
	{"task_twitter", "Follow Our Strategic Twitter", REWARD_TWITTER, "social_follow", LINK_TWITTER, "ZX_Network"},
	{"task_partner", "Launch Wen Lambo Bot", REWARD_PARTNER, "bot_start", LINK_PARTNER, "wen_Lambo_1212bot"},
}

// ═══════════════════════════════════════════════════════════════════════════════
// FUNCTII HELPER CONTEXTUALE ȘI LOGICĂ DE BUSINESS
// ═══════════════════════════════════════════════════════════════════════════════

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

func getRank(balance int64) RankInfo {
	r := ranks[0]
	for _, rk := range ranks {
		if balance >= rk.Min {
			r = rk
		}
	}
	return r
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

func formatInt(n int64) string {
	return strconv.FormatInt(n, 10)
}

// ═══════════════════════════════════════════════════════════════════════════════
// IMPLEMENTARE RATE LIMITER COMPREHENSIBILĂ
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

var rateLimiter = newRateLimiter()

// ═══════════════════════════════════════════════════════════════════════════════
// STRATUL DE ACCES LA DATE (POSTGRESQL MIGRATIONS ȘI INTERAFEȚE)
// ═══════════════════════════════════════════════════════════════════════════════

var db *sql.DB

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("❌ EROARE CRITICĂ: Variabila DATABASE_URL lipsește din mediu.")
	}
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("❌ EROARE LA DESCHIDEREA DB:", err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(10 * time.Minute)
	if err = db.Ping(); err != nil {
		log.Fatal("❌ EROARE LA PING DB:", err)
	}
	migrateDB()
	log.Println("✅ Baza de date PostgreSQL conectată, optimizată și migrată cu succes.")
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
			id            SERIAL PRIMARY KEY,
			telegram_id   BIGINT NOT NULL REFERENCES players(telegram_id) ON DELETE CASCADE,
			amount        BIGINT NOT NULL,
			duration_days INT NOT NULL,
			apy           FLOAT NOT NULL,
			started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			unlocks_at    TIMESTAMPTZ NOT NULL,
			claimed       BOOLEAN NOT NULL DEFAULT FALSE
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
			log.Fatalf("❌ Eroare critică în timpul rulării migrației:\n%s\n%v", s, err)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// METODE PLAYER MANAGEMENT ȘI SINCRONIZARE ENGINE
// ═══════════════════════════════════════════════════════════════════════════════

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
			return nil, fmt.Errorf("eroare insert player nou: %w", err2)
		}
		return getOrCreatePlayer(telegramID, username, firstName, photoURL)
	}
	if err != nil {
		return nil, err
	}
	if (username != "" && username != p.Username) || (firstName != "" && firstName != p.FirstName) || (photoURL != "" && photoURL != p.PhotoURL) {
		_, _ = db.Exec(`UPDATE players SET username=$1, first_name=$2, photo_url=$3 WHERE telegram_id=$4`, username, firstName, photoURL, telegramID)
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
		p.LastBalance, p.TelegramID,
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
// VALIDARE TELEGRAM INIT DATA SECURITY ENGINE
// ═══════════════════════════════════════════════════════════════════════════════

func validateTelegramInitData(initData string) (map[string]string, bool) {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
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
	var parts []string
	for k, v := range parsed {
		if k == "hash" {
			continue
		}
		parts = append(parts, k+"="+v[0])
	}
	sort.Strings(parts)
	dataCheckString := strings.Join(parts, "\n")
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(token))
	secret := mac.Sum(nil)
	mac2 := hmac.New(sha256.New, secret)
	mac2.Write([]byte(dataCheckString))
	expected := hex.EncodeToString(mac2.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(receivedHash)) {
		return nil, false
	}
	if adStr, ok := parsed["auth_date"]; ok && len(adStr) > 0 {
		authDate, err := strconv.ParseInt(adStr[0], 10, 64)
		if err == nil {
			if time.Now().Unix()-authDate > 86400 {
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

func validateRequest(r *http.Request, telegramID int64) bool {
	initData := r.Header.Get("X-Telegram-Init-Data")
	if initData == "" {
		return os.Getenv("TELEGRAM_TOKEN") == ""
	}
	_, ok := validateTelegramInitData(initData)
	return ok
}

// ═══════════════════════════════════════════════════════════════════════════════
// LOGICA DE CALCUL VENIT PASIV ȘI SERVER SIDE ANTI-CHEAT TAP ENGINE
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

func calcMaxEnergy(energyLevel int) int64 {
	return int64(MAX_ENERGY_BASE + energyLevel*500)
}

func processServerTaps(p *Player, taps int64, clientBalance int64) (int64, int64, error) {
	if taps < 0 || taps > 10000 {
		return p.Balance, 0, fmt.Errorf("tap count is anomalous and out of physical range")
	}
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
	maxEnergy := calcMaxEnergy(p.EnergyLevel)
	if taps > maxEnergy {
		taps = maxEnergy
	}
	tapGain := int64(1 + p.TapLevel)
	earned := taps * tapGain
	passive := computePassiveEarned(p)
	p.LastPassive = now
	p.LastSync = now
	newBalance := p.Balance + earned + passive

	rows, err := db.Query(`SELECT balance, ts FROM sync_history WHERE telegram_id=$1 AND ts > $2 ORDER BY ts ASC LIMIT 60`, p.TelegramID, now.Add(-60*time.Second))
	if err == nil {
		defer rows.Close()
		type snap struct {
			bal int64
			t   time.Time
		}
		var history []snap
		for rows.Next() {
			var s snap
			_ = rows.Scan(&s.bal, &s.t)
			history = append(history, s)
		}
		if len(history) >= 2 {
			oldest := history[0]
			windowSec := now.Sub(oldest.t).Seconds()
			if windowSec > 0 {
				maxRate := float64(tapGain)*float64(MAX_TAPS_PER_SEC) + float64(passivePerHour(p))/3600.0 + 100
				maxFromWindow := oldest.bal + int64(windowSec*maxRate*1.3)
				if newBalance > maxFromWindow {
					newBalance = maxFromWindow
				}
			}
		}
	}
	p.Balance = newBalance
	p.LastBalance = newBalance
	_, _ = db.Exec(`INSERT INTO sync_history (telegram_id, balance, ts) VALUES ($1,$2,$3)`, p.TelegramID, newBalance, now)
	_, _ = db.Exec(`DELETE FROM sync_history WHERE telegram_id=$1 AND ts < $2`, p.TelegramID, now.Add(-120*time.Second))
	return newBalance, maxEnergy, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// RECEPTORI HTTP ȘI MANAGERI DE RĂSPUNS INTERNI
// ═══════════════════════════════════════════════════════════════════════════════

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Telegram-Init-Data")
}

func jsonResp(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func getIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	return r.RemoteAddr
}

// ═══════════════════════════════════════════════════════════════════════════════
// ENDPOINT-URI API ROUTING ȘI PROCESARE TRANZACȚIONALĂ
// ═══════════════════════════════════════════════════════════════════════════════

func handleSync(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Method Not Allowed", 405)
		return
	}
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
		jsonError(w, "Bad Request JSON Parsing Failed", 400)
		return
	}
	if req.TelegramID == 0 {
		jsonError(w, "Missing Parameter: Telegram ID", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Unauthorized Telegram Context Security Signature Error", 401)
		return
	}
	key := fmt.Sprintf("sync:%d", req.TelegramID)
	if !rateLimiter.Allow(key, 3) {
		jsonError(w, "Too Many Requests on Sync Endpoint", 429)
		return
	}
	p, err := getOrCreatePlayer(req.TelegramID, req.Username, req.FirstName, req.PhotoURL)
	if err != nil {
		jsonError(w, "Internal Database Failure", 500)
		return
	}
	if req.TapLevel > MAX_TAP_LEVEL {
		req.TapLevel = MAX_TAP_LEVEL
	}
	if req.EnergyLevel > MAX_ENERGY_LEVEL {
		req.EnergyLevel = MAX_ENERGY_LEVEL
	}
	if req.PassiveLevel > MAX_PASSIVE_LEVEL {
		req.PassiveLevel = MAX_PASSIVE_LEVEL
	}
	p.TapLevel = req.TapLevel
	p.EnergyLevel = req.EnergyLevel
	p.PassiveLevel = req.PassiveLevel

	newBal, maxEng, err2 := processServerTaps(p, req.Taps, p.Balance)
	if err2 == nil {
		p.Balance = newBal
	}
	passive := computePassiveEarned(p)
	p.Balance += passive
	p.LastPassive = time.Now()
	p.LastSync = time.Now()
	if err := savePlayer(p); err != nil {
		jsonError(w, "Database Integrity Persist Failure", 500)
		return
	}
	jsonResp(w, map[string]interface{}{
		"status":        "ok",
		"balance":       p.Balance,
		"passiveEarned": passive,
		"maxEnergy":     maxEng,
		"rank":          getRank(p.Balance),
	})
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	rows, err := db.Query(`SELECT telegram_id, username, first_name, photo_url, balance, level FROM players ORDER BY balance DESC LIMIT 100`)
	if err != nil {
		jsonError(w, "Database Query Error on Leaderboard Extraction", 500)
		return
	}
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
	idx := 1
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.TelegramID, &e.Username, &e.FirstName, &e.PhotoURL, &e.Balance, &e.Level); err == nil {
			e.RankName = getRank(e.Balance).Name
			e.Rank = idx
			list = append(list, e)
			idx++
		}
	}
	jsonResp(w, list)
}

func handleCheckin(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Invalid HTTP Method Type", 405)
		return
	}
	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "Bad JSON Request Parameter Parsing Context", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Unauthorized Action", 401)
		return
	}
	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil {
		jsonError(w, "Player Matrix Not Resolved In Database Engine", 500)
		return
	}
	today := time.Now().Format("2006-01-02")
	if p.LastCheckin == today {
		jsonResp(w, map[string]interface{}{
			"success": false,
			"message": "Sistemul a detectat deja un check-in efectuat pentru astazi.",
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
	if reward > 5000 {
		reward = 5000
	}
	if p.CheckinStreak == 7 {
		reward += 10000
	}
	if p.CheckinStreak == 30 {
		reward += 100000
	}
	p.Balance += reward
	p.LastBalance = p.Balance
	_ = savePlayer(p)
	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  reward,
		"streak":  p.CheckinStreak,
		"balance": p.Balance,
		"message": fmt.Sprintf("Recompensă acordată cu succes pentru ziua %d! S-au adăugat %d puncte.", p.CheckinStreak, reward),
	})
}

func handleReferral(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	tidStr := r.URL.Query().Get("telegramId")
	tid, _ := strconv.ParseInt(tidStr, 10, 64)
	if tid == 0 {
		jsonError(w, "Parameter telegramId mapping failed or missing", 400)
		return
	}
	var refCode string
	err := db.QueryRow(`SELECT referral_code FROM players WHERE telegram_id=$1`, tid).Scan(&refCode)
	if err != nil {
		jsonError(w, "Player Entity Record Absent", 404)
		return
	}
	rows, err := db.Query(`
		SELECT p.telegram_id, p.username, p.first_name, p.balance, p.level 
		FROM referrals r 
		JOIN players p ON r.referee_id = p.telegram_id 
		WHERE r.referrer_id = $1 
		ORDER BY r.created_at DESC
	`, tid)
	type RefEntry struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		FirstName  string `json:"firstName"`
		Balance    int64  `json:"balance"`
		Level      int    `json:"level"`
	}
	var list []RefEntry
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var re RefEntry
			if err := rows.Scan(&re.TelegramID, &re.Username, &re.FirstName, &re.Balance, &re.Level); err == nil {
				list = append(list, re)
			}
		}
	}
	var unclaimedCount int
	db.QueryRow(`SELECT COUNT(*) FROM referrals WHERE referrer_id=$1`, tid).Scan(&unclaimedCount)
	jsonResp(w, map[string]interface{}{
		"referralCode": refCode,
		"referralLink": fmt.Sprintf("https://t.me/ZX_Elite_Bot?start=ref_%s", refCode),
		"referrals":    list,
		"unclaimed":    unclaimedCount,
	})
}

func handleClaimReferral(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Post Strategy Context Required", 405)
		return
	}
	var req struct {
		TelegramID int64 `json:"telegramId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "Bad Context Params Mapping Exception", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Security Verification Failure", 401)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "Transaction Engine Lock Error", 500)
		return
	}
	defer tx.Rollback()
	var count int64
	err = tx.QueryRow(`SELECT COUNT(*) FROM referrals WHERE referrer_id = $1`, req.TelegramID).Scan(&count)
	if err != nil || count == 0 {
		jsonError(w, "Nu există recompense de referal disponibile pentru colectare.", 400)
		return
	}
	rewardAmount := count * 25000
	_, err = tx.Exec(`UPDATE players SET balance = balance + $1 WHERE telegram_id = $2`, rewardAmount, req.TelegramID)
	if err != nil {
		jsonError(w, "Incapabilitate la actualizarea balanței contului în baza de date", 500)
		return
	}
	// To prevent double claiming, we track or prune/mark. For simplified robust design, delete or track.
	// Since we compute from table, we clear table or log into history. Let's archive or delete rows to clean up.
	_, _ = tx.Exec(`DELETE FROM referrals WHERE referrer_id = $1`, req.TelegramID)
	if err := tx.Commit(); err != nil {
		jsonError(w, "Eroare la finalizarea tranzacției SQL Commit", 500)
		return
	}
	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  rewardAmount,
		"message": fmt.Sprintf("S-au colectat cu succes %d puncte din invitații.", rewardAmount),
	})
}

func handlePassiveInfo(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 {
		jsonError(w, "Missing Parameter", 400)
		return
	}
	p, err := getOrCreatePlayer(tid, "", "", "")
	if err != nil {
		jsonError(w, "Database Context Resolution Failure", 500)
		return
	}
	cards, _ := getPlayerCards(tid)
	totalPPH := computeTotalPPH(cards)
	p.PointsPerHour = totalPPH
	_ = savePlayer(p)
	earned := computePassiveEarned(p)
	jsonResp(w, map[string]interface{}{
		"pointsPerHour": passivePerHour(p),
		"basePph":       p.PointsPerHour,
		"passiveEarned": earned,
		"lastPassive":   p.LastPassive,
		"isPro":         p.IsPro,
	})
}

func handleTaskClaim(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Invalid Context Action Type Method", 405)
		return
	}
	var req struct {
		TelegramID int64  `json:"telegramId"`
		TaskID     string `json:"taskId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 || req.TaskID == "" {
		jsonError(w, "Invalid parameters payload provided", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Unauthorized Context Validation Error", 401)
		return
	}
	var existing int
	db.QueryRow(`SELECT COUNT(*) FROM player_tasks WHERE telegram_id=$1 AND task_id=$2`, req.TelegramID, req.TaskID).Scan(&existing)
	if existing > 0 {
		jsonError(w, "Sarcina a fost deja marcată ca fiind revendicată în trecut.", 400)
		return
	}
	var targetDef *TaskDef
	for i := range taskCatalog {
		if taskCatalog[i].ID == req.TaskID {
			targetDef = &taskCatalog[i]
			break
		}
	}
	if targetDef == nil {
		jsonError(w, "Definiția sarcinii nu a fost identificată în catalogul nativ.", 404)
		return
	}
	p, err := getOrCreatePlayer(req.TelegramID, "", "", "")
	if err != nil {
		jsonError(w, "Incapabilitate acces player context", 500)
		return
	}
	p.Balance += targetDef.Reward
	p.LastBalance = p.Balance
	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "Database Transaction Fail", 500)
		return
	}
	defer tx.Rollback()
	_, _ = tx.Exec(`INSERT INTO player_tasks (telegram_id, task_id) VALUES ($1, $2)`, req.TelegramID, req.TaskID)
	// Update user state inside transaction block
	_, _ = tx.Exec(`UPDATE players SET balance=$1, last_balance=$2 WHERE telegram_id=$3`, p.Balance, p.LastBalance, p.TelegramID)
	_ = tx.Commit()
	jsonResp(w, map[string]interface{}{
		"success": true,
		"balance": p.Balance,
		"reward":  targetDef.Reward,
		"message": fmt.Sprintf("Sarcina '%s' a fost procesată. S-au adăugat +%d monede.", targetDef.Title, targetDef.Reward),
	})
}

func handleAdsgramReward(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Method Type Exception Rule", 405)
		return
	}
	var req struct {
		TelegramID int64  `json:"telegramId"`
		Username   string `json:"username"`
		BlockID    string `json:"blockId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "Parsing Context Arguments Invalid Layout Struct", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Unauthorized Operation Core Protection", 401)
		return
	}
	if req.BlockID != ADSGRAM_BLOCK_ID {
		jsonError(w, "Corrupted Block Identifier Context Violation Mismatch", 400)
		return
	}
	p, err := getOrCreatePlayer(req.TelegramID, req.Username, "", "")
	if err != nil {
		jsonError(w, "Database Context Resolution Fault Exception", 500)
		return
	}
	p.Balance += REWARD_WATCH_AD
	p.LastBalance = p.Balance
	_ = savePlayer(p)
	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  REWARD_WATCH_AD,
		"balance": p.Balance,
		"message": fmt.Sprintf("+%d ZX adăugați în cont din vizualizarea reclamei.", REWARD_WATCH_AD),
	})
}

func handleWalletSave(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Bad Post Context Mode Matrix", 405)
		return
	}
	var req struct {
		TelegramID    int64  `json:"telegramId"`
		WalletAddress string `json:"walletAddress"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "Invalid mapping context arguments", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Security Validation Cryptographic Breach Prevented", 401)
		return
	}
	p, err := getOrCreatePlayer(req.TelegramID, "", "", "")
	if err != nil {
		jsonError(w, "Player Resolver Invariant Exception Context", 500)
		return
	}
	p.WalletAddress = strings.TrimSpace(req.WalletAddress)
	if err := savePlayer(p); err != nil {
		jsonError(w, "Failed to persist structural layout state on wallet variable", 500)
		return
	}
	jsonResp(w, map[string]interface{}{
		"success": true,
		"address": p.WalletAddress,
		"message": "Portofelul criptografic TON Connect a fost mapat corect pe profilul de utilizator.",
	})
}

func handleAppConfig(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	jsonResp(w, map[string]interface{}{
		"tgChannel":       TG_CHANNEL,
		"adsgramBlockId":  ADSGRAM_BLOCK_ID,
		"appUrl":          APP_URL,
		"rewardWatchAd":   REWARD_WATCH_AD,
		"rewardDailyCombo": REWARD_DAILY_COMBO,
		"ranks":           ranks,
		"cards":           cardCatalog,
		"tasks":           taskCatalog,
		"cipher":          getDailyCipherAnswer(),
	})
}

func handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Method Policy Blocked", 405)
		return
	}
	var req struct {
		TelegramID int64 `json:"telegramId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "Syntax Error Decoding Payload Mapping Elements", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Forbidden Security Scope Clearance Failed", 401)
		return
	}
	_, err := db.Exec(`DELETE FROM players WHERE telegram_id = $1`, req.TelegramID)
	if err != nil {
		jsonError(w, "Failed to remove player record completely from database cluster", 500)
		return
	}
	jsonResp(w, map[string]interface{}{
		"success": true,
		"message": "Toate datele asociate contului dumneavoastră au fost eliminate definitiv din nodul central.",
	})
}

func handleTonManifest(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")
	manifest := map[string]interface{}{
		"url":            APP_URL,
		"name":           "ZX Elite Premium Ecosystem Core",
		"iconUrl":        APP_URL + "/assets/images/logo_icon.png",
		"termsOfService": APP_URL + "/tos",
		"privacyPolicy":  APP_URL + "/privacy",
	}
	_ = json.NewEncoder(w).Encode(manifest)
}

// ═══════════════════════════════════════════════════════════════════════════════
// ENDPOINT-URI PENTRU CARDURI ȘI MANAGEMENTUL VENITULUI PASIV EXTRA
// ═══════════════════════════════════════════════════════════════════════════════

func handleGetCards(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 {
		jsonError(w, "Parameter exception context", 400)
		return
	}
	pCards, err := getPlayerCards(tid)
	if err != nil {
		jsonError(w, "DB Layer Failure", 500)
		return
	}
	ownedMap := make(map[string]int)
	for _, c := range pCards {
		ownedMap[c.CardID] = c.Level
	}
	type CardState struct {
		Def          CardDef `json:"def"`
		CurrentLevel int     `json:"currentLevel"`
		NextPrice    int64   `json:"nextPrice"`
		NextPPH      int64   `json:"nextPph"`
		CanBuy       bool    `json:"canBuy"`
	}
	var response []CardState
	for _, d := range cardCatalog {
		lvl := ownedMap[d.ID]
		price := cardPrice(&d, lvl)
		nextPph := cardPPH(&d, lvl+1)
		response = append(response, CardState{
			Def:          d,
			CurrentLevel: lvl,
			NextPrice:    price,
			NextPPH:      nextPph,
			CanBuy:       lvl < d.MaxLevel,
		})
	}
	jsonResp(w, response)
}

func handleBuyCard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Forbidden Strategy Action Exception Routing", 405)
		return
	}
	var req struct {
		TelegramID int64  `json:"telegramId"`
		CardID     string `json:"cardId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 || req.CardID == "" {
		jsonError(w, "Invalid Structural Arguments Context JSON Definition", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Cryptographic context validation failed", 401)
		return
	}
	def := getCardDef(req.CardID)
	if def == nil {
		jsonError(w, "Card ID target non-existent in active dynamic catalog map", 404)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "Database engine transaction isolation lock failed", 500)
		return
	}
	defer tx.Rollback()
	p := &Player{}
	err = tx.QueryRow(`
		SELECT telegram_id, balance, points_per_hour, tap_level, energy_level, passive_level, last_passive 
		FROM players WHERE telegram_id = $1 FOR UPDATE
	`, req.TelegramID).Scan(&p.TelegramID, &p.Balance, &p.PointsPerHour, &p.TapLevel, &p.EnergyLevel, &p.PassiveLevel, &p.LastPassive)
	if err != nil {
		jsonError(w, "Failed to isolate context row lock for target execution", 500)
		return
	}
	var currentLevel int
	err = tx.QueryRow(`SELECT level FROM player_cards WHERE telegram_id=$1 AND card_id=$2`, req.TelegramID, req.CardID).Scan(&currentLevel)
	if err == sql.ErrNoRows {
		currentLevel = 0
	}
	if currentLevel >= def.MaxLevel {
		jsonError(w, "Cardul a atins nivelul maxim de evoluție permis structural.", 400)
		return
	}
	price := cardPrice(def, currentLevel)
	if p.Balance < price {
		jsonError(w, "Fonduri insuficiente în balanța contului pentru achiziție.", 400)
		return
	}
	p.Balance -= price
	p.LastBalance = p.Balance
	currentLevel++
	if currentLevel == 1 {
		_, _ = tx.Exec(`INSERT INTO player_cards (telegram_id, card_id, level) VALUES ($1, $2, $3)`, p.TelegramID, req.CardID, currentLevel)
	} else {
		_, _ = tx.Exec(`UPDATE player_cards SET level=$1 WHERE telegram_id=$2 AND card_id=$3`, currentLevel, p.TelegramID, req.CardID)
	}
	var allCards []PlayerCard
	rows, _ := tx.Query(`SELECT card_id, level FROM player_cards WHERE telegram_id=$1`, p.TelegramID)
	for rows.Next() {
		var ac PlayerCard
		_ = rows.Scan(&ac.CardID, &ac.Level)
		if ac.CardID == req.CardID {
			ac.Level = currentLevel
		}
		allCards = append(allCards, ac)
	}
	rows.Close()
	p.PointsPerHour = computeTotalPPH(allCards)
	p.Level = getRank(p.Balance).Level
	_, err = tx.Exec(`
		UPDATE players SET balance=$1, last_balance=$2, points_per_hour=$3, level=$4 WHERE telegram_id=$5
	`, p.Balance, p.LastBalance, p.PointsPerHour, p.Level, p.TelegramID)
	if err != nil {
		jsonError(w, "Incapabilitate structurală la salvarea stării interne în tranzacție", 500)
		return
	}
	if err := tx.Commit(); err != nil {
		jsonError(w, "Transaction commit layer fault exception", 500)
		return
	}
	jsonResp(w, map[string]interface{}{
		"success":       true,
		"newLevel":      currentLevel,
		"balance":       p.Balance,
		"pointsPerHour": passivePerHour(p),
		"message":       fmt.Sprintf("Cardul '%s' a fost îmbunătățit la nivelul %d cu succes.", def.Name, currentLevel),
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// LOGICA LUCRULUI CU STAKING ENGINE
// ═══════════════════════════════════════════════════════════════════════════════

func handleStakeCreate(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Post strategy structural handler required", 405)
		return
	}
	var req struct {
		TelegramID   int64 `json:"telegramId"`
		Amount       int64 `json:"amount"`
		DurationDays int   `json:"durationDays"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 || req.Amount <= 0 {
		jsonError(w, "Invalid structural JSON serialization mapping configuration context", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Security token tracking signature parsing rejected", 401)
		return
	}
	var apy float64
	switch req.DurationDays {
	case 7:
		apy = STAKE_7D_APY
	case 30:
		apy = STAKE_30D_APY
	case 90:
		apy = STAKE_90D_APY
	default:
		jsonError(w, "Durata de staking specificată este invalidă structural (trebuie să fie de 7, 30 sau 90 de zile).", 400)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "Transaction initiation failed", 500)
		return
	}
	defer tx.Rollback()
	var bal int64
	err = tx.QueryRow(`SELECT balance FROM players WHERE telegram_id=$1 FOR UPDATE`, req.TelegramID).Scan(&bal)
	if err != nil {
		jsonError(w, "Player query isolation fault context exception", 404)
		return
	}
	if bal < req.Amount {
		jsonError(w, "Fonduri insuficiente pentru a iniția contractul inteligent de staking.", 400)
		return
	}
	now := time.Now()
	unlocks := now.AddDate(0, 0, req.DurationDays)
	_, _ = tx.Exec(`UPDATE players SET balance=balance-$1, last_balance=balance-$1 WHERE telegram_id=$2`, req.Amount, req.TelegramID)
	_, err = tx.Exec(`
		INSERT INTO stakes (telegram_id, amount, duration_days, apy, started_at, unlocks_at, claimed)
		VALUES ($1, $2, $3, $4, $5, $6, FALSE)
	`, req.TelegramID, req.Amount, req.DurationDays, apy, now, unlocks)
	if err != nil {
		jsonError(w, "Eroare la inserarea înregistrării contractului de staking.", 500)
		return
	}
	_ = tx.Commit()
	jsonResp(w, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Contract de staking deschis cu succes pentru %d monede pe o perioadă de %d zile.", req.Amount, req.DurationDays),
	})
}

func handleStakeList(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 {
		jsonError(w, "Missing Parameter Context Exception Handler Map", 400)
		return
	}
	rows, err := db.Query(`SELECT id, telegram_id, amount, duration_days, apy, started_at, unlocks_at, claimed FROM stakes WHERE telegram_id=$1 ORDER BY id DESC`, tid)
	if err != nil {
		jsonError(w, "Staking collection parsing db failure context", 500)
		return
	}
	defer rows.Close()
	var stakesList []Stake
	for rows.Next() {
		var s Stake
		if err := rows.Scan(&s.ID, &s.TelegramID, &s.Amount, &s.DurationDays, &s.APY, &s.StartedAt, &s.UnlocksAt, &s.Claimed); err == nil {
			stakesList = append(stakesList, s)
		}
	}
	jsonResp(w, stakesList)
}

func handleStakeClaim(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Invalid Strategy Mapping Mode Routing Context", 405)
		return
	}
	var req struct {
		TelegramID int64 `json:"telegramId"`
		StakeID    int   `json:"stakeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "Arguments structural mapping failure logic check", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Security check validation rejected context token tracking", 401)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "Transaction allocation mapping logic failure inside structural database block", 500)
		return
	}
	defer tx.Rollback()
	var s Stake
	err = tx.QueryRow(`
		SELECT id, telegram_id, amount, duration_days, apy, unlocks_at, claimed 
		FROM stakes WHERE id=$1 AND telegram_id=$2 FOR UPDATE
	`, req.StakeID, req.TelegramID).Scan(&s.ID, &s.TelegramID, &s.Amount, &s.DurationDays, &s.APY, &s.UnlocksAt, &s.Claimed)
	if err != nil {
		jsonError(w, "Contractul de staking specificat nu a fost identificat sau nu aparține utilizatorului.", 404)
		return
	}
	if s.Claimed {
		jsonError(w, "Această recompensă de staking a fost deja colectată anterior.", 400)
		return
	}
	if time.Now().Before(s.UnlocksAt) {
		jsonError(w, fmt.Sprintf("Fondurile sunt blocate criptografic până la data de: %s", s.UnlocksAt.Format(time.RFC1123)), 400)
		return
	}
	rewardInterest := float64(s.Amount) * s.APY * (float64(s.DurationDays) / 365.0)
	totalReturn := s.Amount + int64(math.Round(rewardInterest))
	_, _ = tx.Exec(`UPDATE players SET balance=balance+$1, last_balance=balance+$1 WHERE telegram_id=$2`, totalReturn, req.TelegramID)
	_, _ = tx.Exec(`UPDATE stakes SET claimed=TRUE WHERE id=$1`, s.ID)
	_ = tx.Commit()
	jsonResp(w, map[string]interface{}{
		"success": true,
		"reward":  totalReturn,
		"message": fmt.Sprintf("Contractul a expirat. S-au deblocat și transferat %d monede în balanța contului dumneavoastră.", totalReturn),
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// COMBINAȚIA ZILNICĂ (DAILY COMBO) ȘI GHICITOAREA ROATĂ (WHEEL ENGINE)
// ═══════════════════════════════════════════════════════════════════════════════

func handleDailyCombo(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Invalid HTTP Execution Strategy", 405)
		return
	}
	var req struct {
		TelegramID int64    `json:"telegramId"`
		CardIDs    []string `json:"cardIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 || len(req.CardIDs) != 3 {
		jsonError(w, "Trebuie selectate exact 3 carduri pentru validarea combo-ului.", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Context Validation Core Rejection Signature Trace", 401)
		return
	}
	p, err := getOrCreatePlayer(req.TelegramID, "", "", "")
	if err != nil {
		jsonError(w, "DB Fetch Core Struct Fault Error Context", 500)
		return
	}
	today := time.Now().Format("2006-01-02")
	if p.DailyComboClaimed == today {
		jsonError(w, "Ați revendicat deja recompensa pentru combo-ul zilei de astăzi.", 400)
		return
	}
	correctCards := getDailyComboCards()
	sort.Strings(correctCards)
	sort.Strings(req.CardIDs)
	match := true
	for i := range correctCards {
		if correctCards[i] != req.CardIDs[i] {
			match = false
			break
		}
	}
	if !match {
		jsonResp(w, map[string]interface{}{
			"success": false,
			"message": "Combinația de carduri selectată este incorectă. Încercați alte departamente corporative.",
		})
		return
	}
	p.Balance += REWARD_DAILY_COMBO
	p.DailyComboClaimed = today
	p.LastBalance = p.Balance
	_ = savePlayer(p)
	jsonResp(w, map[string]interface{}{
		"success": true,
		"balance": p.Balance,
		"reward":  REWARD_DAILY_COMBO,
		"message": fmt.Sprintf("Felicitări! Ați descoperit combo-ul secret al zilei. S-au adăugat +%d monede.", REWARD_DAILY_COMBO),
	})
}

func handleComboStatus(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 {
		jsonError(w, "Missing mapping identity context argument identifier key", 400)
		return
	}
	var claimedToday string
	_ = db.QueryRow(`SELECT daily_combo_claimed FROM players WHERE telegram_id=$1`, tid).Scan(&claimedToday)
	today := time.Now().Format("2006-01-02")
	jsonResp(w, map[string]interface{}{
		"claimed": claimedToday == today,
		"date":    today,
	})
}

func handleWheelSpin(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Method Type Context Action Blocked Strategy", 405)
		return
	}
	var req struct {
		TelegramID int64 `json:"telegramId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "Bad Mapping Context Serialization Pointer JSON Variable", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Security protection token invalid signature checksum tracking", 401)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "Transaction Allocation Fault Segment Core Node", 500)
		return
	}
	defer tx.Rollback()
	p := &Player{}
	err = tx.QueryRow(`SELECT telegram_id, balance, spin_cooldown FROM players WHERE telegram_id=$1 FOR UPDATE`, req.TelegramID).Scan(&p.TelegramID, &p.Balance, &p.SpinCooldown)
	if err != nil {
		jsonError(w, "Player Matrix Resolution Record Internal Isolation Failure Context Map", 404)
		return
	}
	now := time.Now()
	if p.SpinCooldown != nil && now.Before(*p.SpinCooldown) {
		jsonError(w, fmt.Sprintf("Roata Norocului este în perioada de reîncărcare energetică. Revine la: %s", p.SpinCooldown.Format(time.RFC1123)), 400)
		return
	}
	rewardsPool := []int64{2500, 5000, 10000, 25000, 50000, 100000, 250000, 500000}
	rng := rand.New(rand.NewSource(now.UnixNano() + p.TelegramID))
	chosenReward := rewardsPool[rng.Intn(len(rewardsPool))]
	p.Balance += chosenReward
	p.LastBalance = p.Balance
	cooldown := now.Add(WHEEL_COOLDOWN_H * time.Hour)
	p.SpinCooldown = &cooldown
	_, err = tx.Exec(`UPDATE players SET balance=$1, last_balance=$2, spin_cooldown=$3 WHERE telegram_id=$4`, p.Balance, p.LastBalance, p.SpinCooldown, p.TelegramID)
	if err != nil {
		jsonError(w, "Database Persist Core Isolation Tranzact Block Error Map Structure", 500)
		return
	}
	_ = tx.Commit()
	jsonResp(w, map[string]interface{}{
		"success":      true,
		"reward":       chosenReward,
		"balance":      p.Balance,
		"nextSpinTime": cooldown,
		"message":      fmt.Sprintf("Roata s-a oprit pe sectorul norocos! Recompensa primita: +%d puncte.", chosenReward),
	})
}

func handleWheelStatus(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	tid, _ := strconv.ParseInt(r.URL.Query().Get("telegramId"), 10, 64)
	if tid == 0 {
		jsonError(w, "Missing tracking elements identifier payload reference context", 400)
		return
	}
	var spinCooldown *time.Time
	_ = db.QueryRow(`SELECT spin_cooldown FROM players WHERE telegram_id=$1`, tid).Scan(&spinCooldown)
	canSpin := true
	var secLeft float64 = 0
	if spinCooldown != nil && time.Now().Before(*spinCooldown) {
		canSpin = false
		secLeft = time.Until(*spinCooldown).Seconds()
	}
	jsonResp(w, map[string]interface{}{
		"canSpin":      canSpin,
		"secondsLeft":  int(secLeft),
		"spinCooldown": spinCooldown,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// ACTIVARE SUBSCRIPȚIE PRO (PRO UPGRADE SUBSYSTEM)
// ═══════════════════════════════════════════════════════════════════════════════

func handleProActivate(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Invalid strategy layout action method routing rule context", 405)
		return
	}
	var req struct {
		TelegramID int64 `json:"telegramId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TelegramID == 0 {
		jsonError(w, "Missing payload matrix fields mapped identifiers context execution", 400)
		return
	}
	if !validateRequest(r, req.TelegramID) {
		jsonError(w, "Unauthorized cryptographic authorization layer violation detected", 401)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		jsonError(w, "Transaction failed to initiate context model layer block node", 500)
		return
	}
	defer tx.Rollback()
	var bal int64
	err = tx.QueryRow(`SELECT balance FROM players WHERE telegram_id=$1 FOR UPDATE`, req.TelegramID).Scan(&bal)
	if err != nil {
		jsonError(w, "User tracking data record row mapping failure exception layer", 404)
		return
	}
	proCost := int64(1500000) // 1.5M Coins cost activation rule
	if bal < proCost {
		jsonError(w, "Fonduri insuficiente pentru a activa statutul ZX Elite Pro Premium (Cost: 1,500,000 monede).", 400)
		return
	}
	expires := time.Now().AddDate(0, 1, 0) // 30 zile valabilitate
	_, _ = tx.Exec(`UPDATE players SET balance=balance-$1, last_balance=balance-$1, is_pro=TRUE, pro_expires=$2 WHERE telegram_id=$3`, proCost, expires, req.TelegramID)
	_ = tx.Commit()
	jsonResp(w, map[string]interface{}{
		"success":    true,
		"proExpires": expires,
		"message":    "Statutul premium ZX Elite Core PRO a fost activat corect. Multiplicatorul de venit pasiv x2 este acum activ corporativ.",
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// TELEGRAM BOT WEBHOOK GATEWAY CONTROLLER ENGINE
// ═══════════════════════════════════════════════════════════════════════════════

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	if bot == nil {
		w.WriteHeader(503)
		return
	}
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		w.WriteHeader(400)
		return
	}
	if update.Message != nil && update.Message.Text != "" {
		txt := update.Message.Text
		chatID := update.Message.Chat.ID
		userID := update.Message.From.ID
		username := update.Message.From.UserName
		firstName := update.Message.From.FirstName

		if strings.HasPrefix(txt, "/start") {
			p, err := getOrCreatePlayer(userID, username, firstName, "")
			if err == nil {
				parts := strings.Split(txt, " ")
				if len(parts) > 1 && strings.HasPrefix(parts[1], "ref_") {
					code := strings.TrimPrefix(parts[1], "ref_")
					if code != p.ReferralCode && p.ReferredBy == 0 {
						var referrerID BIGINT
						errRef := db.QueryRow(`SELECT telegram_id FROM players WHERE referral_code=$1`, code).Scan(&referrerID)
						if errRef == nil && referrerID != userID {
							p.ReferredBy = referrerID
							_, _ = db.Exec(`UPDATE players SET referred_by=$1 WHERE telegram_id=$2`, referrerID, userID)
							_, _ = db.Exec(`INSERT INTO referrals (referrer_id, referee_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, referrerID, userID)
							// Recompensă instant de bun venit pentru ambii
							_, _ = db.Exec(`UPDATE players SET balance=balance+25000 WHERE telegram_id=$1`, referrerID)
							_, _ = db.Exec(`UPDATE players SET balance=balance+10000 WHERE telegram_id=$1`, userID)
						}
					}
				}
			}
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("👋 Bun venit în ZX Elite Core Network, %s!\n\nAici îți poți dezvolta propria corporație Web3, accesa mecanisme de Staking avansate, Roata Norocului zilnică și optimizări corporative de ultimă generație.\n\n🚀 Lansează Aplicația Web direct din meniul de jos sau accesează controlul panoului administrativ central.", firstName))
			inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("🚀 Deschide Aplicația Core MiniApp", APP_URL),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("📢 Alătură-te Canalului ZX Official", LINK_CHANNEL),
				),
			)
			msg.ReplyMarkup = inlineKeyboard
			_, _ = bot.Send(msg)
		} else if strings.HasPrefix(txt, "/status") {
			p, err := getOrCreatePlayer(userID, username, firstName, "")
			if err == nil {
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📊 Raport de Stare Corporativă:\n\n• Balanță Curentă: %d ZX\n• Venit Pasiv: %d ZX/oră\n• Nivel Companie: %d (%s)\n• Portofel Mapat: %s\n• Calitate Premium PRO: %t", p.Balance, passivePerHour(p), p.Level, getRank(p.Balance).Name, p.WalletAddress, p.IsPro))
				_, _ = bot.Send(msg)
			}
		}
	}
	w.WriteHeader(200)
}

// ═══════════════════════════════════════════════════════════════════════════════
// INTERFAȚA WEB ADMINISTRATIVĂ EMBEDDED VIEW (FULL HTML/CSS DASHBOARD CONTROLLER)
// ═══════════════════════════════════════════════════════════════════════════════

func handleIndexView(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var totalUsers int
	var totalBalance int64
	var activeStakes int
	_ = db.QueryRow(`SELECT COUNT(*) FROM players`).Scan(&totalUsers)
	_ = db.QueryRow(`SELECT COALESCE(SUM(balance), 0) FROM players`).Scan(&totalBalance)
	_ = db.QueryRow(`SELECT COUNT(*) FROM stakes WHERE claimed=FALSE`).Scan(&activeStakes)

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="ro">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ZX Elite Core - Central Business Unit Control Dashboard Panel Terminal</title>
    <style>
        :root {
            --bg-main: #0B0F19;
            --bg-card: #151D30;
            --accent: #3B82F6;
            --accent-success: #10B981;
            --text-color: #F3F4F6;
            --text-dim: #9CA3AF;
            --border-color: #24324F;
        }
        body {
            background-color: var(--bg-main);
            color: var(--text-color);
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 0;
            display: flex;
            flex-direction: column;
            min-height: 100vh;
        }
        header {
            background-color: var(--bg-card);
            padding: 20px 40px;
            border-bottom: 1px solid var(--border-color);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        h1 { margin: 0; font-size: 24px; font-weight: 700; color: var(--text-color); letter-spacing: 0.5px; }
        .badge { background-color: var(--accent); color: white; padding: 4px 12px; border-radius: 12px; font-size: 12px; }
        main { flex: 1; padding: 4px 40px; display: flex; flex-direction: column; gap: 30px; margin-top: 20px; }
        .grid-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 20px; }
        .card-stat { background-color: var(--bg-card); border: 1px solid var(--border-color); border-radius: 8px; padding: 24px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1); }
        .card-stat .title { font-size: 14px; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.5px; }
        .card-stat .value { font-size: 32px; font-weight: 700; margin-top: 10px; color: var(--text-color); }
        .card-stat .sub { font-size: 12px; margin-top: 5px; color: var(--accent-success); }
        .table-section { background-color: var(--bg-card); border: 1px solid var(--border-color); border-radius: 8px; padding: 24px; display: flex; flex-direction: column; gap: 15px; }
        .table-section h2 { margin: 0; font-size: 18px; font-weight: 600; color: var(--text-color); }
        table { width: 100%%; border-collapse: collapse; text-align: left; margin-top: 10px; }
        th { padding: 12px 16px; border-bottom: 2px solid var(--border-color); color: var(--text-dim); font-size: 14px; text-transform: uppercase; }
        td { padding: 14px 16px; border-bottom: 1px solid var(--border-color); color: var(--text-color); font-size: 14px; }
        tr:hover { background-color: rgba(59, 130, 246, 0.05); }
        footer { padding: 20px 40px; text-align: center; font-size: 12px; color: var(--text-dim); border-top: 1px solid var(--border-color); margin-top: auto; }
    </style>
</head>
<body>
    <header>
        <h1>ZX Elite Premium Core System Dashboard</h1>
        <div class="badge">Operational Node - Online</div>
    </header>
    <main>
        <div class="grid-stats">
            <div class="card-stat">
                <div class="title">Total Utilizatori Înregistrați</div>
                <div class="value">%d</div>
                <div class="sub">Metrice cluster principal</div>
            </div>
            <div class="card-stat">
                <div class="title">Balanță Totală Ecosistem</div>
                <div class="value">%d ZX</div>
                <div class="sub">Lichiditate virtuală generată</div>
            </div>
            <div class="card-stat">
                <div class="title">Contracte Active de Staking</div>
                <div class="value">%d</div>
                <div class="sub">Mecanisme locked active</div>
            </div>
        </div>
        <div class="table-section">
            <h2>Top 5 Companii Corporative active în nod</h2>
            <table>
                <thead>
                    <tr>
                        <th>ID Client</th>
                        <th>Nume Utilizator</th>
                        <th>Balanță Cont (ZX)</th>
                        <th>Nivel Companie</th>
                    </tr>
                </thead>
                <tbody>`, totalUsers, totalBalance, activeStakes)

	rows, err := db.Query(`SELECT telegram_id, username, balance, level FROM players ORDER BY balance DESC LIMIT 5`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tid int64
			var uname string
			var bal int64
			var lvl int
			if err := rows.Scan(&tid, &uname, &bal, &lvl); err == nil {
				if uname == "" {
					uname = "Anonymous Player"
				} else {
					uname = "@" + uname
				}
				htmlContent += fmt.Sprintf(`
                    <tr>
                        <td>%d</td>
                        <td>%s</td>
                        <td>%d</td>
                        <td>Nivel %d (%s)</td>
                    </tr>`, tid, uname, bal, lvl, getRank(bal).Name)
			}
		}
	}

	htmlContent += `
                </tbody>
            </table>
        </div>
    </main>
    <footer>
        &copy; 2026 J.A.R.V.I.S. Ecosystem Intelligence Unit. Toate drepturile rezervate structural. Operat prin Render.
    </footer>
</body>
</html>`
	_, _ = io.WriteString(w, htmlContent)
}

// ═══════════════════════════════════════════════════════════════════════════════
// METODA PRINCIPALĂ MAIN - INIȚIALIZARE ȘI CONFIGURARE COMPONENTĂ HTTP SERVER
// ═══════════════════════════════════════════════════════════════════════════════

func main() {
	log.Println("⚡ Inițializare J.A.R.V.I.S. Web Engine Central Operations System Core...")
	initDB()

	botToken := os.Getenv("TELEGRAM_TOKEN")
	if botToken != "" {
		var errBot error
		bot, errBot = tgbotapi.NewBotAPI(botToken)
		if errBot == nil {
			log.Printf("✅ Telegram Bot verificat și autorizat în nod: @%s", bot.Self.UserName)
			webhookURL := os.Getenv("WEBHOOK_URL")
			if webhookURL != "" {
				wh, errWh := tgbotapi.NewWebhook(webhookURL + "/webhook")
				if errWh == nil {
					_, errRequest := bot.Request(wh)
					if errRequest == nil {
						log.Println("✅ Telegram Webhook setat cu succes spre:", webhookURL+"/webhook")
					}
				}
			}
		} else {
			log.Println("⚠️ Avertisment: Initializarea Botului a esuat structural:", errBot)
		}
	} else {
		log.Println("⚠️ Avertisment: Variabila TELEGRAM_TOKEN nu este setată. Botul rulează dezactivat.")
	}

	mux := http.NewServeMux()

	// Înregistrare rutare endpoint-uri API solicitate explicit
	mux.HandleFunc("/api/sync", handleSync)
	mux.HandleFunc("/api/leaderboard", handleLeaderboard)
	mux.HandleFunc("/api/checkin", handleCheckin)
	mux.HandleFunc("/api/referral", handleReferral)
	mux.HandleFunc("/api/referral/claim", handleClaimReferral)
	mux.HandleFunc("/api/passive", handlePassiveInfo)
	mux.HandleFunc("/api/task/claim", handleTaskClaim)
	mux.HandleFunc("/api/ad-reward", handleAdsgramReward)
	mux.HandleFunc("/api/wallet/save", handleWalletSave)
	mux.HandleFunc("/api/config", handleAppConfig)
	mux.HandleFunc("/api/account/delete", handleDeleteAccount)

	// Carduri și extinderi corporative
	mux.HandleFunc("/api/cards", handleGetCards)
	mux.HandleFunc("/api/cards/buy", handleBuyCard)

	// Roata norocului (Wheel System)
	mux.HandleFunc("/api/wheel/spin", handleWheelSpin)
	mux.HandleFunc("/api/wheel/status", handleWheelStatus)

	// Combo zilnic (Daily Combo System)
	mux.HandleFunc("/api/combo/claim", handleDailyCombo)
	mux.HandleFunc("/api/combo/status", handleComboStatus)

	// Mecanismul de Staking
	mux.HandleFunc("/api/stake/create", handleStakeCreate)
	mux.HandleFunc("/api/stake/list", handleStakeList)
	mux.HandleFunc("/api/stake/claim", handleStakeClaim)

	// Extensie Premium upgrade
	mux.HandleFunc("/api/pro/activate", handleProActivate)

	// Manifest Web3 TON Connect și Webhook gateway
	mux.HandleFunc("/tonconnect-manifest.json", handleTonManifest)
	mux.HandleFunc("/webhook", handleWebhook)

	// Interfața de control vizuală principală
	mux.HandleFunc("/", handleIndexView)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Serverul rulează cu succes pe portul global: %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal("❌ EROARE CRITICĂ LA PORNIREA SERVERULUI HTTP:", err)
	}
}
