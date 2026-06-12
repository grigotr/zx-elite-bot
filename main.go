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
	REWARD_DAILY_COMBO   int64 = 5000000 // 5M ZX Bonus

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

type Card struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	BaseCost       int64   `json:"baseCost"`
	CostMultiplier float64 `json:"costMultiplier"`
	ProfitPerHour  int64   `json:"profitPerHour"`
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
	OwnedCards         map[string]int `json:"ownedCards"`         // cardID -> level
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
// GLOBAL CONFIGS & STATE
// ═════════════════════════════════════════════════════════════════════════════

var (
	db        *Database
	dbPath    = "database.json"
	saveTimer *time.Timer
	saveMu    sync.Mutex
	bot       *tgbotapi.BotAPI

	CardConfigs = map[string]Card{
		"marketing": {ID: "marketing", Name: "Marketing Team", Description: "Social media promotion", BaseCost: 1500, CostMultiplier: 1.5, ProfitPerHour: 150},
		"legal":     {ID: "legal", Name: "Legal Advisor", Description: "Global compliance", BaseCost: 5000, CostMultiplier: 1.6, ProfitPerHour: 450},
		"web3":      {ID: "web3", Name: "Web3 Developer", Description: "Smart contract core", BaseCost: 15000, CostMultiplier: 1.7, ProfitPerHour: 1200},
		"exchange":  {ID: "exchange", Name: "CEX Listing", Description: "Tier 1 exchange listing", BaseCost: 50000, CostMultiplier: 1.8, ProfitPerHour: 4500},
		"security":  {ID: "security", Name: "Security Audit", Description: "Anti-hack protocols", BaseCost: 125000, CostMultiplier: 1.9, ProfitPerHour: 11000},
		"influencer": {ID: "influencer", Name: "KOL Partner", Description: "Mass adoption drive", BaseCost: 300000, CostMultiplier: 2.1, ProfitPerHour: 28000},
	}

	CurrentDailyCombo = []string{"marketing", "web3", "exchange"} // Se poate automatiza schimbarea zilnică
)

// ═════════════════════════════════════════════════════════════════════════════
// DATABASE & PLAYER HELPERS
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
		p.ReferralCode = "ZX" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
		db.Players[username] = p
	}
	if p.OwnedCards == nil {
		p.OwnedCards = make(map[string]int)
	}
	return p
}

// ═════════════════════════════════════════════════════════════════════════════
// LOGICA PASSIVE INCOME & COMBO
// ═════════════════════════════════════════════════════════════════════════════

func passivePerHour(p *PlayerState) int64 {
	var total int64 = 0
	// Profitul din PassiveLevel (vechiul sistem)
	if p.PassiveLevel > 0 {
		lvl := p.PassiveLevel
		if lvl > MAX_PASSIVE_LEVEL {
			lvl = MAX_PASSIVE_LEVEL
		}
		total += int64(math.Round(100 * math.Pow(1.8, float64(lvl-1))))
	}
	// Profitul din Carduri (noul sistem)
	for id, lvl := range p.OwnedCards {
		if config, ok := CardConfigs[id]; ok && lvl > 0 {
			total += config.ProfitPerHour * int64(lvl)
		}
	}
	return total
}

