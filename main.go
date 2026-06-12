package main

import (
	"encoding/json"
	"fmt"
	"io"
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

// ═════════════════════════════════════════════════════════════════════════════
// ██  CONFIG — ZX ELITE CORE ██
// ═════════════════════════════════════════════════════════════════════════════

const (
	TG_CHANNEL       = "@ZXchatofficial"
	ADSGRAM_BLOCK_ID = "34749"
	APP_URL          = "https://zx-elite-core.onrender.com"

	REWARD_WATCH_AD      int64 = 1000
	REWARD_JOIN_CHANNEL  int64 = 10000
	REWARD_JOIN_CHANNEL2 int64 = 1500
	REWARD_TWITTER       int64 = 5000
	REWARD_PARTNER_BOT   int64 = 20000
	REWARD_DAILY_COMBO   int64 = 5000000 // 5.000.000 ZX

	LINK_CHANNEL  = "https://t.me/Swordstarsibot?start=_tgr_6ZeW5DBkNTli"
	LINK_CHANNEL2 = "https://t.me/CandyAIOfficialbot?start=_tgr_92TSa084ODcy"
	LINK_TWITTER  = "https://t.me/StarsiFotBot?start=_tgr_KvKAi-5hZDQy"
	LINK_PARTNER  = "https://t.me/wen_Lambo_1212bot?start=_tgr_fBveixRhNzQy"

	TG_CHANNEL2 = ""

	MAX_TAP_LEVEL     = 20
	MAX_ENERGY_LEVEL  = 20
	MAX_PASSIVE_LEVEL = 20
)

// ═════════════════════════════════════════════════════════════════════════════
// DATA STRUCTURES
// ═════════════════════════════════════════════════════════════════════════════

type CardConfig struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	BaseCost       int64   `json:"baseCost"`
	CostMultiplier float64 `json:"costMultiplier"`
	ProfitPerHour  int64   `json:"profitPerHour"`
	Icon           string  `json:"icon"`
}

type PlayerState struct {
	Username           string         `json:"username"`
	FirstName          string         `json:"firstName"`
	PhotoURL           string         `json:"photoUrl"`
	Balance            int64          `json:"balance"`
	TapLevel           int            `json:"tapLevel"`
	EnergyLevel        int            `json:"energyLevel"`
	LastSync           time.Time      `json:"lastSync"`
	LastBalance        int64          `json:"lastBalance"`
	LastCheckin        string         `json:"lastCheckin"`
	CheckinStreak      int            `json:"checkinStreak"`
	ReferralCode       string         `json:"referralCode"`
	ReferredBy         string         `json:"referredBy"`
	Referrals          []string       `json:"referrals"`
	LastPassive        time.Time      `json:"lastPassive"`
	PassiveLevel       int            `json:"passiveLevel"`
	OwnedCards         map[string]int `json:"ownedCards"`         // ID -> Level
	LastDailyComboDate string         `json:"lastDailyComboDate"` // YYYY-MM-DD
	SyncHistory        []SyncRecord   `json:"syncHistory"`
	TelegramID         int64          `json:"telegramId"`
	ClaimedTasks       []string       `json:"claimedTasks"`
	WalletAddress      string         `json:"walletAddress"`
}

type SyncRecord struct {
	T   time.Time `json:"t"`
	Bal int64     `json:"bal"`
}

type Database struct {
	Players map[string]*PlayerState `json:"players"`
	mu      sync.RWMutex
}

// ═════════════════════════════════════════════════════════════════════════════
// GLOBAL STATE & CONFIG
// ═════════════════════════════════════════════════════════════════════════════

