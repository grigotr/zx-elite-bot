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
	REWARD_DAILY_COMBO   int64 = 5000000 

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
	OwnedCards         map[string]int `json:"ownedCards"`         
	LastDailyComboDate string         `json:"lastDailyComboDate"` 
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
// GLOBAL STATE
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

	DailyComboIDs = []string{"marketing", "web3", "security"}
)

func loadDatabase() *Database {
	data, err := os.ReadFile(dbPath)
	if err != nil { return &Database{Players: make(map[string]*PlayerState)} }
	d := &Database{Players: make(map[string]*PlayerState)}
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
	if saveTimer != nil { saveTimer.Stop() }
	saveTimer = time.AfterFunc(5*time.Second, saveDatabase)
}

func getOrCreatePlayer(username string, telegramID int64) *PlayerState {
	p, ok := db.Players[username]
	if !ok {
		p = &PlayerState{
			Username: username, LastSync: time.Now(), LastPassive: time.Now(),
			Referrals: []string{}, ClaimedTasks: []string{}, OwnedCards: make(map[string]int),
			TelegramID: telegramID, ReferralCode: "ZX" + strconv.FormatInt(time.Now().UnixNano()%100000, 10),
		}
		db.Players[username] = p
	}
	if p.OwnedCards == nil { p.OwnedCards = make(map[string]int) }
	return p
}

func getPassivePerHour(p *PlayerState) int64 {
	var total int64 = 0
	if p.PassiveLevel > 0 {
		lvl := p.PassiveLevel
		if lvl > MAX_PASSIVE_LEVEL { lvl = MAX_PASSIVE_LEVEL }
		total = int64(math.Round(100 * math.Pow(1.8, float64(lvl-1))))
	}
	for id, lvl := range p.OwnedCards {
		for _, cfg := range Cards {
			if cfg.ID == id { total += cfg.ProfitPerHour * int64(lvl) }
		}
	}
	return total
}

func computePassiveEarned(p *PlayerState) int64 {
	pph := getPassivePerHour(p)
	if pph == 0 { return 0 }
	elapsed := time.Since(p.LastPassive)
	if elapsed > 8*time.Hour { elapsed = 8 * time.Hour }
	return int64(elapsed.Hours() * float64(pph))
}