func computePassiveEarned(p *PlayerState) int64 {
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
		EnergyLevel  int    `json:"energyLevel"`
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
	p.EnergyLevel = req.EnergyLevel
	p.PassiveLevel = req.PassiveLevel
	p.LastBalance = p.Balance
	p.LastSync = time.Now()
	db.mu.Unlock()
	scheduleSave()

	jsonResp(w, map[string]interface{}{"status": "ok", "balance": p.Balance, "passiveEarned": earned})
}

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
		"configs": CardConfigs,
		"owned":   p.OwnedCards,
		"combo":   CurrentDailyCombo,
		"claimed": p.LastDailyComboDate == time.Now().Format("2006-01-02"),
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
	config, exists := CardConfigs[req.CardID]
	if !exists {
		return
	}

	db.mu.Lock()
	p := db.Players[req.Username]
	if p == nil {
		db.mu.Unlock()
		return
	}

	currentLevel := p.OwnedCards[req.CardID]
	cost := int64(float64(config.BaseCost) * math.Pow(config.CostMultiplier, float64(currentLevel)))

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
		jsonResp(w, map[string]interface{}{"success": false, "message": "Deja revendicat azi!"})
		return
	}

	for _, id := range CurrentDailyCombo {
		if p.OwnedCards[id] < 1 {
			db.mu.Unlock()
			jsonResp(w, map[string]interface{}{"success": false, "message": "Nu deții toate cardurile combo!"})
			return
		}
	}

	p.Balance += REWARD_DAILY_COMBO
	p.LastDailyComboDate = today
	p.LastBalance = p.Balance
	db.mu.Unlock()
	scheduleSave()

	jsonResp(w, map[string]interface{}{"success": true, "reward": REWARD_DAILY_COMBO})
}

// ═════════════════════════════════════════════════════════════════════════════
// ALTE HANDLERE EXISTENTE (REDUSE PENTRU SPAȚIU DAR FUNCȚIONALE)
// ═════════════════════════════════════════════════════════════════════════════

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	db.mu.RLock()
	var list []map[string]interface{}
	for u, s := range db.Players {
		if u != "guest" {
			list = append(list, map[string]interface{}{"username": u, "firstName": s.FirstName, "balance": s.Balance, "photoUrl": s.PhotoURL})
		}
	}
	db.mu.RUnlock()
	sort.Slice(list, func(i, j int) bool { return list[i]["balance"].(int64) > list[j]["balance"].(int64) })
	if len(list) > 15 {
		list = list[:15]
	}
	jsonResp(w, list)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	jsonResp(w, map[string]interface{}{
		"adsgramBlockId": ADSGRAM_BLOCK_ID,
		"linkChannel":    LINK_CHANNEL,
		"linkTwitter":    LINK_TWITTER,
		"appUrl":         APP_URL,
	})
}

