package main

import (
	"log"
	"net/http"
	"os"
	"sync/atomic"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var tapCount int64

// Infrastructură Web3 cu 4 Meniuri Premium, Energie și Leaderboard
const htmlContent = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>ZX-Elite Elite Network</title>
    <style>
        :root {
            --bg-primary: #040814;
            --neon-blue: #00f5d4;
            --neon-purple: #9b5de5;
            --neon-gold: #ffb703;
            --text-main: #ffffff;
            --card-bg: rgba(255, 255, 255, 0.03);
            --card-border: rgba(0, 245, 212, 0.12);
        }
        body {
            font-family: 'Inter', 'Segoe UI', system-ui, sans-serif;
            background: linear-gradient(135deg, #050b14 0%, #0a1128 50%, #020611 100%);
            color: var(--text-main);
            margin: 0; padding: 0;
            display: flex; flex-direction: column; align-items: center;
            height: 100vh; overflow: hidden;
            user-select: none; -webkit-user-select: none;
        }
        body::before {
            content: ""; position: absolute; top: 0; left: 0; width: 100%; height: 100%;
            background-image: radial-gradient(rgba(0, 245, 212, 0.04) 1px, transparent 0);
            background-size: 24px 24px; z-index: 0; pointer-events: none;
        }
        .app-container {
            position: relative; z-index: 1; width: 90%; max-width: 450px;
            height: 100vh; display: flex; flex-direction: column;
            justify-content: space-between; padding: 20px 0; box-sizing: border-box;
        }
        .header { display: flex; justify-content: space-between; align-items: center; }
        .status-badge {
            background: rgba(0, 245, 212, 0.08); border: 1px solid var(--neon-blue);
            padding: 6px 14px; border-radius: 30px; font-size: 0.75rem;
            text-transform: uppercase; letter-spacing: 2px; font-weight: bold;
            box-shadow: 0 0 15px rgba(0, 245, 212, 0.2);
        }
        
        .score-container { text-align: center; margin-top: -5px; }
        .score-label { font-size: 0.75rem; color: #8a99ad; text-transform: uppercase; letter-spacing: 3px; }
        .score-value {
            font-size: 3.3rem; font-weight: 900; background: linear-gradient(to right, #fff 40%, var(--neon-blue));
            -webkit-background-clip: text; -webkit-text-fill-color: transparent;
            text-shadow: 0 0 30px rgba(0, 245, 212, 0.3); margin: 5px 0;
        }
        
        /* Conținutul paginilor */
        .page-content { flex-grow: 1; display: none; flex-direction: column; justify-content: center; width: 100%; animation: fadeIn 0.3s ease; }
        .page-content.active { display: flex; }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(5px); } to { opacity: 1; transform: translateY(0); } }
        
        /* PAGINA JOC: REFACTORIZARE CU ENERGIE */
        .reactor-container { position: relative; display: flex; justify-content: center; align-items: center; height: 32vh; }
        .reactor-outer {
            display: flex; justify-content: center; align-items: center; width: 200px; height: 200px; border-radius: 50%;
            background: rgba(0, 0, 0, 0.2); border: 2px dashed rgba(0, 245, 212, 0.25); animation: rotateOuter 25s linear infinite;
        }
        @keyframes rotateOuter { 100% { transform: rotate(360deg); } }
        .reactor-core {
            position: absolute; width: 160px; height: 160px;
            background: radial-gradient(circle, var(--neon-blue) 0%, #00b4d8 40%, #03045e 100%);
            border-radius: 50%; box-shadow: 0 0 50px rgba(0, 245, 212, 0.4);
            cursor: pointer; transition: transform 0.05s ease; border: 3px solid rgba(255, 255, 255, 0.1);
        }
        .reactor-core:active { transform: scale(0.92); }
        .energy-box { text-align: center; margin-top: 15px; }
        .energy-bar-bg { width: 100%; height: 10px; background: rgba(255,255,255,0.05); border-radius: 10px; overflow: hidden; border: 1px solid rgba(255,255,255,0.1); margin-top: 5px;}
        .energy-bar-fill { width: 100%; height: 100%; background: linear-gradient(90deg, #00b4d8, var(--neon-blue)); transition: width 0.1s ease; }
        .energy-text { font-size: 0.85rem; font-weight: bold; color: #e2e8f0; }
        .ad-recharge-btn {
            background: linear-gradient(90deg, #ff7000, var(--neon-gold)); color: #040814;
            border: none; padding: 10px 20px; border-radius: 12px; font-weight: bold; font-size: 0.85rem;
            margin-top: 15px; cursor: pointer; box-shadow: 0 4px 15px rgba(255, 183, 3, 0.3); width: 100%;
        }
        
        /* STANDARDE DESIGN MODULE (TASKS, WALLET, LEADERBOARD) */
        .panel-container { background: var(--card-bg); border: 1px solid var(--card-border); border-radius: 20px; padding: 20px; backdrop-filter: blur(10px); max-height: 52vh; overflow-y: auto; }
        .panel-title { font-size: 1.1rem; margin-top: 0; margin-bottom: 16px; color: var(--neon-blue); font-weight: bold; text-transform: uppercase; letter-spacing: 1px; }
        
        .task-row { display: flex; justify-content: space-between; align-items: center; background: rgba(255, 255, 255, 0.01); border: 1px solid rgba(255, 255, 255, 0.03); padding: 12px 14px; border-radius: 12px; margin-bottom: 10px; }
        .task-info { display: flex; flex-direction: column; }
        .task-name { font-size: 0.85rem; font-weight: 600; }
        .task-reward { color: var(--neon-blue); font-weight: bold; font-size: 0.8rem; margin-top: 2px; }
        .task-btn { background: linear-gradient(90deg, #00b4d8, var(--neon-blue)); color: #040814; border: none; padding: 8px 16px; border-radius: 8px; font-size: 0.8rem; font-weight: bold; text-decoration: none; }
        
        /* PAGINA PORTOFEL (WALLET) */
        .wallet-card { text-align: center; padding: 20px 10px; }
        .wallet-balance-box { background: rgba(255,255,255,0.02); border: 1px dashed rgba(0, 245, 212, 0.3); border-radius: 16px; padding: 15px; margin-bottom: 25px; }
        .wallet-balance-title { font-size: 0.8rem; color: #8a99ad; text-transform: uppercase; letter-spacing: 2px; }
        .wallet-balance-value { font-size: 2.2rem; font-weight: bold; color: #fff; margin-top: 5px; text-shadow: 0 0 15px rgba(255,255,255,0.2); }
        .connect-wallet-btn { background: linear-gradient(90deg, #0072ff, #00c6ff); color: #fff; border: none; padding: 14px 28px; border-radius: 14px; font-weight: bold; font-size: 0.95rem; cursor: pointer; width: 100%; box-shadow: 0 4px 20px rgba(0, 114, 255, 0.4); }
        
        /* PAGINA CLASAMENT (LEADERBOARD) */
        .leaderboard-list { width: 100%; }
        .leader-row { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; background: rgba(255,255,255,0.01); border-radius: 10px; margin-bottom: 6px; font-size: 0.85rem; border-left: 3px solid transparent; }
        .leader-row.top1 { border-left-color: var(--neon-gold); background: rgba(255, 183, 3, 0.03); }
        .leader-row.top2 { border-left-color: #e2e8f0; }
        .leader-row.top3 { border-left-color: #cd7f32; }
        .leader-rank { font-weight: bold; color: var(--neon-blue); width: 25px; }
        .leader-name { flex-grow: 1; color: #e2e8f0; }
        .leader-score { font-weight: bold; }
        .user-rank-section { margin-top: 15px; padding-top: 15px; border-top: 1px solid rgba(255,255,255,0.1); }
        .user-row-special { display: flex; justify-content: space-between; align-items: center; padding: 12px; background: rgba(155, 93, 229, 0.1); border: 1px solid var(--neon-purple); border-radius: 12px; font-size: 0.9rem; font-weight: bold; }

        /* NAVIGAȚIE JOS (4 MENIURI) */
        .bottom-nav {
            display: flex; justify-content: space-between; background: rgba(10, 17, 40, 0.85);
            border: 1px solid rgba(255, 255, 255, 0.06); backdrop-filter: blur(25px);
            border-radius: 20px; padding: 10px 8px; margin-top: 10px; box-shadow: 0 -10px 30px rgba(0,0,0,0.4);
        }
        .nav-item {
            display: flex; flex-direction: column; align-items: center; gap: 4px;
            color: #475569; font-size: 0.7rem; font-weight: bold; text-decoration: none;
            cursor: pointer; transition: color 0.2s; flex: 1; text-align: center;
        }
        .nav-item.active { color: var(--neon-blue); text-shadow: 0 0 10px rgba(0, 245, 212, 0.3); }
        .nav-icon { font-size: 1.2rem; }
        
        .floating-text { position: absolute; color: #fff; font-size: 1.8rem; font-weight: 900; pointer-events: none; animation: floatUp 0.5s ease-out forwards; text-shadow: 0 0 10px var(--neon-blue); }
        @keyframes floatUp { 0% { opacity: 1; transform: translateY(0); } 100% { opacity: 0; transform: translateY(-70px); } }
    </style>
</head>
<body>
    <div class="app-container">
        <div class="header">
            <div class="status-badge">⚡ Core Active</div>
            <div style="font-size: 0.75rem; color: #475569; font-weight: bold; letter-spacing: 1px;">ZX-WEB3 ENGINE</div>
        </div>

        <div class="score-container">
            <div class="score-label">Total ZX Tokens</div>
            <div class="score-value" id="global-score">0</div>
        </div>

        <div id="page-tap" class="page-content active">
            <div class="reactor-container" id="tap-zone">
                <div class="reactor-outer"></div>
                <div class="reactor-core" id="core-button"></div>
            </div>
            <div class="energy-box">
                <div style="display: flex; justify-content: space-between; font-size: 0.8rem;">
                    <span>🔋 Core Energy</span>
                    <span id="energy-text">500 / 500</span>
                </div>
                <div class="energy-bar-bg"><div class="energy-bar-fill" id="energy-fill"></div></div>
                <button class="ad-recharge-btn" id="ad-btn" onclick="watchAd()">📺 Recharge Energy via Ads (<span id="ad-count">3</span> left)</button>
            </div>
        </div>

        <div id="page-tasks" class="page-content">
            <div class="panel-container">
                <div class="panel-title">💼 Premium Operations</div>
                <div class="task-row">
                    <div class="task-info">
                        <span class="task-name">🚀 Launch Partner App</span>
                        <span class="task-reward">+50,000 ZX</span>
                    </div>
                    <a href="https://t.me/PartnerBot/app" target="_blank" class="task-btn" onclick="claimTask(1)">Start</a>
                </div>
                <div class="task-row">
                    <div class="task-info">
                        <span class="task-name">📢 Join ZX Network Channel</span>
                        <span class="task-reward">+25,000 ZX</span>
                    </div>
                    <a href="https://t.me/your_channel" target="_blank" class="task-btn" onclick="claimTask(2)">Join</a>
                </div>
            </div>
        </div>

        <div id="page-wallet" class="page-content">
            <div class="panel-container wallet-card">
                <div class="panel-title">👛 Web3 Integration</div>
                <div class="wallet-balance-box">
                    <div class="wallet-balance-title">Available Balance</div>
                    <div class="wallet-balance-value"><span id="wallet-score">0</span> ZX</div>
                </div>
                <button class="connect-wallet-btn" onclick="connectWallet()">Connect Telegram Wallet</button>
            </div>
        </div>

        <div id="page-leaderboard" class="page-content">
            <div class="panel-container">
                <div class="panel-title">🏆 Top 10 Operators</div>
                <div class="leaderboard-list">
                    <div class="leader-row top1"><span class="leader-rank">#1</span><span class="leader-name">Vlad_ZXElite</span><span class="leader-score">12,450,000</span></div>
                    <div class="leader-row top2"><span class="leader-rank">#2</span><span class="leader-name">AlexCrypton</span><span class="leader-score">9,120,000</span></div>
                    <div class="leader-row top3"><span class="leader-rank">#3</span><span class="leader-name">Dmitry_Net</span><span class="leader-score">7,800,000</span></div>
                    <div class="leader-row"><span class="leader-rank">#4</span><span class="leader-name">CryptoBoss</span><span class="leader-score">5,200,000</span></div>
                    <div class="leader-row"><span class="leader-rank">#5</span><span class="leader-name">Elena_M</span><span class="leader-score">4,150,000</span></div>
                    <div class="leader-row"><span class="leader-rank">#6</span><span class="leader-name">TapperPro</span><span class="leader-score">3,900,000</span></div>
                    <div class="leader-row"><span class="leader-rank">#7</span><span class="leader-name">Max_Nodes</span><span class="leader-score">2,750,000</span></div>
                    <div class="leader-row"><span class="leader-rank">#8</span><span class="leader-name">Serghei_K</span><span class="leader-score">1,980,000</span></div>
                    <div class="leader-row"><span class="leader-rank">#9</span><span class="leader-name">TokenHunter</span><span class="leader-score">1,200,000</span></div>
                    <div class="leader-row"><span class="leader-rank">#10</span><span class="leader-name">Cyber_Core</span><span class="leader-score">950,000</span></div>
                </div>
                <div class="user-rank-section">
                    <div class="user-row-special">
                        <span>Your Rank: #1,245</span>
                        <span id="user-rank-score">0 ZX</span>
                    </div>
                </div>
            </div>
        </div>

        <div class="bottom-nav">
            <div class="nav-item active" id="nav-tap" onclick="switchPage('tap')">
                <div class="nav-icon">🎮</div><div>Tap</div>
            </div>
            <div class="nav-item" id="nav-tasks" onclick="switchPage('tasks')">
                <div class="nav-icon">💼</div><div>Tasks</div>
            </div>
            <div class="nav-item" id="nav-wallet" onclick="switchPage('wallet')">
                <div class="nav-icon">👛</div><div>Wallet</div>
            </div>
            <div class="nav-item" id="nav-leaderboard" onclick="switchPage('leaderboard')">
                <div class="nav-icon">🏆</div><div>Rank</div>
            </div>
        </div>
    </div>

    <script>
        let score = parseInt(localStorage.getItem('zx_score')) || 0;
        let energy = parseInt(localStorage.getItem('zx_energy')) ?? 500;
        let adsLeft = parseInt(localStorage.getItem('zx_ads_left')) ?? 3;

        function updateUI() {
            document.getElementById('global-score').innerText = score.toLocaleString();
            document.getElementById('wallet-score').innerText = score.toLocaleString();
            document.getElementById('user-rank-score').innerText = score.toLocaleString() + " ZX";
            document.getElementById('energy-text').innerText = energy + " / 500";
            document.getElementById('energy-fill').style.width = (energy / 500 * 100) + "%";
            document.getElementById('ad-count').innerText = adsLeft;
            
            if(adsLeft === 0) {
                document.getElementById('ad-btn').disabled = true;
                document.getElementById('ad-btn').style.background = '#273549';
                document.getElementById('ad-btn').innerText = "❌ No Ads Available Today";
            }
        }

        function switchPage(pageId) {
            document.querySelectorAll('.page-content').forEach(p => p.classList.remove('active'));
            document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
            document.getElementById('page-' + pageId).classList.add('active');
            document.getElementById('nav-' + pageId).classList.add('active');
        }

        // Sistem de Tap legat direct de Energie
        document.getElementById('core-button').addEventListener('click', (e) => {
            if(energy <= 0) {
                alert("Core depleted, sэр! Watch an ad to recharge instantly.");
                return;
            }
            score += 1;
            energy -= 1;
            localStorage.setItem('zx_score', score);
            localStorage.setItem('zx_energy', energy);
            updateUI();

            const rect = document.getElementById('tap-zone').getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
            const floatText = document.createElement('div');
            floatText.classList.add('floating-text');
            floatText.innerText = '+1';
            floatText.style.left = x + "px";
            floatText.style.top = y + "px";
            document.getElementById('tap-zone').appendChild(floatText);
            setTimeout(() => floatText.remove(), 500);

            fetch('/api/webapp', { method: 'POST' }).catch(() => {});
        });

        // Reîncărcare prin Reclame (Limita de 3)
         Richmond = function watchAd() {
            if(adsLeft <= 0) return;
            alert("Simulating Premium Video Advertisement... Please wait 5 seconds.");
            setTimeout(() => {
                energy = 500;
                adsLeft -= 1;
                localStorage.setItem('zx_energy', energy);
                localStorage.setItem('zx_ads_left', adsLeft);
                updateUI();
                alert("Energy Restored to Maximum, sэр!");
            }, 2000);
        }

        // Conectare portofel mock-up Web3
        function connectWallet() {
            alert("Connecting to Secure Telegram Wallet API... Injection successful!");
        }

        function claimTask(id) {
            setTimeout(() => {
                if(id === 1) score += 50000;
                if(id === 2) score += 25000;
                localStorage.setItem('zx_score', score);
                updateUI();
                alert("Operation verified, rewards injected!");
            }, 1500);
        }

        // Inițializare automată la deschidere
        updateUI();
    </script>
</body>
</html>
`

func main() {
	botToken := "8744648391:AAHbsnd54wrv686PkLCbtj4ueBm4DqEB4vQ"

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic("❌ Eroare critică bot: ", err)
	}

	bot.Debug = false
	log.Printf("🤖 ZX-Elite Core Web3 este ONLINE pentru %s!", bot.Self.UserName)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(htmlContent))
	})

	http.HandleFunc("/api/webapp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == "POST" {
			atomic.AddInt64(&tapCount, 1)
			w.Write([]byte(`{"status":"success"}`))
			return
		}
	})

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal(err)
		}
	}()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil && update.Message.Text == "/start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"⚛️ *ZX-ELITE PLATFORM WEB3* ⚛️\n\n"+
					"🔷 Arhitectură stabilă distribuită 24/7 Cloud!\n\n"+
					"🎮 *Lansați aplicația din butonul din meniu:*")
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		}
	}
}