// ═════════════════════════════════════════════════════════════════════════════
// HANDLERS
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
	if r.Method == http.MethodOptions { return }
	var req struct {
		Username string `json:"username"`; Balance int64 `json:"balance"`
		TapLevel int `json:"tapLevel"`; PassiveLevel int `json:"passiveLevel"`; TelegramID int64 `json:"telegramId"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Username == "" || req.Username == "guest" { return }
	db.mu.Lock()
	p := getOrCreatePlayer(req.Username, req.TelegramID)
	earned := computePassiveEarned(p)
	p.LastPassive = time.Now()
	p.Balance = req.Balance + earned
	p.TapLevel = req.TapLevel
	p.PassiveLevel = req.PassiveLevel
	db.mu.Unlock()
	scheduleSave()
	jsonResp(w, map[string]interface{}{"balance": p.Balance, "passiveEarned": earned})
}

func handleGetCards(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	u := r.URL.Query().Get("username")
	db.mu.RLock()
	p := db.Players[u]
	db.mu.RUnlock()
	if p == nil { return }
	jsonResp(w, map[string]interface{}{
		"configs": Cards, "owned": p.OwnedCards, "combo": DailyComboIDs,
		"claimed": p.LastDailyComboDate == time.Now().Format("2006-01-02"),
		"pph": getPassivePerHour(p),
	})
}

func handleUpgradeCard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	var req struct { Username string `json:"username"`; CardID string `json:"cardId"` }
	json.NewDecoder(r.Body).Decode(&req)
	db.mu.Lock()
	defer db.mu.Unlock()
	p := db.Players[req.Username]
	if p == nil { return }
	var cfg *CardConfig
	for _, c := range Cards { if c.ID == req.CardID { cfg = &c; break } }
	if cfg == nil { return }
	cost := int64(float64(cfg.BaseCost) * math.Pow(cfg.CostMultiplier, float64(p.OwnedCards[req.CardID])))
	if p.Balance < cost { jsonResp(w, map[string]interface{}{"success": false}); return }
	p.Balance -= cost
	p.OwnedCards[req.CardID]++
	scheduleSave()
	jsonResp(w, map[string]interface{}{"success": true, "balance": p.Balance})
}

func handleClaimCombo(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	var req struct { Username string `json:"username"` }
	json.NewDecoder(r.Body).Decode(&req)
	db.mu.Lock()
	defer db.mu.Unlock()
	p := db.Players[req.Username]
	if p == nil { return }
	today := time.Now().Format("2006-01-02")
	if p.LastDailyComboDate == today { return }
	for _, id := range DailyComboIDs { if p.OwnedCards[id] < 1 { return } }
	p.Balance += REWARD_DAILY_COMBO
	p.LastDailyComboDate = today
	scheduleSave()
	jsonResp(w, map[string]interface{}{"success": true})
}

// ═════════════════════════════════════════════════════════════════════════════
// FRONTEND
// ═════════════════════════════════════════════════════════════════════════════

const webAppHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0,maximum-scale=1.0,user-scalable=no,viewport-fit=cover">
<title>ZX Elite</title>
<script src="https://telegram.org/js/telegram-web-app.js"></script>
<style>
:root{ --bg:#070b10; --panel:#0d1420; --green:#00f5d4; --text:#f3fffc; --muted:#87a39d; --orange:#ff9f43; --border:rgba(255,255,255,.08); }
body{ background:var(--bg); color:var(--text); font-family:sans-serif; margin:0; padding-bottom:100px; }
.app{ max-width:500px; margin:auto; padding:15px; }
.section{ background:var(--panel); border-radius:20px; padding:20px; border:1px solid var(--border); margin-bottom:15px; }
.balanceValue{ font-size:40px; font-weight:900; text-align:center; color:white; text-shadow:0 0 20px var(--green); margin:10px 0; }
.coin{ width:200px; height:200px; margin:20px auto; display:block; cursor:pointer; }
.bottomNav{ position:fixed; bottom:0; left:0; right:0; background:rgba(13,20,32,0.95); display:flex; padding:10px; border-top:1px solid var(--border); }
.tabBtn{ flex:1; background:none; border:none; color:var(--muted); font-size:10px; font-weight:bold; cursor:pointer; }
.tabBtn.active{ color:var(--green); }
.btn{ border:none; padding:12px; border-radius:12px; font-weight:800; cursor:pointer; width:100%; background:var(--green); color:#000; }
.hidden{ display:none; }
.combo-grid{ display:flex; justify-content:center; gap:10px; margin:15px 0; }
.combo-item{ width:45px; height:45px; background:#111b2b; border:1px solid var(--border); border-radius:10px; display:flex; align-items:center; justify-content:center; }
.combo-item.active{ border-color:var(--green); background:rgba(0,245,212,0.1); }
.cards-grid{ display:grid; grid-template-columns:1fr 1fr; gap:10px; }
.card-node{ background:#111b2b; border-radius:15px; padding:12px; border:1px solid var(--border); }
</style>
</head>
<body>
<div class="app">
    <div style="text-align:center"><div id="pphDisp" style="color:var(--green);font-size:12px;font-weight:bold">⚙️ +0/hr</div><div id="balDisp" class="balanceValue">0</div></div>
    <div id="genTab" class="tab-content">
        <img id="coin" class="coin" src="https://cdn-icons-png.flaticon.com/512/8015/8015431.png">
        <div class="section"><div style="display:flex;justify-content:space-between;font-size:12px"><span>⚡ Energie</span><span id="enTxt">500/500</span></div><div style="width:100%;height:8px;background:#000;border-radius:5px;margin-top:5px;overflow:hidden"><div id="enFill" style="width:100%;height:100%;background:var(--green)"></div></div></div>
    </div>
    <div id="bstTab" class="tab-content hidden">
        <div class="section" style="text-align:center;border-color:var(--orange)">
            <div style="font-weight:bold;color:var(--orange)">DAILY COMBO (+5M ZX)</div>
            <div id="cmboGrd" class="combo-grid"></div>
            <button id="clmCmbo" class="btn" style="background:var(--orange);color:#fff">Claim Bonus</button>
        </div>
        <div id="crdsGrd" class="cards-grid"></div>
    </div>
</div>
<nav class="bottomNav">
    <button class="tabBtn active" data-tab="gen">⛏️ MINE</button>
    <button class="tabBtn" data-tab="bst">🚀 BOOSTS</button>
</nav>
<script>
let tg = window.Telegram.WebApp;
let cu = { un: tg.initDataUnsafe?.user?.username || 'guest', id: tg.initDataUnsafe?.user?.id || 0 };
let st = { bal: 0, en: 500, mx: 500 };
function fmt(n) { return Number(n).toLocaleString(); }
async function load() {
    const r = await fetch('/api/cards?username=' + cu.un);
    const d = await r.json();
    document.getElementById('pphDisp').textContent = '⚙️ +' + fmt(d.pph) + '/hr';
    const grd = document.getElementById('crdsGrd'); grd.innerHTML = '';
    d.configs.forEach(c => {
        const lvl = d.owned[c.id] || 0;
        const cst = Math.floor(c.baseCost * Math.pow(c.costMultiplier, lvl));
        grd.innerHTML += '<div class="card-node"><b>'+c.icon+' '+c.name+'</b><div style="font-size:10px;color:var(--green)">+'+fmt(c.profitPerHour)+'/hr</div><div style="font-size:10px">Niv.'+lvl+'</div><div style="font-size:12px;color:var(--orange);margin:5px 0">💰 '+fmt(cst)+'</div><button class="btn" style="padding:5px;font-size:10px" onclick="upgr(\''+c.id+'\')">Upgrade</button></div>';
    });
    const cGrd = document.getElementById('cmboGrd'); cGrd.innerHTML = '';
    let has = 0;
    d.combo.forEach(id => {
        const ok = (d.owned[id] > 0); if(ok) has++;
        cGrd.innerHTML += '<div class="combo-item '+(ok?'active':'')+'">'+(ok?'✅':'?')+'</div>';
    });
    document.getElementById('clmCmbo').disabled = (has < 3 || d.claimed);
}
async function upgr(id) {
    const r = await fetch('/api/upgrade', { method:'POST', body: JSON.stringify({username:cu.un, cardId:id}) });
    const d = await r.json(); if(d.success) { st.bal = d.balance; load(); }
}
document.getElementById('coin').onclick = () => { if(st.en > 0) { st.bal++; st.en--; ui(); } };
function ui() { document.getElementById('balDisp').textContent = fmt(Math.floor(st.bal)); document.getElementById('enTxt').textContent = st.en+'/500'; document.getElementById('enFill').style.width = (st.en/5)+'%'; }
document.querySelectorAll('.tabBtn').forEach(b => b.onclick = () => {
    document.querySelectorAll('.tabBtn').forEach(x => x.classList.remove('active')); b.classList.add('active');
    document.querySelectorAll('.tab-content').forEach(x => x.classList.add('hidden')); document.getElementById(b.dataset.tab+'Tab').classList.remove('hidden');
    if(b.dataset.tab === 'bst') load();
});
document.getElementById('clmCmbo').onclick = async () => { const r = await fetch('/api/combo/claim', { method:'POST', body:JSON.stringify({username:cu.un}) }); if(r.ok) location.reload(); };
setInterval(ui, 100); setInterval(() => { if(st.en < 500) st.en++; }, 2000);
load();
</script>
</body>
</html>`

func main() {
	db = loadDatabase()
	http.HandleFunc("/api/sync", handleSync)
	http.HandleFunc("/api/cards", handleGetCards)
	http.HandleFunc("/api/upgrade", handleUpgradeCard)
	http.HandleFunc("/api/combo/claim", handleClaimCombo)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(webAppHTML))
	})
	p := os.Getenv("PORT")
	if p == "" { p = "8080" }
	log.Fatal(http.ListenAndServe(":"+p, nil))
}