// ═════════════════════════════════════════════════════════════════════════════
// FRONTEND HTML (CU NOUL TAB DE BOOSTS)
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
  --green:#00f5d4;--text:#f3fffc;--muted:#87a39d;--orange:#ff9f43;
  --border:rgba(255,255,255,.08);
}
*{margin:0;padding:0;box-sizing:border-box;}
body{ background:var(--bg); color:var(--text); font-family:sans-serif; overflow-x:hidden; padding-bottom:100px; }
.app{ max-width:500px; margin:auto; padding:15px; }
.section{ background:var(--panel); border-radius:20px; padding:20px; border:1px solid var(--border); margin-bottom:15px; }
.balanceValue{ font-size:40px; font-weight:900; text-align:center; color:white; text-shadow:0 0 20px var(--green); margin:10px 0; }
.coin{ width:200px; height:200px; margin:20px auto; display:block; cursor:pointer; transition:transform 0.1s; }
.coin:active{ transform:scale(0.92); }
.btn{ border:none; padding:12px; border-radius:12px; font-weight:800; cursor:pointer; width:100%; margin-top:10px; background:var(--green); color:#000; }
.btn:disabled{ opacity:0.4; }
.hidden{ display:none; }
.bottomNav{ position:fixed; bottom:10px; left:10px; right:10px; background:rgba(13,20,32,0.95); backdrop-filter:blur(10px); border-radius:20px; display:flex; padding:10px; border:1px solid var(--border); z-index:1000; }
.tabBtn{ flex:1; background:none; border:none; color:var(--muted); font-size:10px; font-weight:bold; cursor:pointer; }
.tabBtn.active{ color:var(--green); }

/* BOOSTS STYLE */
.combo-section{ background:rgba(255,159,67,0.1); border:1px dashed var(--orange); border-radius:15px; padding:15px; text-align:center; margin-bottom:15px; }
.combo-dots{ display:flex; justify-content:center; gap:10px; margin:10px 0; }
.combo-dot{ width:45px; height:45px; background:var(--panel2); border:1px solid var(--border); border-radius:10px; display:flex; align-items:center; justify-content:center; font-size:20px; }
.combo-dot.active{ border-color:var(--green); background:rgba(0,245,212,0.1); }
.cards-grid{ display:grid; grid-template-columns:1fr 1fr; gap:10px; }
.card-item{ background:var(--panel2); border-radius:15px; padding:12px; border:1px solid var(--border); display:flex; flex-direction:column; }
.card-name{ font-size:13px; font-weight:bold; color:var(--green); }
.card-profit{ font-size:10px; color:var(--muted); margin:4px 0; }
.card-cost{ font-size:14px; font-weight:900; color:var(--orange); margin-top:auto; }
</style>
</head>
<body>
<div class="app">
    <div id="balance-header" style="text-align:center">
        <div style="font-size:12px;color:var(--muted)">BALANCE</div>
        <div id="balanceDisplay" class="balanceValue">0</div>
        <div id="pphDisplay" style="color:var(--green);font-size:11px;font-weight:bold">⚙️ +0/hour</div>
    </div>

    <!-- TAB MINE -->
    <div id="mineTab" class="tab-content">
        <img id="coin" class="coin" src="https://cdn-icons-png.flaticon.com/512/8015/8015431.png">
        <div class="section">
            <div style="display:flex;justify-content:space-between;font-size:12px;margin-bottom:5px">
                <span>⚡ Energy</span><span id="energyText">500/500</span>
            </div>
            <div style="width:100%;height:10px;background:#000;border-radius:10px;overflow:hidden">
                <div id="energyFill" style="width:100%;height:100%;background:var(--green)"></div>
            </div>
        </div>
    </div>

    <!-- TAB BOOSTS -->
    <div id="boostsTab" class="tab-content hidden">
        <div class="combo-section">
            <div style="font-weight:bold;color:var(--orange)">DAILY COMBO (+5,000,000 ZX)</div>
            <div class="combo-dots" id="comboDots"></div>
            <button id="claimComboBtn" class="btn" style="background:var(--orange);color:white">Claim Bonus</button>
        </div>
        <div class="cards-grid" id="cardsGrid"></div>
    </div>

    <!-- ALT TABURI (Leaderboard etc) -->
    <div id="rankTab" class="tab-content hidden">
        <div class="section"><h3 style="margin-bottom:10px">🏆 Leaderboard</h3><div id="leaderboardList"></div></div>
    </div>
</div>

<div class="bottomNav">
    <button class="tabBtn active" data-tab="mine">⛏️ MINE</button>
    <button class="tabBtn" data-tab="boosts">🚀 BOOSTS</button>
    <button class="tabBtn" data-tab="rank">🏆 RANK</button>
</div>

<script>
let tg = window.Telegram.WebApp;
let cu = { username: tg.initDataUnsafe?.user?.username || 'guest', id: tg.initDataUnsafe?.user?.id || 0 };
let state = { balance: 0, energy: 500, maxEnergy: 500, tapLvl: 1, passiveLvl: 0 };
let cardConfigs = {};
let ownedCards = {};
let dailyCombo = [];

function fmt(n) { return Number(n).toLocaleString(); }

async function sync() {
    const r = await fetch('/api/sync', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({ username: cu.username, balance: state.balance, tapLevel: state.tapLvl, energyLevel: 1, passiveLevel: state.passiveLvl, telegramId: cu.id })
    });
    const d = await r.json();
    state.balance = d.balance;
    updateUI();
}

async function loadBoosts() {
    const r = await fetch('/api/cards?username=' + cu.username);
    const d = await r.json();
    cardConfigs = d.configs;
    ownedCards = d.owned || {};
    dailyCombo = d.combo;
    renderCards(d.claimed);
}