var (
	db        *Database
	dbPath    = "database.json"
	saveTimer *time.Timer
	saveMu    sync.Mutex
	bot       *tgbotapi.BotAPI

	Cards = []CardConfig{
		{ID: "marketing", Name: "Marketing", BaseCost: 1000, CostMultiplier: 1.5, ProfitPerHour: 100, Icon: "📢"},
		{ID: "legal", Name: "Legal", BaseCost: 5000, CostMultiplier: 1.6, ProfitPerHour: 400, Icon: "⚖️"},
		{ID: "web3", Name: "Web3 Dev", BaseCost: 15000, CostMultiplier: 1.7, ProfitPerHour: 1100, Icon: "💻"},
		{ID: "security", Name: "Security", BaseCost: 40000, CostMultiplier: 1.8, ProfitPerHour: 3500, Icon: "🛡️"},
		{ID: "exchange", Name: "Exchange", BaseCost: 100000, CostMultiplier: 1.9, ProfitPerHour: 10000, Icon: "🏦"},
		{ID: "license", Name: "License", BaseCost: 250000, CostMultiplier: 2.0, ProfitPerHour: 28000, Icon: "📜"},
	}

	// Combo-ul zilei (poate fi setat manual sau randomizat)
	DailyComboIDs = []string{"marketing", "web3", "security"}
)

// ═════════════════════════════════════════════════════════════════════════════
// DATABASE
// ═════════════════════════════════════════════════════════════════════════════

func newDatabase() *Database {
	return &Database{Players: make(map[string]*PlayerState)}
}

func loadDatabase() *Database {
	data, err := os.ReadFile(dbPath)
	if err != nil {
		return newDatabase()
	}
	d := newDatabase()
	json.Unmarshal(data, d)
	return d
}

func saveDatabase() {
	db.mu.RLock()
	data, _ := json.MarshalIndent(db, "", "  ")
	db.mu.RUnlock()
	os.WriteFile(dbPath, data, 0644)
}

func scheduleSave() {
	saveMu.Lock()
	defer saveMu.Unlock()
	if saveTimer != nil {
		saveTimer.Stop()
	}
	saveTimer = time.AfterFunc(5*time.Second, saveDatabase)
}

func startPeriodicSave() {
	go func() {
		for range time.NewTicker(60 * time.Second).C {
			saveDatabase()
		}
	}()
}

// ═════════════════════════════════════════════════════════════════════════════
// PLAYER HELPERS
// ═════════════════════════════════════════════════════════════════════════════

func getOrCreatePlayer(username string, telegramID int64) *PlayerState {
	p, ok := db.Players[username]
	if !ok {
		p = &PlayerState{
			Username:     username,
			LastSync:     time.Now(),
			LastPassive:  time.Now(),
			Referrals:    []string{},
			SyncHistory:  []SyncRecord{},
			ClaimedTasks: []string{},
			OwnedCards:   make(map[string]int),
			TelegramID:   telegramID,
		}
		p.ReferralCode = "ZX" + strconv.FormatInt(time.Now().UnixNano()%100000, 10)
		db.Players[username] = p
	}
	if p.OwnedCards == nil {
		p.OwnedCards = make(map[string]int)
	}
	return p
}

// ═════════════════════════════════════════════════════════════════════════════
// PASSIVE INCOME (MODIFICATĂ)
// ═════════════════════════════════════════════════════════════════════════════

func getPassivePerHour(p *PlayerState) int64 {
	// Profit de bază din PassiveLevel
	var total int64 = 0
	if p.PassiveLevel > 0 {
		lvl := p.PassiveLevel
		if lvl > MAX_PASSIVE_LEVEL {
			lvl = MAX_PASSIVE_LEVEL
		}
		total = int64(math.Round(100 * math.Pow(1.8, float64(lvl-1))))
	}

	// Profit din Carduri
	for id, lvl := range p.OwnedCards {
		for _, cfg := range Cards {
			if cfg.ID == id {
				total += cfg.ProfitPerHour * int64(lvl)
			}
		}
	}
	return total
}

func computePassiveEarned(p *PlayerState) int64 {
	pph := getPassivePerHour(p)
	if pph == 0 {
		return 0
	}
	elapsed := time.Since(p.LastPassive)
	if elapsed > 8*time.Hour {
		elapsed = 8 * time.Hour
	}
	return int64(elapsed.Hours() * float64(pph))
}

