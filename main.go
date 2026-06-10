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
<!-- TON Connect UI -->
<script src="https://unpkg.com/@tonconnect/ui@latest/dist/tonconnect-ui.min.js"></script>
<!-- ADSGRAM script – obligatoriu pentru reclame reale -->
<script src="https://sad.adsgram.ai/js/sad.min.js"></script>
<style>
:root{
  --bg:#070b10; --panel:#0d1420; --panel2:#111b2b;
  --green:#00f5d4; --green2:#00ff87; --text:#f3fffc;
  --muted:#87a39d; --purple:#9d4dff; --orange:#ff9f43;
  --blue:#3b82f6; --ton:#0098EA;
  --border:rgba(255,255,255,.08);
}
*{ margin:0; padding:0; box-sizing:border-box; -webkit-tap-highlight-color:transparent!important; }
html,body{ overscroll-behavior:none; -webkit-text-size-adjust:none; text-size-adjust:none; }
body{
  background: radial-gradient(circle at top left,rgba(0,245,212,.12),transparent 40%),
              radial-gradient(circle at top right,rgba(0,255,135,.08),transparent 40%), #070b10;
  color:var(--text); font-family:-apple-system,'Inter','Segoe UI',sans-serif;
  min-height:100vh; min-height:-webkit-fill-available;
  overflow-x:hidden; -webkit-user-select:none; user-select:none; touch-action:pan-y;
}
body::before{
  content:""; position:fixed; inset:0; pointer-events:none; z-index:0;
  background-image:linear-gradient(rgba(255,255,255,.03) 1px,transparent 1px),
                   linear-gradient(90deg,rgba(255,255,255,.03) 1px,transparent 1px);
  background-size:40px 40px;
}
.app{ position:relative; z-index:1; max-width:600px; margin:auto; padding-bottom:130px; }
.header{
  position:sticky; top:0; display:flex; justify-content:space-between; align-items:center;
  padding:14px 16px; backdrop-filter:blur(15px); -webkit-backdrop-filter:blur(15px);
  background:rgba(7,11,16,.9); border-bottom:1px solid var(--border); z-index:100;
}
.userBox{ display:flex; flex-direction:column; gap:2px; }
.userLabel{ font-size:11px; color:var(--muted); text-transform:uppercase; letter-spacing:.5px; }
.userName{ font-weight:800; font-size:15px; color:#fff; }
.netBadge{
  background:rgba(0,245,212,.15); color:var(--green);
  border:1px solid rgba(0,245,212,.3); border-radius:999px;
  padding:8px 12px; font-size:11px; font-weight:800;
  box-shadow:0 0 18px rgba(0,245,212,.2);
}
.page{ padding:16px; display:flex; flex-direction:column; gap:14px; }
.section{
  background:linear-gradient(180deg,rgba(17,27,43,.96),rgba(13,20,32,.96));
  border-radius:24px; border:1px solid var(--border); padding:20px;
}
.balance-container{ text-align:center; margin-bottom:8px; margin-top:4px; }
.balanceTitle{ color:var(--muted); font-size:11px; letter-spacing:2px; text-transform:uppercase; }
.balanceValue{
  margin-top:6px; font-size:44px; font-weight:900; color:white;
  text-shadow:0 0 20px rgba(0,245,212,.5),0 0 40px rgba(0,245,212,.25);
  letter-spacing:-1px; line-height:1;
}
.passive-badge{
  margin-top:6px; display:inline-flex; align-items:center; gap:5px;
  background:rgba(255,159,67,.12); border:1px solid rgba(255,159,67,.25);
  color:var(--orange); border-radius:999px; padding:4px 10px;
  font-size:11px; font-weight:700;
}
.coinArea{
  display:flex; flex-direction:column; align-items:center;
  justify-content:center; padding:8px 0; position:relative;
}
.coin{
  width:min(240px,62vw); height:min(240px,62vw); cursor:pointer;
  transition:transform .07s cubic-bezier(.25,.46,.45,.94);
  -webkit-user-select:none; user-select:none;
  -webkit-tap-highlight-color:transparent!important;
  outline:none; touch-action:manipulation; display:block; will-change:transform;
}
.coin:active{ transform:scale(.91)!important; }
.energyRow{ margin-top:16px; display:flex; gap:10px; align-items:center; }
.energyBox{ flex:1; }
.energyLabel{ font-size:11px; color:var(--muted); margin-bottom:5px; display:flex; justify-content:space-between; }
.energyBar{ width:100%; height:12px; border-radius:999px; overflow:hidden; background:#08111b; border:1px solid rgba(255,255,255,.06); }
.energyFill{ width:100%; height:100%; background:linear-gradient(90deg,var(--green),var(--green2)); box-shadow:0 0 12px rgba(0,245,212,.3); transition:width .15s ease; }
.btn{
  border:none; cursor:pointer; padding:11px 16px; border-radius:14px;
  font-weight:800; font-size:13px;
  background:linear-gradient(135deg,var(--green),var(--green2));
  color:#04120d; box-shadow:0 0 20px rgba(0,245,212,.2);
  transition:filter .15s,transform .1s; white-space:nowrap;
  -webkit-tap-highlight-color:transparent!important;
}
.btn:active{ transform:scale(.96); }
.btn:hover{ filter:brightness(1.07); }
.btn:disabled{ opacity:.45; cursor:not-allowed; pointer-events:none; }
.btn-secondary{ background:#1b2a3d; color:#a0c4c0; box-shadow:none; border:1px solid rgba(255,255,255,.07); }
.btn-danger{ background:linear-gradient(135deg,#c0392b,#e74c3c); color:#fff; box-shadow:0 0 20px rgba(231,76,60,.25); }
.btn-purple{ background:linear-gradient(135deg,#7c3aed,#9d4dff); color:#fff; }
.btn-orange{ background:linear-gradient(135deg,#e67e22,#ff9f43); color:#fff; }
.btn-ton{ background:linear-gradient(135deg,#0077c2,#0098EA); color:#fff; box-shadow:0 0 20px rgba(0,152,234,.3); }
.btn-sm{ padding:8px 14px; font-size:12px; border-radius:10px; }
.upgrades{ margin-top:18px; display:grid; grid-template-columns:repeat(2,1fr); gap:10px; }
.upgradeCard{ background:rgba(0,0,0,.22); border:1px solid rgba(255,255,255,.06); border-radius:18px; padding:14px; }
.upgradeCardFull{ background:rgba(0,0,0,.22); border:1px solid rgba(255,159,67,.15); border-radius:18px; padding:14px; margin-top:10px; }
.upgradeTitle{ font-size:13px; font-weight:800; margin-bottom:4px; }
.upgradeSub{ font-size:11px; color:var(--muted); margin-bottom:10px; }
.bottomNav{ position:fixed; left:0; right:0; bottom:0; padding:8px 12px; padding-bottom:max(8px,env(safe-area-inset-bottom)); z-index:999; }
.bottomInner{
  max-width:600px; margin:auto; display:grid; grid-template-columns:repeat(5,1fr); gap:6px;
  background:rgba(8,14,22,.97); border:1px solid rgba(255,255,255,.06);
  border-radius:22px; padding:8px; backdrop-filter:blur(20px); -webkit-backdrop-filter:blur(20px);
}
.tabBtn{ background:none; border:none; color:#87a39d; padding:8px 4px; border-radius:14px; cursor:pointer; font-weight:800; font-size:10px; transition:.2s; line-height:1.3; -webkit-tap-highlight-color:transparent!important; }
.tabBtn.active{ color:#04120d; background:linear-gradient(135deg,var(--green),var(--green2)); }
.hidden{ display:none!important; }
.verify-btn{ display:none!important; }
.floatGain{
  position:fixed; color:white; font-weight:900; font-size:22px;
  pointer-events:none; z-index:99999;
  text-shadow:-1px -1px 0 #000,1px -1px 0 #000,-1px 1px 0 #000,1px 1px 0 #000,0 0 12px rgba(0,255,135,.9);
  animation:floatUp .75s ease-out forwards; will-change:transform,opacity;
}
@keyframes floatUp{ from{opacity:1;transform:translateY(0) scale(1)} to{opacity:0;transform:translateY(-90px) scale(.75)} }
.leaderboardItem{ display:flex; align-items:center; padding:12px 0; border-bottom:1px solid rgba(255,255,255,.05); gap:8px; }
.leaderboardRank{ width:28px; color:var(--green); font-weight:900; font-size:14px; }
.leaderboardName{ flex:1; font-size:13px; }
.leaderboardBalance{ color:#fff; font-weight:700; font-size:13px; }
.taskCard{
  background:rgba(0,0,0,.2); border:1px solid rgba(255,255,255,.06);
  border-radius:18px; padding:14px 16px;
  display:flex; justify-content:space-between; align-items:center; gap:12px;
  margin-bottom:10px;
}
.taskInfo{ flex:1; }
.taskTitle{ font-size:14px; font-weight:800; margin-bottom:3px; }
.taskDesc{ font-size:11px; color:var(--muted); }
.taskReward{ font-size:12px; color:var(--green); font-weight:700; margin-top:3px; }
.checkinGrid{ display:grid; grid-template-columns:repeat(7,1fr); gap:6px; margin-top:14px; }
.checkinDay{
  aspect-ratio:1; border-radius:10px; display:flex; flex-direction:column;
  align-items:center; justify-content:center; font-size:9px; font-weight:700;
  background:rgba(0,0,0,.2); border:1px solid rgba(255,255,255,.06); gap:2px;
}
.checkinDay.done{ background:rgba(0,245,212,.15); border-color:rgba(0,245,212,.3); color:var(--green); }
.checkinDay.today{ border-color:var(--orange); box-shadow:0 0 10px rgba(255,159,67,.3); }
.referralBox{ background:rgba(0,0,0,.2); border:1px solid rgba(157,77,255,.2); border-radius:18px; padding:16px; }
.referralCode{
  font-size:22px; font-weight:900; color:var(--purple); letter-spacing:3px;
  text-align:center; padding:12px; background:rgba(157,77,255,.1);
  border-radius:12px; margin:10px 0; text-shadow:0 0 20px rgba(157,77,255,.5);
}
.sectionTitle{ font-size:16px; font-weight:900; margin-bottom:16px; color:#d9fff5; }
.stat-row{ display:flex; justify-content:space-between; align-items:center; padding:8px 0; border-bottom:1px solid rgba(255,255,255,.05); font-size:13px; }
.stat-label{ color:var(--muted); }
.stat-value{ font-weight:700; }
.toast{
  position:fixed; top:70px; left:50%; transform:translateX(-50%);
  background:#1a2940; border:1px solid rgba(0,245,212,.3); color:var(--green);
  padding:10px 20px; border-radius:999px; font-weight:700; font-size:13px;
  z-index:99999; pointer-events:none; opacity:0; transition:opacity .3s;
  white-space:nowrap; max-width:90vw; text-align:center;
}
.toast.show{ opacity:1; }
/* TON Connect button container */
#ton-connect-btn{ display:block; }
.ton-wallet-card{
  background:rgba(0,152,234,.08); border:1px solid rgba(0,152,234,.25);
  border-radius:18px; padding:16px; margin-bottom:14px;
}
.ton-addr{
  font-size:12px; color:#7dd3fc; word-break:break-all;
  background:rgba(0,0,0,.2); padding:10px; border-radius:10px;
  margin-top:10px; font-family:monospace;
}
.wallet-status{ display:flex; align-items:center; gap:8px; font-size:13px; margin-bottom:12px; }
.wallet-dot{ width:8px; height:8px; border-radius:50%; background:#666; }
.wallet-dot.connected{ background:var(--green); box-shadow:0 0 6px var(--green); }
/* Adsgram loading overlay */
.ad-loading{
  display:none; position:fixed; inset:0; background:rgba(0,0,0,.7);
  backdrop-filter:blur(8px); z-index:9000; align-items:center; justify-content:center;
  flex-direction:column; gap:16px;
}
.ad-loading.show{ display:flex; }
.ad-spinner{
  width:40px; height:40px; border:3px solid rgba(0,245,212,.2);
  border-top-color:var(--green); border-radius:50%; animation:spin .8s linear infinite;
}
@keyframes spin{ to{transform:rotate(360deg)} }
</style>
</head>
<body>
<div id="toastEl" class="toast"></div>

<!-- Ad loading overlay -->
<div id="adLoading" class="ad-loading">
  <div class="ad-spinner"></div>
  <div style="color:var(--muted);font-size:13px;">Se încarcă reclama...</div>
</div>

<div class="app">
<!-- HEADER -->
<header class="header">
  <div class="userBox">
    <div class="userLabel">User Core</div>
    <div id="telegramUser" class="userName">Guest</div>
  </div>
  <div class="netBadge">⚡ ZX-NET LIVE</div>
</header>

<div class="page">

<!-- ════════ GENERATOR TAB ════════ -->
<div id="generatorTab" class="section">
  <div class="balance-container">
    <div class="balanceTitle">Total ZX Tokens</div>
    <div id="balanceDisplay" class="balanceValue">0</div>
    <div id="passiveBadge" class="passive-badge hidden">⚙️ <span id="passiveRate">0</span> ZX/oră</div>
  </div>
  <div class="coinArea">
    <svg id="coin" class="coin" viewBox="0 0 500 500" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <radialGradient id="cg" cx="50%" cy="40%" r="55%">
          <stop offset="0%" stop-color="#7affd8"/>
          <stop offset="55%" stop-color="#00ff87"/>
          <stop offset="100%" stop-color="#00a85a"/>
        </radialGradient>
        <filter id="glow" x="-30%" y="-30%" width="160%" height="160%">
          <feGaussianBlur stdDeviation="5" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
        <radialGradient id="ig" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="rgba(0,255,135,0.15)"/>
          <stop offset="100%" stop-color="transparent"/>
        </radialGradient>
      </defs>
      <circle cx="250" cy="250" r="235" fill="none" stroke="rgba(0,255,135,0.08)" stroke-width="2"/>
      <circle cx="250" cy="250" r="218" fill="none" stroke="#b0c4de" stroke-width="6"/>
      <circle cx="250" cy="250" r="205" fill="none" stroke="url(#cg)" stroke-width="4"/>
      <circle cx="250" cy="250" r="190" fill="#05101c" stroke="#00ff87" stroke-width="1.5"/>
      <circle cx="250" cy="250" r="190" fill="url(#ig)"/>
      <rect x="130" y="135" width="240" height="230" rx="8" fill="none" stroke="rgba(0,255,135,0.12)" stroke-width="1"/>
      <g stroke="#00ff87" stroke-width="1" opacity=".35">
        <line x1="155" y1="155" x2="155" y2="180"/><line x1="175" y1="155" x2="175" y2="180"/>
        <line x1="195" y1="155" x2="195" y2="180"/><line x1="215" y1="155" x2="215" y2="180"/>
        <line x1="235" y1="155" x2="235" y2="180"/><line x1="255" y1="155" x2="255" y2="180"/>
        <line x1="275" y1="155" x2="275" y2="180"/><line x1="295" y1="155" x2="295" y2="180"/>
        <line x1="315" y1="155" x2="315" y2="180"/><line x1="335" y1="155" x2="335" y2="180"/>
      </g>
      <g stroke="#00ff87" stroke-width="1" opacity=".35">
        <line x1="155" y1="345" x2="155" y2="320"/><line x1="175" y1="345" x2="175" y2="320"/>
        <line x1="195" y1="345" x2="195" y2="320"/><line x1="215" y1="345" x2="215" y2="320"/>
        <line x1="235" y1="345" x2="235" y2="320"/><line x1="255" y1="345" x2="255" y2="320"/>
        <line x1="275" y1="345" x2="275" y2="320"/><line x1="295" y1="345" x2="295" y2="320"/>
        <line x1="315" y1="345" x2="315" y2="320"/><line x1="335" y1="345" x2="335" y2="320"/>
      </g>
      <rect x="228" y="228" width="44" height="44" fill="none" stroke="#00ff87" stroke-width="1.5" rx="4"/>
      <circle cx="250" cy="250" r="14" fill="#00ff87" opacity=".18"/>
      <g filter="url(#glow)">
        <path d="M175 185 L325 185 L175 315 L325 315" fill="none" stroke="#fff" stroke-width="20" stroke-linecap="round" stroke-linejoin="round"/>
        <path d="M175 185 L325 315 M325 185 L175 315" fill="none" stroke="#fff" stroke-width="20" stroke-linecap="round" stroke-linejoin="round"/>
      </g>
    </svg>
  </div>
  <div class="energyRow">
    <div class="energyBox">
      <div class="energyLabel"><span>⚡ Energie</span><span id="energyText">500/500</span></div>
      <div class="energyBar"><div id="energyFill" class="energyFill"></div></div>
    </div>
    <button id="rechargeBtn" class="btn btn-secondary">Încarcă</button>
  </div>
  <div class="upgrades">
    <div class="upgradeCard">
      <div class="upgradeTitle">👆 Multitap</div>
      <div class="upgradeSub">Lv.<span id="tapLvl">0</span> · <span id="tapCost">1.000</span> ZX</div>
      <button id="buyTap" class="btn" style="width:100%">Upgrade</button>
    </div>
    <div class="upgradeCard">
      <div class="upgradeTitle">⚡ Max Energie</div>
      <div class="upgradeSub">Lv.<span id="energyLvl">0</span> · <span id="energyCost">2.500</span> ZX</div>
      <button id="buyEnergy" class="btn" style="width:100%">Upgrade</button>
    </div>
  </div>
  <div class="upgradeCardFull">
    <div class="upgradeTitle">⚙️ Passive Mining — Lv.<span id="passiveLvl">0</span></div>
    <div class="upgradeSub"><span id="passiveRateDisp">0</span> ZX/oră · Cost: <span id="passiveCost">5.000</span> ZX</div>
    <button id="buyPassive" class="btn btn-orange" style="width:100%;margin-top:8px">Upgrade Passive Mining</button>
  </div>
</div>

<!-- ════════ TASKS TAB ════════ -->
<div id="tasksTab" class="section hidden">
  <div class="sectionTitle">💼 Tasks & Misiuni</div>

  <!-- Daily check-in -->
  <div class="referralBox" style="margin-bottom:14px">
    <div class="upgradeTitle">📅 Daily Check-in</div>
    <div style="font-size:12px;color:var(--muted);margin:6px 0 10px">
      Streak: <strong id="streakDisp">0</strong> zile consecutive
    </div>
    <div class="checkinGrid" id="checkinGrid"></div>
    <button id="checkinBtn" class="btn" style="width:100%;margin-top:14px">🎁 Revendică Recompensa Zilnică</button>
  </div>

  <!-- Adsgram ad -->
  <div class="taskCard" id="taskAdCard">
    <div class="taskInfo">
      <div class="taskTitle">📺 Vizionează Reclamă (Adsgram)</div>
      <div class="taskDesc">Reclamă reală, recompensă garantată</div>
      <div class="taskReward">+<span id="rewardAdDisp">1.000</span> ZX per reclamă</div>
    </div>
    <button id="watchAdBtn" class="btn btn-sm">▶ Watch</button>
  </div>

  <!-- Canal 1 -->
  <div class="taskCard" id="taskCh1Card">
    <div class="taskInfo">
      <div class="taskTitle">📢 Abonare Canal Telegram</div>
      <div class="taskDesc" id="ch1Desc">Abonează-te la canalul oficial</div>
      <div class="taskReward">+<span id="rewardCh1Disp">2.500</span> ZX</div>
    </div>
    <div style="display:flex;flex-direction:column;gap:6px;align-items:flex-end">
      <a id="ch1Link" href="#" target="_blank" class="btn btn-sm btn-secondary"
         onclick="showVerifyBtn(event,'channel1')">Deschide</a>
      <button id="taskBtn_channel1" class="btn btn-sm verify-btn"
              onclick="claimTask('channel1', this)">Verifică</button>
    </div>
  </div>

  <!-- Canal 2 (opțional) -->
  <div class="taskCard hidden" id="taskCh2Card">
    <div class="taskInfo">
      <div class="taskTitle">📢 Al doilea Canal</div>
      <div class="taskDesc" id="ch2Desc">Abonează-te la canalul secundar</div>
      <div class="taskReward">+<span id="rewardCh2Disp">1.500</span> ZX</div>
    </div>
    <div style="display:flex;flex-direction:column;gap:6px;align-items:flex-end">
      <a id="ch2Link" href="#" target="_blank" class="btn btn-sm btn-secondary"
         onclick="showVerifyBtn(event,'channel2')">Deschide</a>
      <button id="taskBtn_channel2" class="btn btn-sm verify-btn"
              onclick="claimTask('channel2', this)">Verifică</button>
    </div>
  </div>

  <!-- Twitter -->
  <div class="taskCard" id="taskTwCard">
    <div class="taskInfo">
      <div class="taskTitle">🐦 Follow Twitter / X</div>
      <div class="taskDesc">Urmărește contul oficial</div>
      <div class="taskReward">+<span id="rewardTwDisp">750</span> ZX</div>
    </div>
    <div style="display:flex;flex-direction:column;gap:6px;align-items:flex-end">
      <a id="twLink" href="#" target="_blank" class="btn btn-sm btn-secondary"
         onclick="showVerifyBtn(event,'twitter')">Deschide</a>
      <button id="taskBtn_twitter" class="btn btn-sm verify-btn"
              onclick="claimTask('twitter', this)">Revendică</button>
    </div>
  </div>

  <!-- Partner bot -->
  <div class="taskCard" id="taskPartnerCard">
    <div class="taskInfo">
      <div class="taskTitle">🤖 Start Bot Partener</div>
      <div class="taskDesc">Activează botul partener</div>
      <div class="taskReward">+<span id="rewardPartnerDisp">2.000</span> ZX</div>
    </div>
    <div style="display:flex;flex-direction:column;gap:6px;align-items:flex-end">
      <a id="partnerLink" href="#" target="_blank" class="btn btn-sm btn-secondary"
         onclick="showVerifyBtn(event,'partner')">Deschide</a>
      <button id="taskBtn_partner" class="btn btn-sm verify-btn"
              onclick="claimTask('partner', this)">Revendică</button>
    </div>
  </div>
</div>

<!-- ════════ REFERRAL TAB ════════ -->
<div id="referralTab" class="section hidden">
  <div class="sectionTitle">👥 Sistem Referrals</div>
  <div class="referralBox">
    <div style="font-size:12px;color:var(--muted);text-align:center">Codul tău unic de invitație</div>
    <div id="myRefCode" class="referralCode">LOADING</div>
    <button id="copyRefLink" class="btn btn-purple" style="width:100%">📋 Copiază Link</button>
  </div>
  <div class="stat-row" style="margin-top:16px">
    <span class="stat-label">Jucători invitați</span><span class="stat-value" id="refCount">0</span>
  </div>
  <div class="stat-row">
    <span class="stat-label">Câștiguri referral</span><span class="stat-value" id="refEarnings">0 ZX</span>
  </div>
  <div class="stat-row">
    <span class="stat-label">Bonus per invitat</span><span class="stat-value" style="color:var(--green)">500 ZX</span>
  </div>
  <div style="margin-top:18px">
    <div style="font-size:14px;font-weight:800;margin-bottom:10px">Introdu cod primit</div>
    <div style="display:flex;gap:8px">
      <input id="refInput" type="text" placeholder="ZXABCD12" maxlength="8"
        style="flex:1;background:rgba(0,0,0,.25);border:1px solid rgba(255,255,255,.1);border-radius:12px;padding:12px 14px;color:white;font-size:14px;font-weight:700;letter-spacing:2px;outline:none;text-transform:uppercase"/>
      <button id="claimRef" class="btn btn-purple">Aplică</button>
    </div>
    <div id="refStatus" style="font-size:12px;margin-top:8px;color:var(--muted)"></div>
  </div>
  <div style="margin-top:16px;font-size:12px;color:var(--muted);line-height:1.6">
    ℹ️ Invită prietenii și primești <strong style="color:var(--green)">500 ZX</strong> per jucător. Ei primesc <strong style="color:var(--green)">1000 ZX</strong> bonus.
  </div>
</div>

<!-- ════════ WALLET TAB ════════ -->
<div id="walletTab" class="section hidden">
  <div class="sectionTitle">👛 TON Wallet & Balanță</div>

  <!-- TON Connect card -->
  <div class="ton-wallet-card">
    <div class="wallet-status">
      <div class="wallet-dot" id="walletDot"></div>
      <span id="walletStatusTxt">Wallet neconectat</span>
    </div>
    <!-- TON Connect UI injectează butonul aici -->
    <div id="ton-connect-btn">
      <div style="color:var(--muted);font-size:13px;padding:10px;">Se încarcă wallet...</div>
    </div>
    <div id="tonAddrDisplay" class="ton-addr hidden"></div>
    <div id="tonBalanceRow" class="hidden" style="margin-top:10px;font-size:12px;color:var(--muted)">
      Adresa salvată pentru retrageri viitoare ✅
    </div>
  </div>

  <!-- ZX Balance -->
  <div style="background:rgba(0,0,0,.18);border-radius:18px;padding:16px;border:1px solid rgba(255,255,255,.05)">
    <div style="font-size:12px;color:var(--muted)">ZX Balance</div>
    <div id="walletBalance" style="margin-top:6px;font-size:32px;font-weight:900">0</div>
  </div>

  <!-- History -->
  <div style="margin-top:16px">
    <div style="font-size:14px;font-weight:800;margin-bottom:12px">📋 Istoric Retrageri</div>
    <table style="width:100%;border-collapse:collapse;font-size:12px">
      <thead>
        <tr style="color:var(--muted)">
          <th style="text-align:left;padding:8px 4px;border-bottom:1px solid rgba(255,255,255,.08)">Data</th>
          <th style="border-bottom:1px solid rgba(255,255,255,.08)">ZX</th>
          <th style="border-bottom:1px solid rgba(255,255,255,.08)">Status</th>
        </tr>
      </thead>
      <tbody id="withdrawTable">
        <tr>
          <td style="padding:10px 4px">2026-06-01</td>
          <td style="text-align:center">120.000</td>
          <td style="text-align:center;color:#ffd166">În așteptare</td>
        </tr>
        <tr>
          <td style="padding:10px 4px">2026-05-24</td>
          <td style="text-align:center">50.000</td>
          <td style="text-align:center;color:#00ff87">Finalizat</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div style="margin-top:20px">
    <button id="deleteAccountBtn" class="btn btn-danger" style="width:100%">🗑️ Șterge Datele Locale</button>
  </div>
</div>

<!-- ════════ RANK TAB ════════ -->
<div id="rankTab" class="section hidden">
  <div class="sectionTitle">🏆 Global Leaderboard</div>
  <div id="leaderboard"><div style="color:var(--muted);text-align:center;padding:20px">Se încarcă...</div></div>
  <div style="margin-top:20px;height:1px;background:linear-gradient(90deg,transparent,#9d4dff,transparent);box-shadow:0 0 15px #9d4dff"></div>
  <div style="margin-top:16px;background:rgba(157,77,255,.08);border:1px solid rgba(157,77,255,.2);border-radius:18px;padding:16px">
    <div style="font-size:12px;color:#b79cff">Poziția ta</div>
    <div id="myRank" style="margin-top:6px;font-size:28px;font-weight:900">#-</div>
    <div id="myBalanceRank" style="margin-top:4px;color:#e9dbff;font-size:14px">0 ZX</div>
  </div>
</div>

</div><!-- end .page -->
</div><!-- end .app -->

<!-- Recharge modal -->
<div id="rechargeModal" style="display:none;position:fixed;inset:0;background:rgba(0,0,0,.85);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);z-index:8000;justify-content:center;align-items:center">
  <div style="width:90%;max-width:400px;background:#0f1d2e;border-radius:24px;border:1px solid rgba(0,245,212,.2);padding:24px">
    <h2 style="margin-bottom:10px">📺 Reîncarcă Energia</h2>
    <p style="color:var(--muted);font-size:13px">Vizionează 3 reclame pentru energie completă.</p>
    <div id="adCounter" style="margin-top:14px;font-size:28px;font-weight:900;text-align:center">0/3</div>
    <div style="width:100%;height:8px;background:#08111b;border-radius:999px;margin-top:10px;overflow:hidden">
      <div id="adProgress" style="height:100%;width:0%;background:linear-gradient(90deg,var(--green),var(--green2));transition:width .4s"></div>
    </div>
    <button id="watchRechargeAd" class="btn" style="width:100%;margin-top:18px">▶ Watch Ad</button>
    <button id="closeRecharge" class="btn btn-secondary" style="width:100%;margin-top:10px">Închide</button>
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
(function(){
'use strict';

// ─── Telegram init ───────────────────────────────────────────────────────────
var tg = (window.Telegram && window.Telegram.WebApp) ? window.Telegram.WebApp : null;
var cu = { username:'guest', firstName:'Guest', id:0 };
if(tg){
  tg.ready(); tg.expand();
  if(tg.disableVerticalSwipes) tg.disableVerticalSwipes();
  var u = tg.initDataUnsafe ? tg.initDataUnsafe.user : null;
  if(u){
    cu.username = u.username || ('id'+u.id);
    cu.firstName = u.first_name || u.username || 'Player';
    cu.id = u.id || 0;
  }
}
document.getElementById('telegramUser').textContent = cu.firstName;

// ─── App config (loaded from server) ─────────────────────────────────────────
var cfg = {
  adsgramBlockId:'', linkChannel:'#', linkChannel2:'#', linkTwitter:'#', linkPartner:'#',
  channel2Enabled:false,
  rewardChannel:2500, rewardChannel2:1500, rewardTwitter:750, rewardPartner:2000, rewardAd:1000,
  appUrl:''
};

fetch('/api/config').then(function(r){ return r.json(); }).then(function(d){
  cfg = d;
  // Update UI links and rewards
  var ch1 = document.getElementById('ch1Link');
  if(ch1) ch1.href = cfg.linkChannel;
  var ch2Card = document.getElementById('taskCh2Card');
  if(cfg.channel2Enabled && ch2Card) ch2Card.classList.remove('hidden');
  var ch2l = document.getElementById('ch2Link');
  if(ch2l) ch2l.href = cfg.linkChannel2;
  var twl = document.getElementById('twLink');
  if(twl) twl.href = cfg.linkTwitter;
  var pl = document.getElementById('partnerLink');
  if(pl) pl.href = cfg.linkPartner;
  // Update reward displays
  function setDisp(id, val){ var el=document.getElementById(id); if(el) el.textContent=fmt(val); }
  setDisp('rewardAdDisp', cfg.rewardAd);
  setDisp('rewardCh1Disp', cfg.rewardChannel);
  setDisp('rewardCh2Disp', cfg.rewardChannel2);
  setDisp('rewardTwDisp', cfg.rewardTwitter);
  setDisp('rewardPartnerDisp', cfg.rewardPartner);
  // Init Adsgram after config loaded
  initAdsgram();
  // Init TON Connect after config loaded
  initTonConnect();
  // Restore verify buttons visibility
  restoreVerifyButtons();
}).catch(function(){
  initAdsgram();
  initTonConnect();
  restoreVerifyButtons();
});

// ─── State ────────────────────────────────────────────────────────────────────
var SK = 'zxnet-v3';
var raw = {};
try{ raw = JSON.parse(localStorage.getItem(SK)||'{}'); }catch(e){}
var state = {
  balance:       raw.balance       !== undefined ? raw.balance : 0,
  energy:        raw.energy        !== undefined ? raw.energy  : 500,
  maxEnergy:     raw.maxEnergy     !== undefined ? raw.maxEnergy : 500,
  tapLevel:      raw.tapLevel      || 0,
  energyLevel:   raw.energyLevel   || 0,
  passiveLevel:  raw.passiveLevel  || 0,
  claimedTasks:  raw.claimedTasks  || {},
  checkinStreak: raw.checkinStreak || 0,
  lastCheckin:   raw.lastCheckin   || '',
  referralCode:  raw.referralCode  || '',
  referredByDone: raw.referredByDone || false,
  walletAddress: raw.walletAddress || '',
  linkClicked:   raw.linkClicked   || {}   // { channel1: true, channel2: true, twitter: true, partner: true }
};

function save(){ try{ localStorage.setItem(SK, JSON.stringify(state)); }catch(e){} }

// ─── Helpers ──────────────────────────────────────────────────────────────────
function fmt(v){ return Number(v).toLocaleString('ro-RO'); }

var _toast;
function toast(msg, dur){
  var el = document.getElementById('toastEl');
  el.textContent = msg; el.classList.add('show');
  clearTimeout(_toast);
  _toast = setTimeout(function(){ el.classList.remove('show'); }, dur||2500);
}

// ─── Costs ────────────────────────────────────────────────────────────────────
function tapCost()     { return Math.round(1000 * Math.pow(state.tapLevel+1, 2.2)); }
function energyCost()  { return Math.round(2500 * Math.pow(state.energyLevel+1, 2.2)); }
function passiveCost() { return Math.round(5000 * Math.pow(state.passiveLevel+1, 2.5)); }
function pph(lvl){ return lvl<=0 ? 0 : Math.round(100 * Math.pow(1.8, lvl-1)); }

// ─── UI update ────────────────────────────────────────────────────────────────
function updateUI(){
  document.getElementById('balanceDisplay').textContent = fmt(state.balance);
  document.getElementById('walletBalance').textContent  = fmt(state.balance);
  var pct = state.maxEnergy > 0 ? (state.energy/state.maxEnergy)*100 : 0;
  document.getElementById('energyFill').style.width = pct+'%';
  document.getElementById('energyText').textContent = state.energy+'/'+state.maxEnergy;
  document.getElementById('tapLvl').textContent     = state.tapLevel;
  document.getElementById('tapCost').textContent    = fmt(tapCost());
  document.getElementById('buyTap').disabled        = state.balance < tapCost();
  document.getElementById('energyLvl').textContent  = state.energyLevel;
  document.getElementById('energyCost').textContent = fmt(energyCost());
  document.getElementById('buyEnergy').disabled     = state.balance < energyCost();
  document.getElementById('passiveLvl').textContent     = state.passiveLevel;
  document.getElementById('passiveCost').textContent    = fmt(passiveCost());
  document.getElementById('passiveRateDisp').textContent= fmt(pph(state.passiveLevel));
  document.getElementById('buyPassive').disabled        = state.balance < passiveCost();
  var ph = pph(state.passiveLevel);
  var badge = document.getElementById('passiveBadge');
  document.getElementById('passiveRate').textContent = fmt(ph);
  if(ph>0) badge.classList.remove('hidden'); else badge.classList.add('hidden');
  document.getElementById('streakDisp').textContent = state.checkinStreak;
  updateCheckinGrid();
  restoreTaskButtons();
  restoreVerifyButtons();
}

function updateCheckinGrid(){
  var grid = document.getElementById('checkinGrid');
  grid.innerHTML = '';
  var today = new Date().toISOString().split('T')[0];
  for(var i=1; i<=7; i++){
    var d = document.createElement('div');
    d.className = 'checkinDay';
    var r = Math.min(500+100*i, 3000);
    d.innerHTML = '<span>D'+i+'</span><span style="font-size:8px;color:var(--green)">+'+(r>=1000?Math.round(r/1000)+'k':r)+'</span>';
    if(i <= state.checkinStreak) d.classList.add('done');
    if(i === state.checkinStreak+1) d.classList.add('today');
    grid.appendChild(d);
  }
  var btn = document.getElementById('checkinBtn');
  if(state.lastCheckin === today){ btn.textContent='✅ Revine mâine'; btn.disabled=true; }
  else { btn.textContent='🎁 Revendică Recompensa Zilnică'; btn.disabled=false; }
}

function restoreTaskButtons(){
  var tasks = { channel1:'taskBtn_channel1', channel2:'taskBtn_channel2', twitter:'taskBtn_twitter', partner:'taskBtn_partner' };
  for(var id in tasks){
    var btn = document.getElementById(tasks[id]);
    if(btn && state.claimedTasks[id]){ btn.textContent='✅ Revendicat'; btn.disabled=true; }
  }
}

// ─── Verify buttons logic ─────────────────────────────────────────────────────
function showVerifyBtn(e, taskId){
  // Let the link open normally, then reveal the claim button
  var btn = document.getElementById('taskBtn_' + taskId);
  if(btn){
    btn.style.display = 'inline-block';
    state.linkClicked[taskId] = true;
    save();
  }
}

function restoreVerifyButtons(){
  var tasks = ['channel1','channel2','twitter','partner'];
  tasks.forEach(function(id){
    var btn = document.getElementById('taskBtn_'+id);
    if(btn && state.linkClicked && state.linkClicked[id]){
      btn.style.display = 'inline-block';
    }
  });
}

// ─── Server sync ──────────────────────────────────────────────────────────────
var syncT = null;
function syncNow(immediate){
  if(cu.username==='guest') return;
  clearTimeout(syncT);
  syncT = setTimeout(function(){
    fetch('/api/sync',{
      method:'POST', headers:{'Content-Type':'application/json'},
      body:JSON.stringify({ username:cu.username, balance:state.balance,
        tapLevel:state.tapLevel, energyLevel:state.energyLevel,
        passiveLevel:state.passiveLevel, telegramId:cu.id })
    }).then(function(r){ return r.json(); }).then(function(d){
      if(d.balance !== undefined && d.balance !== state.balance){
        state.balance = d.balance; save(); updateUI();
      }
      if(d.passiveEarned && d.passiveEarned > 0)
        toast('⚙️ +'+fmt(d.passiveEarned)+' ZX passive', 3000);
    }).catch(function(){});
  }, immediate ? 300 : 1800);
}

// ─── Coin tap ─────────────────────────────────────────────────────────────────
var coin = document.getElementById('coin');
var lastTap = 0;
function gainTap(x, y){
  if(state.energy <= 0){ toast('⚡ Energie epuizată!'); return; }
  var now = Date.now();
  if(now - lastTap < 40) return;
  lastTap = now;
  var gain = 1 + state.tapLevel;
  state.balance += gain; state.energy = Math.max(0, state.energy-1);
  spawnFloat(gain, x, y); save(); updateUI(); syncNow(false);
}
function spawnFloat(val, x, y){
  var el = document.createElement('div');
  el.className = 'floatGain'; el.textContent = '+'+val;
  el.style.left = x+'px'; el.style.top = (y-10)+'px';
  document.body.appendChild(el);
  setTimeout(function(){ el.parentNode && el.parentNode.removeChild(el); }, 800);
}
coin.addEventListener('touchstart', function(e){
  e.preventDefault(); e.stopPropagation();
  for(var i=0; i<e.changedTouches.length; i++){
    var t=e.changedTouches[i]; gainTap(t.clientX, t.clientY);
  }
}, {passive:false});
coin.addEventListener('mousedown', function(e){ if(e.button===0) gainTap(e.clientX, e.clientY); });
coin.addEventListener('contextmenu', function(e){ e.preventDefault(); });

// Energy regen 1/3s
setInterval(function(){
  if(state.energy < state.maxEnergy){ state.energy = Math.min(state.maxEnergy, state.energy+1); save(); updateUI(); }
}, 3000);

// ─── Tabs ─────────────────────────────────────────────────────────────────────
var TABS = ['generator','tasks','referral','wallet','rank'];
document.querySelectorAll('.tabBtn').forEach(function(btn){
  btn.addEventListener('click', function(){
    var tab = btn.dataset.tab;
    document.querySelectorAll('.tabBtn').forEach(function(b){ b.classList.remove('active'); });
    btn.classList.add('active');
    TABS.forEach(function(id){ document.getElementById(id+'Tab').classList.add('hidden'); });
    document.getElementById(tab+'Tab').classList.remove('hidden');
    if(tab==='rank') loadLeaderboard();
    if(tab==='referral') loadReferral();
    syncNow(true);
  });
});

// ─── Upgrades ─────────────────────────────────────────────────────────────────
document.getElementById('buyTap').addEventListener('click', function(){
  var c=tapCost(); if(state.balance<c) return;
  state.balance-=c; state.tapLevel++; save(); updateUI(); syncNow(true);
  toast('👆 Multitap Lv.'+state.tapLevel+' — +'+(1+state.tapLevel)+' ZX/tap');
});
document.getElementById('buyEnergy').addEventListener('click', function(){
  var c=energyCost(); if(state.balance<c) return;
  state.balance-=c; state.energyLevel++; state.maxEnergy+=500; state.energy=state.maxEnergy;
  save(); updateUI(); syncNow(true);
  toast('⚡ Max energie: '+state.maxEnergy);
});
document.getElementById('buyPassive').addEventListener('click', function(){
  var c=passiveCost(); if(state.balance<c) return;
  state.balance-=c; state.passiveLevel++; save(); updateUI(); syncNow(true);
  toast('⚙️ Passive Mining Lv.'+state.passiveLevel+' → '+fmt(pph(state.passiveLevel))+' ZX/oră');
});

// ─── Recharge modal ───────────────────────────────────────────────────────────
var adCount=0;
document.getElementById('rechargeBtn').addEventListener('click', function(){
  adCount=0;
  document.getElementById('adCounter').textContent='0/3';
  document.getElementById('adProgress').style.width='0%';
  document.getElementById('rechargeModal').style.display='flex';
});
document.getElementById('watchRechargeAd').addEventListener('click', function(){
  // Use Adsgram for recharge ads too if available
  if(window._adsgramController){
    document.getElementById('rechargeModal').style.display='none';
    window._adsgramController.show().then(function(){
      adCount++;
      if(adCount>=3){ state.energy=state.maxEnergy; save(); updateUI(); toast('⚡ Energie restaurată!'); syncNow(true); }
      else { document.getElementById('rechargeModal').style.display='flex'; document.getElementById('adCounter').textContent=adCount+'/3'; document.getElementById('adProgress').style.width=(adCount/3*100)+'%'; }
    }).catch(function(){ document.getElementById('rechargeModal').style.display='flex'; });
  } else {
    adCount++;
    document.getElementById('adCounter').textContent=adCount+'/3';
    document.getElementById('adProgress').style.width=(adCount/3*100)+'%';
    if(adCount>=3){ state.energy=state.maxEnergy; save(); updateUI(); document.getElementById('rechargeModal').style.display='none'; toast('⚡ Energie restaurată!'); syncNow(true); }
  }
});
document.getElementById('closeRecharge').addEventListener('click', function(){ document.getElementById('rechargeModal').style.display='none'; });

// ─── ADSGRAM ──────────────────────────────────────────────────────────────────
function initAdsgram(){
  if(!cfg.adsgramBlockId || cfg.adsgramBlockId==='your-adsgram-block-id') return;
  if(typeof window.Adsgram === 'undefined'){
    console.warn('Adsgram library not loaded');
    return;
  }
  window._adsgramController = window.Adsgram.init({ blockId: cfg.adsgramBlockId });
}

document.getElementById('watchAdBtn').addEventListener('click', function(){
  if(cu.username==='guest'){ toast('⚠️ Autentifică-te prin Telegram!'); return; }

  // If Adsgram not configured/available, fallback
  if(!window._adsgramController){
    // Fallback: direct reward without ad (dev mode)
    state.balance += (cfg.rewardAd || 1000);
    save(); updateUI(); syncNow(true);
    toast('+'+fmt(cfg.rewardAd||1000)+' ZX (mod dev)');
    return;
  }

  var loading = document.getElementById('adLoading');
  loading.classList.add('show');

  window._adsgramController.show().then(function(){
    loading.classList.remove('show');
    // Notify server — server validates and adds reward
    fetch('/api/ad-reward', {
      method:'POST', headers:{'Content-Type':'application/json'},
      body:JSON.stringify({ username:cu.username, telegramId:cu.id, blockId:cfg.adsgramBlockId })
    }).then(function(r){ return r.json(); }).then(function(d){
      if(d.success){
        state.balance = d.balance; save(); updateUI();
        toast(d.message || '+'+fmt(d.reward)+' ZX!');
      }
    }).catch(function(){
      // Fallback if server unreachable
      state.balance += (cfg.rewardAd||1000); save(); updateUI();
      toast('+'+fmt(cfg.rewardAd||1000)+' ZX!');
    });
  }).catch(function(err){
    loading.classList.remove('show');
    if(err && err.error){ toast('ℹ️ Reclame indisponibile momentan.'); }
  });
});

// ─── TASKS (channel verification) ────────────────────────────────────────────
function claimTask(taskId, btn){
  if(cu.username==='guest'){ toast('⚠️ Autentifică-te prin Telegram!'); return; }
  if(state.claimedTasks[taskId]){ toast('Deja revendicat.'); return; }

  btn.disabled = true; btn.textContent = '⏳...';

  fetch('/api/task/claim', {
    method:'POST', headers:{'Content-Type':'application/json'},
    body:JSON.stringify({ username:cu.username, telegramId:cu.id, taskId:taskId })
  }).then(function(r){ return r.json(); }).then(function(d){
    if(d.success){
      state.claimedTasks[taskId] = true;
      state.balance = d.balance; save(); updateUI();
      btn.textContent = '✅ Revendicat'; btn.disabled = true;
      toast(d.message || '+'+fmt(d.reward)+' ZX!');
    } else {
      btn.textContent = 'Verifică'; btn.disabled = false;
      toast(d.message || 'Eroare.');
    }
  }).catch(function(){
    btn.textContent = 'Verifică'; btn.disabled = false;
    toast('❌ Eroare conexiune.');
  });
}

// ─── Daily check-in ───────────────────────────────────────────────────────────
document.getElementById('checkinBtn').addEventListener('click', function(){
  if(cu.username==='guest'){ toast('⚠️ Autentifică-te prin Telegram!'); return; }
  fetch('/api/checkin',{
    method:'POST', headers:{'Content-Type':'application/json'},
    body:JSON.stringify({ username:cu.username, telegramId:cu.id })
  }).then(function(r){ return r.json(); }).then(function(d){
    if(d.success){
      state.balance += d.reward; state.checkinStreak=d.streak;
      state.lastCheckin = new Date().toISOString().split('T')[0];
      save(); updateUI(); toast('🎁 '+d.message);
    } else { toast(d.message||'Deja ai check-in azi!'); }
  }).catch(function(){ toast('❌ Eroare.'); });
});

// ─── Referral ─────────────────────────────────────────────────────────────────
function loadReferral(){
  if(cu.username==='guest'){ document.getElementById('myRefCode').textContent='GUEST'; return; }
  fetch('/api/referral?username='+encodeURIComponent(cu.username)+'&telegramId='+cu.id)
  .then(function(r){ return r.json(); }).then(function(d){
    state.referralCode = d.code||'';
    document.getElementById('myRefCode').textContent = d.code||'—';
    document.getElementById('refCount').textContent = (d.referrals||[]).length;
    document.getElementById('refEarnings').textContent = fmt(d.earnings||0)+' ZX';
    save();
  }).catch(function(){});
}

document.getElementById('copyRefLink').addEventListener('click', function(){
  if(!state.referralCode){ loadReferral(); toast('Se generează...'); return; }
  var botName = 'ZXNetworkBot';
  var link = 'https://t.me/'+botName+'?start=ref_'+state.referralCode;
  if(navigator.clipboard){ navigator.clipboard.writeText(link).then(function(){ toast('📋 Link copiat!'); }); }
  else { toast('Cod: '+state.referralCode); }
});

document.getElementById('claimRef').addEventListener('click', function(){
  if(cu.username==='guest'){ toast('⚠️ Autentifică-te prin Telegram!'); return; }
  if(state.referredByDone){ toast('Ai folosit deja un cod.'); return; }
  var code = document.getElementById('refInput').value.trim().toUpperCase();
  if(code.length < 8){ toast('Cod invalid.'); return; }
  fetch('/api/referral/claim',{
    method:'POST', headers:{'Content-Type':'application/json'},
    body:JSON.stringify({ username:cu.username, referralCode:code, telegramId:cu.id })
  }).then(function(r){ return r.json(); }).then(function(d){
    var el = document.getElementById('refStatus');
    el.textContent = d.message||''; el.style.color = d.success?'#00ff87':'#ff6b6b';
    if(d.success){ state.balance+=(d.reward||1000); state.referredByDone=true; save(); updateUI(); syncNow(true); }
  }).catch(function(){ toast('❌ Eroare.'); });
});

// ─── TON CONNECT ──────────────────────────────────────────────────────────────
function initTonConnect(){
  if(typeof TonConnectUI === 'undefined') return;
  var manifestUrl = (cfg.appUrl||'')+'/tonconnect-manifest.json';

  var tonConnect;
  try{
    tonConnect = new TonConnectUI({
      manifestUrl: manifestUrl,
      buttonRootId: 'ton-connect-btn'
    });
  } catch(e){ return; }

  // Subscribe to wallet changes
  tonConnect.onStatusChange(function(wallet){
    var dot = document.getElementById('walletDot');
    var statusTxt = document.getElementById('walletStatusTxt');
    var addrDisp = document.getElementById('tonAddrDisplay');
    var balRow = document.getElementById('tonBalanceRow');

    if(wallet){
      var addr = wallet.account.address;
      // Convert to friendly format if needed (first 8...last 8)
      var friendly = addr.length > 20 ? addr.slice(0,8)+'...'+addr.slice(-8) : addr;

      dot.classList.add('connected');
      statusTxt.textContent = 'Wallet conectat ✅';
      addrDisp.textContent = addr;
      addrDisp.classList.remove('hidden');
      balRow.classList.remove('hidden');

      // Save address to server
      state.walletAddress = addr;
      save();

      if(cu.username !== 'guest'){
        fetch('/api/wallet/save',{
          method:'POST', headers:{'Content-Type':'application/json'},
          body:JSON.stringify({ username:cu.username, telegramId:cu.id, address:addr })
        }).then(function(r){ return r.json(); }).then(function(d){
          if(d.success) toast('🔗 Adresă TON salvată: '+friendly);
        }).catch(function(){});
      }
    } else {
      dot.classList.remove('connected');
      statusTxt.textContent = 'Wallet neconectat';
      addrDisp.classList.add('hidden');
      balRow.classList.add('hidden');
      state.walletAddress = ''; save();
    }
  });

  // Restore if already had wallet
  if(state.walletAddress){
    var addrDisp = document.getElementById('tonAddrDisplay');
    addrDisp.textContent = state.walletAddress;
    addrDisp.classList.remove('hidden');
    document.getElementById('walletDot').classList.add('connected');
    document.getElementById('walletStatusTxt').textContent = 'Wallet conectat ✅';
    document.getElementById('tonBalanceRow').classList.remove('hidden');
  }
}

// ─── Leaderboard ──────────────────────────────────────────────────────────────
function loadLeaderboard(){
  var board = document.getElementById('leaderboard');
  board.innerHTML = '<div style="color:var(--muted);text-align:center;padding:16px">Se încarcă...</div>';
  fetch('/api/leaderboard').then(function(r){ return r.json(); }).then(function(entries){
    board.innerHTML = '';
    if(!entries||!entries.length){ board.innerHTML='<div style="color:var(--muted);text-align:center;padding:20px">Niciun jucător înregistrat.</div>'; return; }
    var medals=['🥇','🥈','🥉'];
    var myRank='-';
    entries.forEach(function(e,i){
      var isMe = e.username===cu.username;
      if(isMe) myRank='#'+(i+1);
      var d = document.createElement('div');
      d.className='leaderboardItem';
      d.innerHTML='<span class="leaderboardRank">'+(medals[i]||'#'+(i+1))+'</span>'+
        '<span class="leaderboardName" style="'+(isMe?'color:#00ff87;font-weight:900;':'')+'">'+(isMe?e.username+' ◀':e.username)+'</span>'+
        '<span class="leaderboardBalance">'+fmt(e.balance)+' ZX</span>';
      board.appendChild(d);
    });
    document.getElementById('myRank').textContent = myRank;
    document.getElementById('myBalanceRank').textContent = fmt(state.balance)+' ZX';
  }).catch(function(){
    board.innerHTML='<div style="color:#ff6b6b;text-align:center">Eroare.</div>';
  });
}

// ─── Wallet delete ────────────────────────────────────────────────────────────
document.getElementById('deleteAccountBtn').addEventListener('click', function(){
  if(!confirm('Ștergi datele locale? Progresul pe server rămâne.')) return;
  localStorage.removeItem(SK); location.reload();
});

// ─── Init ─────────────────────────────────────────────────────────────────────
updateUI();
if(cu.username!=='guest'){
  setTimeout(function(){ syncNow(true); }, 600);
  fetch('/api/passive?username='+encodeURIComponent(cu.username))
  .then(function(r){ return r.json(); })
  .then(function(d){ if(d.earned&&d.earned>0) toast('⚙️ Offline mining: +'+fmt(d.earned)+' ZX!', 4000); })
  .catch(function(){});
}

})();
</script>
</body>
</html>`