function renderCards(isClaimed) {
    const grid = document.getElementById('cardsGrid');
    grid.innerHTML = '';
    let pphTotal = 0;

    for (let id in cardConfigs) {
        const c = cardConfigs[id];
        const lvl = ownedCards[id] || 0;
        const cost = Math.floor(c.BaseCost * Math.pow(c.CostMultiplier, lvl));
        pphTotal += c.ProfitPerHour * lvl;

        grid.innerHTML += `
            <div class="card-item">
                <div class="card-name">${c.Name}</div>
                <div class="card-profit">📈 +${fmt(c.ProfitPerHour)}/hr</div>
                <div style="font-size:9px;color:var(--muted)">Level ${lvl}</div>
                <div class="card-cost">💰 ${fmt(cost)}</div>
                <button class="btn" style="padding:5px;font-size:11px" onclick="upgradeCard('${id}')" ${state.balance < cost ? 'disabled' : ''}>Upgrade</button>
            </div>`;
    }
    document.getElementById('pphDisplay').textContent = '⚙️ +' + fmt(pphTotal) + '/hour';

    const dots = document.getElementById('comboDots');
    dots.innerHTML = '';
    let ownedCount = 0;
    dailyCombo.forEach(id => {
        const owned = (ownedCards[id] > 0);
        if(owned) ownedCount++;
        dots.innerHTML += `<div class="combo-dot ${owned ? 'active' : ''}">${owned ? '✅' : '?'}</div>`;
    });
    
    const cBtn = document.getElementById('claimComboBtn');
    if(isClaimed) { cBtn.textContent = 'CLAIMED ✓'; cBtn.disabled = true; }
    else { cBtn.disabled = ownedCount < 3; }
}

async function upgradeCard(id) {
    const r = await fetch('/api/upgrade', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({ username: cu.username, cardId: id })
    });
    const d = await r.json();
    if(d.success) {
        state.balance = d.balance;
        ownedCards[id] = d.newLevel;
        updateUI();
        loadBoosts();
    }
}

document.getElementById('claimComboBtn').onclick = async () => {
    const r = await fetch('/api/combo/claim', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({ username: cu.username })
    });
    const d = await r.json();
    if(d.success) { alert("Mega Bonus claimed: 5,000,000 ZX!"); location.reload(); }
};

function updateUI() {
    document.getElementById('balanceDisplay').textContent = fmt(Math.floor(state.balance));
    document.getElementById('energyText').textContent = state.energy + '/' + state.maxEnergy;
    document.getElementById('energyFill').style.width = (state.energy/state.maxEnergy*100) + '%';
}

document.getElementById('coin').onclick = (e) => {
    if(state.energy > 0) {
        state.balance += state.tapLvl;
        state.energy -= 1;
        updateUI();
    }
};

document.querySelectorAll('.tabBtn').forEach(btn => {
    btn.onclick = () => {
        document.querySelectorAll('.tab-content').forEach(c => c.classList.add('hidden'));
        document.getElementById(btn.dataset.tab + 'Tab').classList.remove('hidden');
        document.querySelectorAll('.tabBtn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        if(btn.dataset.tab === 'boosts') loadBoosts();
    };
});

setInterval(sync, 5000);
setInterval(() => { if(state.energy < state.maxEnergy) { state.energy++; updateUI(); } }, 2000);
sync();
loadBoosts();
</script>
</body>
</html>`

// ═════════════════════════════════════════════════════════════════════════════
// MAIN & ROUTES
// ═════════════════════════════════════════════════════════════════════════════

func main() {
	db = loadDatabase()

	token := os.Getenv("TELEGRAM_TOKEN")
	if token != "" {
		bot, _ = tgbotapi.NewBotAPI(token)
	}

	http.HandleFunc("/api/sync", handleSync)
	http.HandleFunc("/api/leaderboard", handleLeaderboard)
	http.HandleFunc("/api/config", handleConfig)
	http.HandleFunc("/api/cards", handleGetCards)
	http.HandleFunc("/api/upgrade", handleUpgradeCard)
	http.HandleFunc("/api/combo/claim", handleClaimCombo)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(webAppHTML))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("🚀 ZX Elite running on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