// ═════════════════════════════════════════════════════════════════════════════
// HANDLERS API
// ═════════════════════════════════════════════════════════════════════════════

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func jsonResp(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct {
		Username     string `json:"username"`
		Balance      int64  `json:"balance"`
		TapLevel     int    `json:"tapLevel"`
		PassiveLevel int    `json:"passiveLevel"`
		TelegramID   int64  `json:"telegramId"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Username == "" || req.Username == "guest" {
		return
	}

	db.mu.Lock()
	p := getOrCreatePlayer(req.Username, req.TelegramID)
	earned := computePassiveEarned(p)
	p.LastPassive = time.Now()
	p.Balance = req.Balance + earned
	p.TapLevel = req.TapLevel
	p.PassiveLevel = req.PassiveLevel
	db.mu.Unlock()
	scheduleSave()

	jsonResp(w, map[string]interface{}{"status": "ok", "balance": p.Balance, "passiveEarned": earned})
}

// --- NOILE HANDLERE PENTRU CARDURI ---

func handleGetCards(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	username := r.URL.Query().Get("username")
	db.mu.RLock()
	p := db.Players[username]
	db.mu.RUnlock()

	if p == nil {
		http.Error(w, "Not found", 404)
		return
	}

	jsonResp(w, map[string]interface{}{
		"configs": Cards,
		"owned":   p.OwnedCards,
		"combo":   DailyComboIDs,
		"claimed": p.LastDailyComboDate == time.Now().Format("2006-01-02"),
		"pph":     getPassivePerHour(p),
	})
}

func handleUpgradeCard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method != http.MethodPost {
		return
	}
	var req struct {
		Username string `json:"username"`
		CardID   string `json:"cardId"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var cardCfg *CardConfig
	for _, c := range Cards {
		if c.ID == req.CardID {
			cardCfg = &c
			break
		}
	}
	if cardCfg == nil {
		return
	}

	db.mu.Lock()
	p := db.Players[req.Username]
	if p == nil {
		db.mu.Unlock()
		return
	}

	currentLevel := p.OwnedCards[req.CardID]
	cost := int64(float64(cardCfg.BaseCost) * math.Pow(cardCfg.CostMultiplier, float64(currentLevel)))

	if p.Balance < cost {
		db.mu.Unlock()
		jsonResp(w, map[string]interface{}{"success": false, "message": "Balanță insuficientă!"})
		return
	}

	p.Balance -= cost
	p.OwnedCards[req.CardID]++
	p.LastBalance = p.Balance
	db.mu.Unlock()
	scheduleSave()

	jsonResp(w, map[string]interface{}{"success": true, "balance": p.Balance, "newLevel": p.OwnedCards[req.CardID]})
}

func handleClaimCombo(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	var req struct {
		Username string `json:"username"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	today := time.Now().Format("2006-01-02")

	db.mu.Lock()
	p := db.Players[req.Username]
	if p == nil {
		db.mu.Unlock()
		return
	}

	if p.LastDailyComboDate == today {
		db.mu.Unlock()
		jsonResp(w, map[string]interface{}{"success": false, "message": "Combo revendicat deja!"})
		return
	}

	ownedAll := true
	for _, id := range DailyComboIDs {
		if p.OwnedCards[id] < 1 {
			ownedAll = false
			break
		}
	}

	if !ownedAll {
		db.mu.Unlock()
		jsonResp(w, map[string]interface{}{"success": false, "message": "Nu deții cardurile combo!"})
		return
	}

	p.Balance += REWARD_DAILY_COMBO
	p.LastDailyComboDate = today
	p.LastBalance = p.Balance
	db.mu.Unlock()
	scheduleSave()

	jsonResp(w, map[string]interface{}{"success": true, "reward": REWARD_DAILY_COMBO})
}

// Handlerele existente rămân neschimbate (Leaderboard, Checkin, Referral etc.)
func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	db.mu.RLock()
	var list []map[string]interface{}
	for u, s := range db.Players {
		if u != "guest" {
			list = append(list, map[string]interface{}{"username": u, "balance": s.Balance, "photo": s.PhotoURL})
		}
	}
	db.mu.RUnlock()
	sort.Slice(list, func(i, j int) bool { return list[i]["balance"].(int64) > list[j]["balance"].(int64) })
	if len(list) > 10 {
		list = list[:10]
	}
	jsonResp(w, list)
}

func handleAppConfig(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	jsonResp(w, map[string]interface{}{
		"adsgramBlockId": ADSGRAM_BLOCK_ID,
		"linkChannel":    LINK_CHANNEL,
		"rewardAd":       REWARD_WATCH_AD,
	})
}

// ═════════════════════════════════════════════════════════════════════════════
// FRONTEND HTML
// ═════════════════════════════════════════════════════════════════════════════

const webAppHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0,maximum-scale=1.0,user-scalable=no,viewport-fit=cover">
<title>ZX Elite</title>
<script src="https://telegram.org/js/telegram-web-app.js"></script>
<script src="https://sad.adsgram.ai/js/sad.min.js"></script>
<style>
:root{
  --bg:#070b10;--panel:#0d1420;--panel2:#111b2b;
  --green:#00f5d4;--green2:#00ff87;--text:#f3fffc;
  --muted:#87a39d;--purple:#9d4dff;--orange:#ff9f43;
  --border:rgba(255,255,255,.08);
}
*{margin:0;padding:0;box-sizing:border-box;}
body{ background:var(--bg); color:var(--text); font-family:sans-serif; overflow-x:hidden; padding-bottom:100px; }
.app{ max-width:600px; margin:auto; padding:15px; }
.header{ display:flex; justify-content:space-between; align-items:center; margin-bottom:20px; }
.section{ background:var(--panel); border-radius:24px; padding:20px; border:1px solid var(--border); margin-bottom:15px; }
.balanceValue{ font-size:42px; font-weight:900; text-align:center; color:#fff; text-shadow:0 0 20px var(--green); margin:10px 0; }
.pph-badge{ background:rgba(0,245,212,0.1); color:var(--green); border-radius:999px; padding:6px 12px; font-size:12px; font-weight:bold; display:inline-block; margin-bottom:10px; }
.coin{ width:220px; height:220px; margin:20px auto; display:block; cursor:pointer; transition:transform 0.1s; }
.coin:active{ transform:scale(0.94); }
.bottomNav{ position:fixed; bottom:0; left:0; right:0; background:rgba(13,20,32,0.95); backdrop-filter:blur(10px); display:flex; padding:10px; border-top:1px solid var(--border); z-index:1000; }
.tabBtn{ flex:1; background:none; border:none; color:var(--muted); font-size:10px; font-weight:bold; cursor:pointer; padding:10px; }
.tabBtn.active{ color:var(--green); }
.btn{ border:none; padding:12px; border-radius:14px; font-weight:800; cursor:pointer; width:100%; margin-top:8px; background:var(--green); color:#000; }
.btn:disabled{ opacity:0.4; }
.hidden{ display:none; }

/* BOOSTS STYLES */
.combo-container{ background:linear-gradient(135deg, rgba(255,159,67,0.1), rgba(157,77,255,0.1)); border:1px dashed var(--orange); border-radius:20px; padding:15px; margin-bottom:20px; text-align:center; }
.combo-grid{ display:flex; justify-content:center; gap:10px; margin:15px 0; }
.combo-item{ width:50px; height:50px; background:var(--panel2); border-radius:12px; border:1px solid var(--border); display:flex; align-items:center; justify-content:center; font-size:22px; }
.combo-item.active{ border-color:var(--green); background:rgba(0,245,212,0.1); box-shadow:0 0 10px var(--green); }
.cards-grid{ display:grid; grid-template-columns:1fr 1fr; gap:12px; }
.card-node{ background:var(--panel2); border-radius:18px; border:1px solid var(--border); padding:15px; display:flex; flex-direction:column; gap:8px; }
.card-icon{ font-size:24px; }
.card-name{ font-weight:bold; font-size:14px; }
.card-profit{ color:var(--green); font-size:11px; font-weight:bold; }
.card-cost{ font-size:14px; font-weight:900; color:var(--orange); margin-top:auto; padding-top:10px; }
</style>
</head>
<body>
<div class="app">
    <div style="text-align:center">
        <div class="pph-badge" id="pphDisplay">⚙️ +0 / oră</div>
        <div id="balanceDisplay" class="balanceValue">0</div>
    </div>

    <!-- TAB GENERATOR -->
    <div id="generatorTab" class="tab-content">
        <img id="coin" class="coin" src="https://cdn-icons-png.flaticon.com/512/8015/8015431.png">
        <div class="section">
            <div style="display:flex;justify-content:space-between;font-size:12px;margin-bottom:8px">
                <span>⚡ Energie</span><span id="energyText">500/500</span>
            </div>
            <div style="width:100%;height:10px;background:#000;border-radius:10px;overflow:hidden">
                <div id="energyFill" style="width:100%;height:100%;background:var(--green);transition:width 0.3s"></div>
            </div>
        </div>
    </div>

    <!-- TAB BOOSTS -->
    <div id="boostsTab" class="tab-content hidden">
        <div class="combo-container">
            <div style="font-weight:bold;font-size:14px;color:var(--orange)">🚀 DAILY COMBO (+5.000.000 ZX)</div>
            <div class="combo-grid" id="comboGrid"></div>
            <button id="claimComboBtn" class="btn" style="background:var(--orange);color:#fff">Revendică Bonus</button>
        </div>
        <div class="cards-grid" id="cardsGrid"></div>
    </div>

    <!-- TAB RANK -->
    <div id="rankTab" class="tab-content hidden">
        <div class="section"><h3 style="margin-bottom:15px">🏆 Top Jucători</h3><div id="leaderboard"></div></div>
    </div>

</div>

<nav class="bottomNav">
    <button class="tabBtn active" data-tab="generator">⛏️ MINE</button>
    <button class="tabBtn" data-tab="boosts">🚀 BOOSTS</button>
    <button class="tabBtn" data-tab="rank">🏆 RANK</button>
</nav>

<script>
let tg = window.Telegram.WebApp;
let cu = { username: tg.initDataUnsafe?.user?.username || 'guest', id: tg.initDataUnsafe?.user?.id || 0 };
let state = { balance: 0, energy: 500, maxEnergy: 500, tapLvl: 1 };
let ownedCards = {};
let cardConfigs = [];
let comboIDs = [];

function fmt(n) { return Number(n).toLocaleString(); }

async function sync() {
    const res = await fetch('/api/sync', {
        method: 'POST',
        headers: {'Content-Type':'application/json'},
        body: JSON.stringify({ username: cu.username, balance: state.balance, tapLevel: state.tapLvl, telegramId: cu.id })
    });
    const d = await res.json();
    state.balance = d.balance;
    updateUI();
}

async function loadCards() {
    const res = await fetch('/api/cards?username=' + cu.username);
    const d = await res.json();
    cardConfigs = d.configs;
    ownedCards = d.owned || {};
    comboIDs = d.combo;
    renderCards(d.claimed);
    document.getElementById('pphDisplay').textContent = '⚙️ +' + fmt(d.pph) + ' / oră';
}

function renderCards(claimed) {
    const grid = document.getElementById('cardsGrid');
    grid.innerHTML = '';
    cardConfigs.forEach(c => {
        const lvl = ownedCards[c.id] || 0;
        const cost = Math.floor(c.baseCost * Math.pow(c.costMultiplier, lvl));
        grid.innerHTML += `
            <div class="card-node">
                <div class="card-icon">${c.icon}</div>
                <div class="card-name">${c.name}</div>
                <div class="card-profit">+${fmt(c.profitPerHour)}/hr</div>
                <div style="font-size:10px;color:var(--muted)">Nivel ${lvl}</div>
                <div class="card-cost">💰 ${fmt(cost)}</div>
                <button class="btn" style="padding:6px;font-size:11px" onclick="upgradeCard('${c.id}')" ${state.balance < cost ? 'disabled' : ''}>Upgrade</button>
            </div>
        `;
    });

    const cGrid = document.getElementById('comboGrid');
    cGrid.innerHTML = '';
    let ownedCount = 0;
    comboIDs.forEach(id => {
        const has = (ownedCards[id] > 0);
        if(has) ownedCount++;
        cGrid.innerHTML += `<div class="combo-item ${has ? 'active' : ''}">${has ? '✅' : '?'}</div>`;
    });
    const comboBtn = document.getElementById('claimComboBtn');
    if(claimed) { comboBtn.textContent = 'RECOMPENSĂ REVENDICATĂ'; comboBtn.disabled = true; }
    else { comboBtn.disabled = ownedCount < 3; }
}

async function upgradeCard(id) {
    const res = await fetch('/api/upgrade', {
        method: 'POST',
        headers: {'Content-Type':'application/json'},
        body: JSON.stringify({ username: cu.username, cardId: id })
    });
    const d = await res.json();
    if(d.success) { state.balance = d.balance; loadCards(); }
    else alert(d.message);
}

document.getElementById('claimComboBtn').onclick = async () => {
    const res = await fetch('/api/combo/claim', {
        method: 'POST',
        headers: {'Content-Type':'application/json'},
        body: JSON.stringify({ username: cu.username })
    });
    const d = await res.json();
    if(d.success) { alert("Felicitări! Ai primit 5M ZX!"); loadCards(); }
};

function updateUI() {
    document.getElementById('balanceDisplay').textContent = fmt(Math.floor(state.balance));
    document.getElementById('energyText').textContent = state.energy + '/' + state.maxEnergy;
    document.getElementById('energyFill').style.width = (state.energy/state.maxEnergy*100) + '%';
}

document.getElementById('coin').onclick = () => {
    if(state.energy > 0) {
        state.balance += state.tapLvl;
        state.energy -= 1;
        updateUI();
    }
};

document.querySelectorAll('.tabBtn').forEach(b => {
    b.onclick = () => {
        document.querySelectorAll('.tabBtn').forEach(x => x.classList.remove('active'));
        document.querySelectorAll('.tab-content').forEach(x => x.classList.add('hidden'));
        b.classList.add('active');
        document.getElementById(b.dataset.tab + 'Tab').classList.remove('hidden');
        if(b.dataset.tab === 'boosts') loadCards();
    };
});

setInterval(sync, 5000);
setInterval(() => { if(state.energy < state.maxEnergy) { state.energy++; updateUI(); } }, 2000);
sync();
</script>
</body>
</html>`

// ═════════════════════════════════════════════════════════════════════════════
// MAIN & ROUTES
// ═════════════════════════════════════════════════════════════════════════════

func main() {
	db = loadDatabase()
	startPeriodicSave()

	token := os.Getenv("TELEGRAM_TOKEN")
	if token != "" {
		bot, _ = tgbotapi.NewBotAPI(token)
	}

	http.HandleFunc("/api/sync", handleSync)
	http.HandleFunc("/api/leaderboard", handleLeaderboard)
	http.HandleFunc("/api/cards", handleGetCards)
	http.HandleFunc("/api/upgrade", handleUpgradeCard)
	http.HandleFunc("/api/combo/claim", handleClaimCombo)
	http.HandleFunc("/api/config", handleAppConfig)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(webAppHTML))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
