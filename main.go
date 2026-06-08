package main

import (
	"log"
	"net/http"
	"os"
	"sync/atomic"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var tapCount int64

const htmlContent = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>ZX-Elite Reactor Core</title>
    <style>
        body {
            font-family: 'Segoe UI', sans-serif;
            background: radial-gradient(circle at center, #0d1b2a 0%, #010811 100%);
            color: #ffffff;
            margin: 0; padding: 0;
            display: flex; flex-direction: column; align-items: center; justify-content: space-between;
            height: 100vh; overflow: hidden;
            user-select: none; -webkit-user-select: none;
        }
        .header { width: 90%; display: flex; justify-content: space-between; align-items: center; padding-top: 20px; }
        .status-badge { background: rgba(0, 180, 216, 0.1); border: 1px solid #00b4d8; padding: 5px 12px; border-radius: 20px; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 1px; }
        .score-container { text-align: center; margin-top: 10px; }
        .score-label { font-size: 0.85rem; color: #00b4d8; text-transform: uppercase; letter-spacing: 2px; }
        .score-value { font-size: 3rem; font-weight: bold; text-shadow: 0 0 20px rgba(0, 180, 216, 0.6); margin: 5px 0; }
        .reactor-container { position: relative; display: flex; justify-content: center; align-items: center; width: 100%; height: 40vh; }
        .reactor-core { width: 180px; height: 180px; background: radial-gradient(circle, #00f5d4 0%, #00b4d8 50%, #03045e 100%); border-radius: 50%; box-shadow: 0 0 40px rgba(0, 245, 212, 0.6); cursor: pointer; transition: transform 0.05s ease; }
        .reactor-core:active { transform: scale(0.92); }
        .floating-text { position: absolute; color: #00f5d4; font-size: 1.8rem; font-weight: bold; pointer-events: none; animation: floatUp 0.6s ease-out forwards; text-shadow: 0 0 10px #00f5d4; }
        @keyframes floatUp { 0% { opacity: 1; transform: translateY(0) scale(1); } 100% { opacity: 0; transform: translateY(-80px) scale(1.2); } }
        .tasks-container { width: 90%; background: rgba(255, 255, 255, 0.03); border: 1px solid rgba(255, 255, 255, 0.07); border-radius: 16px; padding: 15px; margin-bottom: 25px; box-sizing: border-box; }
        .tasks-title { font-size: 1rem; margin-top: 0; margin-bottom: 12px; color: #00f5d4; text-transform: uppercase; letter-spacing: 1px; border-bottom: 1px solid rgba(255, 255, 255, 0.1); padding-bottom: 5px; }
        .task-row { display: flex; justify-content: space-between; align-items: center; background: rgba(255, 255, 255, 0.02); padding: 10px 12px; border-radius: 10px; margin-bottom: 8px; }
        .task-info { font-size: 0.85rem; }
        .task-reward { color: #00f5d4; font-weight: bold; font-size: 0.8rem; display: block; margin-top: 2px; }
        .task-btn { background: #00b4d8; color: #fff; border: none; padding: 6px 14px; border-radius: 8px; font-size: 0.8rem; font-weight: bold; cursor: pointer; text-decoration: none; }
    </style>
</head>
<body>
    <div class="header">
        <div class="status-badge">⚡ Core Active</div>
        <div style="font-size: 0.9rem; color: #aaa;">ZX-Elite v1.0</div>
    </div>
    <div class="score-container">
        <div class="score-label">Energy Extracted</div>
        <div class="score-value" id="score">0</div>
    </div>
    <div class="reactor-container" id="tap-zone">
        <div class="reactor-core" id="core-button"></div>
    </div>
    <div class="tasks-container">
        <div class="tasks-title">🛠️ Premium Operations</div>
        <div class="task-row">
            <div class="task-info">
                <span>🚀 Launch Partner App</span>
                <span class="task-reward">+50,000 ZX</span>
            </div>
            <a href="https://t.me/PartnerBot/app" target="_blank" class="task-btn" onclick="claimTask(1)">Start</a>
        </div>
        <div class="task-row">
            <div class="task-info">
                <span>📢 Join ZX Network Channel</span>
                <span class="task-reward">+25,000 ZX</span>
            </div>
            <a href="https://t.me/your_channel" target="_blank" class="task-btn" onclick="claimTask(2)">Join</a>
        </div>
    </div>
    <script>
        let score = parseInt(localStorage.getItem('zx_score')) || 0;
        document.getElementById('score').innerText = score.toLocaleString();
        const coreButton = document.getElementById('core-button');
        const tapZone = document.getElementById('tap-zone');

        coreButton.addEventListener('click', (e) => {
            score += 1;
            document.getElementById('score').innerText = score.toLocaleString();
            localStorage.setItem('zx_score', score);
            const rect = tapZone.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
            const floatText = document.createElement('div');
            floatText.classList.add('floating-text');
            floatText.innerText = '+1';
            floatText.style.left = x + "px";
            floatText.style.top = y + "px";
            tapZone.appendChild(floatText);
            setTimeout(() => { floatText.remove(); }, 600);

            fetch('/api/webapp', { method: 'POST' }).catch(err => console.log("Server error"));
        });

        function claimTask(id) {
            setTimeout(() => {
                if(id === 1) score += 50000;
                if(id === 2) score += 25000;
                document.getElementById('score').innerText = score.toLocaleString();
                localStorage.setItem('zx_score', score);
                alert("Operation verified, sэр!");
            }, 2000);
        }
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
	log.Printf("🤖 ZX-Elite Reactor Core este ONLINE pentru %s!", bot.Self.UserName)

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
				"⚛️ *ZX-ELITE REACTOR CORE* ⚛️\n\n"+
					"🔷 Sistemul Global este ONLINE 24/7!\n\n"+
					"🎮 *Apasă pe butonul de mai jos din meniu pentru lansare instant:*")
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		}
	}
}
