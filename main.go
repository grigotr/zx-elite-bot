```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const webAppHTML = `
<!DOCTYPE html>
<html lang="ro">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0,maximum-scale=1.0,user-scalable=no">

<title>ZX Network</title>

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
 --border:rgba(255,255,255,.08);
}

*{
 margin:0;
 padding:0;
 box-sizing:border-box;
}

body{
 background:
 radial-gradient(circle at top left,
 rgba(0,245,212,.12),
 transparent 40%),
 radial-gradient(circle at top right,
 rgba(0,255,135,.08),
 transparent 40%),
 #070b10;

 color:var(--text);
 font-family:
 Inter,
 Segoe UI,
 sans-serif;

 min-height:100vh;
 overflow-x:hidden;
}

body::before{
 content:"";
 position:fixed;
 inset:0;

 background-image:
 linear-gradient(
 rgba(255,255,255,.03) 1px,
 transparent 1px),
 linear-gradient(
 90deg,
 rgba(255,255,255,.03) 1px,
 transparent 1px);

 background-size:40px 40px;

 pointer-events:none;
 z-index:0;
}

.app{
 position:relative;
 z-index:1;
 max-width:600px;
 margin:auto;
 padding-bottom:120px;
}

.header{
 position:sticky;
 top:0;

 display:flex;
 justify-content:space-between;
 align-items:center;

 padding:16px;

 backdrop-filter:blur(15px);

 background:
 rgba(7,11,16,.85);

 border-bottom:
 1px solid var(--border);

 z-index:100;
}

.userBox{
 display:flex;
 flex-direction:column;
 gap:4px;
}

.userLabel{
 font-size:12px;
 color:var(--muted);
}

.userName{
 font-weight:800;
 font-size:15px;
}

.netBadge{
 background:
 rgba(0,245,212,.15);

 color:var(--green);

 border:
 1px solid rgba(0,245,212,.3);

 border-radius:999px;

 padding:
 10px 14px;

 font-size:12px;
 font-weight:800;

 box-shadow:
 0 0 18px rgba(0,245,212,.25);
}

.balanceWrap{
 padding:18px;
}

.balanceCard{
 background:
 linear-gradient(
 180deg,
 rgba(17,27,43,.95),
 rgba(13,20,32,.95)
 );

 border:
 1px solid rgba(0,245,212,.18);

 border-radius:26px;

 padding:22px;

 box-shadow:
 0 0 35px rgba(0,245,212,.15),
 0 0 60px rgba(0,245,212,.08);
}

.balanceTitle{
 color:var(--muted);
 font-size:12px;
 letter-spacing:1px;
 text-transform:uppercase;
}

.balanceValue{
 margin-top:8px;

 font-size:42px;
 font-weight:900;

 color:white;

 text-shadow:
 0 0 15px rgba(0,245,212,.6),
 0 0 35px rgba(0,245,212,.35);
}

.page{
 padding:18px;
}

.section{
 background:
 linear-gradient(
 180deg,
 rgba(17,27,43,.96),
 rgba(13,20,32,.96)
 );

 border-radius:24px;

 border:
 1px solid var(--border);

 padding:20px;
}

.coinArea{
 display:flex;
 flex-direction:column;
 align-items:center;
 justify-content:center;
}

.coin{
 width:240px;
 height:240px;

 cursor:pointer;

 transition:
 transform .12s ease;
}

.coin:active{
 transform:scale(.94);
}

.tapHint{
 margin-top:10px;

 color:var(--muted);

 font-size:12px;
 text-transform:uppercase;
 letter-spacing:1px;
}

.energyRow{
 margin-top:24px;

 display:flex;
 gap:10px;
 align-items:center;
}

.energyBox{
 flex:1;
}

.energyText{
 display:flex;
 justify-content:space-between;

 margin-bottom:8px;

 font-size:13px;
}

.energyBar{
 width:100%;
 height:14px;

 border-radius:999px;

 overflow:hidden;

 background:#08111b;

 border:
 1px solid rgba(255,255,255,.05);
}

.energyFill{
 width:100%;
 height:100%;

 background:
 linear-gradient(
 90deg,
 var(--green),
 var(--green2));

 box-shadow:
 0 0 20px rgba(0,245,212,.35);
}

.btn{
 border:none;

 cursor:pointer;

 padding:
 12px 18px;

 border-radius:14px;

 font-weight:800;

 background:
 linear-gradient(
 135deg,
 var(--green),
 var(--green2));

 color:#04120d;

 box-shadow:
 0 0 25px rgba(0,245,212,.25);
}

.btn:hover{
 filter:brightness(1.05);
}

.upgrades{
 margin-top:22px;

 display:grid;
 grid-template-columns:
 repeat(2,1fr);

 gap:12px;
}

.upgradeCard{
 background:
 rgba(0,0,0,.18);

 border:
 1px solid rgba(255,255,255,.05);

 border-radius:20px;

 padding:16px;
}

.upgradeTitle{
 font-size:14px;
 font-weight:800;
}

.upgradeDesc{
 margin-top:6px;

 color:var(--muted);

 font-size:12px;
 line-height:1.4;
}

.upgradeMeta{
 margin-top:10px;

 font-size:13px;
 color:#d5fff6;
}

.bottomNav{
 position:fixed;

 left:0;
 right:0;
 bottom:0;

 padding:12px;

 z-index:999;
}

.bottomInner{
 max-width:600px;

 margin:auto;

 display:grid;

 grid-template-columns:
 repeat(4,1fr);

 gap:10px;

 background:
 rgba(10,16,24,.96);

 border:
 1px solid rgba(255,255,255,.05);

 border-radius:22px;

 padding:10px;

 backdrop-filter:blur(20px);
}

.tabBtn{
 background:none;
 border:none;

 color:#87a39d;

 padding:10px;

 border-radius:16px;

 cursor:pointer;

 font-weight:800;

 transition:.2s;
}

.tabBtn.active{
 color:#04120d;

 background:
 linear-gradient(
 135deg,
 var(--green),
 var(--green2));
}

.hidden{
 display:none;
}

.floatGain{
 position:fixed;

 color:#8effdf;

 font-weight:900;

 pointer-events:none;

 animation:
 floatUp .9s ease-out forwards;
}

@keyframes floatUp{

 from{
  opacity:1;
  transform:
   translateY(0px);
 }

 to{
  opacity:0;
  transform:
   translateY(-80px);
 }
}

</style>
</head>

<body>

<div class="app">

<header class="header">

 <div class="userBox">

   <div class="userLabel">
      User
   </div>

   <div
    id="telegramUser"
    class="userName">

    Guest

   </div>

 </div>

 <div class="netBadge">
   ⚡ ZX-NET LIVE
 </div>

</header>

<div class="balanceWrap">

 <div class="balanceCard">

   <div class="balanceTitle">
     Total ZX Tokens
   </div>

   <div
    id="balanceDisplay"
    class="balanceValue">

    0

   </div>

 </div>

</div>

<div class="page">

<div
 id="generatorTab"
 class="section">

 <div class="coinArea">

  <svg
   id="coin"
   class="coin"
   viewBox="0 0 500 500">

   <defs>

    <radialGradient
     id="coinGrad">

      <stop
       offset="0%"
       stop-color="#7affd8"/>

      <stop
       offset="60%"
       stop-color="#00ff87"/>

      <stop
       offset="100%"
       stop-color="#00c96c"/>

    </radialGradient>

   </defs>

   <circle
    cx="250"
    cy="250"
    r="220"
    fill="url(#coinGrad)"
    stroke="#d8fff4"
    stroke-width="8"/>

   <circle
    cx="250"
    cy="250"
    r="190"
    fill="none"
    stroke="rgba(255,255,255,.3)"
    stroke-width="4"/>

   <text
    x="250"
    y="285"
    text-anchor="middle"
    font-size="130"
    font-weight="900"
    fill="#ffffff"
    stroke="#004f3a"
    stroke-width="8">

    ZX

   </text>

  </svg>

  <div class="tapHint">
   Tap pentru ZX Tokens
  </div>

 </div>

 <div class="energyRow">

  <div class="energyBox">

   <div class="energyText">

    <span>
      Energie
    </span>

    <span id="energyText">
      500 / 500
    </span>

   </div>

   <div class="energyBar">

    <div
     id="energyFill"
     class="energyFill">
    </div>

   </div>

  </div>

  <button
   id="rechargeBtn"
   class="btn">

   Încarcă

  </button>

 </div>

 <div class="upgrades">

<div class="upgradeCard">

 <div class="upgradeTitle">
  👆 Multitap Booster
 </div>

 <div class="upgradeDesc">
  Crește câștigul pe fiecare tap cu +1 ZX pentru fiecare nivel cumpărat.
 </div>

 <div
  id="tapLevel"
  class="upgradeMeta">

  Nivel 0

 </div>

 <div
  id="tapCost"
  class="upgradeMeta">

  Cost: 1000 ZX

 </div>

 <button
  id="buyTapUpgrade"
  class="btn"
  style="width:100%;margin-top:12px;">

  Upgrade

 </button>

</div>

<div class="upgradeCard">

 <div class="upgradeTitle">
  ⚡ Energy Capacity Booster
 </div>

 <div class="upgradeDesc">
  Crește permanent energia maximă la 1000, 1500, 2000 și mai mult.
 </div>

 <div
  id="energyLevel"
  class="upgradeMeta">

  Max 500

 </div>

 <div
  id="energyCost"
  class="upgradeMeta">

  Cost: 2500 ZX

 </div>

 <button
  id="buyEnergyUpgrade"
  class="btn"
  style="width:100%;margin-top:12px;">

  Upgrade

 </button>

</div>

</div>

</div>

<!-- TASKS TAB -->

<div
 id="tasksTab"
 class="section hidden">

 <h2
  style="
  margin-bottom:20px;
  color:#d9fff5;">

  💼 Tasks

 </h2>

 <div
  class="upgradeCard"
  style="margin-bottom:14px;">

  <div class="upgradeTitle">
   📺 Watch Ad
  </div>

  <div class="upgradeDesc">
   Vizionează o reclamă simulată și primește instant 1000 ZX.
  </div>

  <button
   id="watchAdBtn"
   class="btn"
   style="width:100%;margin-top:12px;">

   +1000 ZX

  </button>

 </div>

 <div
  class="upgradeCard"
  style="margin-bottom:14px;">

  <div class="upgradeTitle">
   📢 Abonare Canal Telegram
  </div>

  <div class="upgradeDesc">
   Reward instant pentru conectare.
  </div>

  <button
   id="taskTelegram"
   class="btn"
   style="width:100%;margin-top:12px;">

   +500 ZX

  </button>

 </div>

 <div
  class="upgradeCard"
  style="margin-bottom:14px;">

  <div class="upgradeTitle">
   🤖 Start Partener Bot
  </div>

  <div class="upgradeDesc">
   Activează partenerul și revendică recompensa.
  </div>

  <button
   id="taskPartner"
   class="btn"
   style="width:100%;margin-top:12px;">

   +2000 ZX

  </button>

 </div>

</div>

<!-- WALLET TAB -->

<div
 id="walletTab"
 class="section hidden">

 <h2
  style="
  margin-bottom:20px;
  color:#d9fff5;">

  👛 Wallet

 </h2>

 <div
  style="
  background:rgba(0,0,0,.18);
  border-radius:20px;
  padding:18px;
  border:1px solid rgba(255,255,255,.05);
  ">

  <div
   style="
   font-size:13px;
   color:#87a39d;">

   ZX Balance Mirror

  </div>

  <div
   id="walletBalance"
   style="
   margin-top:10px;
   font-size:34px;
   font-weight:900;">

   0

  </div>

  <button
   id="connectWallet"
   class="btn"
   style="
   margin-top:18px;
   width:100%;">

   Connect TON Wallet

  </button>

  <div
   id="walletAddress"
   style="
   display:none;
   margin-top:14px;
   color:#9fffe9;
   word-break:break-all;
   font-size:13px;">

  </div>

 </div>

 <h3
  style="
  margin-top:24px;
  margin-bottom:14px;">

  Withdrawal History

 </h3>
<table
 style="
 width:100%;
 border-collapse:collapse;">

 <thead>

  <tr>

   <th
    style="
    text-align:left;
    padding:12px;
    border-bottom:1px solid rgba(255,255,255,.08);">

    Date

   </th>

   <th
    style="
    text-align:left;
    padding:12px;
    border-bottom:1px solid rgba(255,255,255,.08);">

    Amount

   </th>

   <th
    style="
    text-align:left;
    padding:12px;
    border-bottom:1px solid rgba(255,255,255,.08);">

    Asset

   </th>

   <th
    style="
    text-align:left;
    padding:12px;
    border-bottom:1px solid rgba(255,255,255,.08);">

    Status

   </th>

  </tr>

 </thead>

 <tbody id="withdrawTable">

  <tr>

   <td style="padding:12px;">
    2026-06-01
   </td>

   <td style="padding:12px;">
    120000
   </td>

   <td style="padding:12px;">
    TON
   </td>

   <td style="padding:12px;color:#ffd166;">
    În așteptare
   </td>

  </tr>

  <tr>

   <td style="padding:12px;">
    2026-05-24
   </td>

   <td style="padding:12px;">
    50000
   </td>

   <td style="padding:12px;">
    TON
   </td>

   <td style="padding:12px;color:#00ff87;">
    Finalizat
   </td>

  </tr>

 </tbody>

</table>

</div>

<!-- RANK TAB -->

<div
 id="rankTab"
 class="section hidden">

 <h2
  style="
  margin-bottom:20px;
  color:#d9fff5;">

  🏆 Rank

 </h2>

 <div id="leaderboard">

 </div>

 <div
  style="
  margin-top:25px;
  height:2px;

  background:
  linear-gradient(
  90deg,
  transparent,
  #9d4dff,
  transparent);

  box-shadow:
  0 0 20px #9d4dff;">

 </div>

 <div
  style="
  margin-top:20px;

  background:
  rgba(157,77,255,.08);

  border:
  1px solid rgba(157,77,255,.3);

  border-radius:20px;

  padding:18px;">

  <div
   style="
   font-size:13px;
   color:#b79cff;">

   Poziția ta

  </div>

  <div
   id="myRank"
   style="
   margin-top:8px;
   font-size:30px;
   font-weight:900;">

   #1245

  </div>

  <div
   id="myBalance"
   style="
   margin-top:8px;
   color:#e9dbff;">

   0 ZX

  </div>

 </div>

</div>

</div>

<!-- RECHARGE MODAL -->

<div
 id="rechargeModal"
 style="
 display:none;

 position:fixed;
 inset:0;

 background:
 rgba(0,0,0,.8);

 backdrop-filter:
 blur(10px);

 z-index:5000;

 justify-content:center;
 align-items:center;">

 <div
  style="
  width:90%;
  max-width:420px;

  background:
  #101827;

  border-radius:24px;

  border:
  1px solid rgba(0,245,212,.2);

  padding:22px;">

  <h2>
   📺 Vizionează 3 Reclame
  </h2>

  <p
   style="
   margin-top:10px;
   color:#87a39d;">

   După finalizare energia va fi restaurată complet.

  </p>

  <div
   id="adCounter"
   style="
   margin-top:16px;
   font-size:24px;
   font-weight:900;">

   0 / 3

  </div>

  <button
   id="watchRechargeAd"
   class="btn"
   style="
   width:100%;
   margin-top:16px;">

   Watch Ad

  </button>

  <button
   id="closeRecharge"
   class="btn"
   style="
   width:100%;
   margin-top:12px;
   background:#202b3d;
   color:white;">

   Close

  </button>

 </div>

</div>

<div class="bottomNav">

 <div class="bottomInner">

  <button
   class="tabBtn active"
   data-tab="generator">

   🎮
   <br>
   Generator

  </button>

  <button
   class="tabBtn"
   data-tab="tasks">

   💼
   <br>
   Tasks

  </button>

  <button
   class="tabBtn"
   data-tab="wallet">

   👛
   <br>
   Wallet

  </button>

  <button
   class="tabBtn"
   data-tab="rank">

   🏆
   <br>
   Rank

  </button>

 </div>

</div>

<script>

const STORAGE_KEY =
"zx-network-state";

let state =
JSON.parse(
 localStorage.getItem(
 STORAGE_KEY
 ) || "{}"
);

if(!state.balance){

 state = {

  balance:0,

  energy:500,

  maxEnergy:500,

  tapLevel:0,

  energyLevel:0,

  rechargeAds:0,

  walletConnected:false,

  walletAddress:"",

  claimedTasks:{}

 };

}

function saveState(){

 localStorage.setItem(
  STORAGE_KEY,
  JSON.stringify(state)
 );

}

function formatNumber(v){

 return Number(v)
 .toLocaleString("ro-RO");

}

function updateUI(){

 document
 .getElementById(
 "balanceDisplay"
 )
 .innerText =
 formatNumber(
 state.balance
 );

 document
 .getElementById(
 "walletBalance"
 )
 .innerText =
 formatNumber(
 state.balance
 );

 document
 .getElementById(
 "myBalance"
 )
 .innerText =
 formatNumber(
 state.balance
 ) + " ZX";

 document
 .getElementById(
 "energyText"
 )
 .innerText =
 state.energy +
 " / " +
 state.maxEnergy;

 const percent =
 (state.energy /
 state.maxEnergy)
 * 100;

 document
 .getElementById(
 "energyFill"
 )
 .style.width =
 percent + "%";
function gainTap(){

 if(state.energy <= 0) return;

 const gain =
  1 + state.tapLevel;

 state.balance += gain;

 state.energy -= 1;

 spawnFloat(gain);

 saveState();

 updateUI();

}

function spawnFloat(value){

 const el =
 document.createElement("div");

 el.className =
 "floatGain";

 el.innerText =
 "+" + value;

 el.style.left =
 (window.innerWidth/2) + "px";

 el.style.top =
 (window.innerHeight/2) + "px";

 document.body.appendChild(el);

 setTimeout(() => {

  el.remove();

 }, 900);

}

document
.getElementById("coin")
.addEventListener("click",
 gainTap
);

// TABS

document
.querySelectorAll(".tabBtn")
.forEach(btn => {

 btn.addEventListener("click", () => {

  document
  .querySelectorAll(".tabBtn")
  .forEach(b =>
   b.classList.remove("active")
  );

  btn.classList.add("active");

  const tab =
  btn.dataset.tab;

  document
  .getElementById("generatorTab")
  .classList.add("hidden");

  document
  .getElementById("tasksTab")
  .classList.add("hidden");

  document
  .getElementById("walletTab")
  .classList.add("hidden");

  document
  .getElementById("rankTab")
  .classList.add("hidden");

  document
  .getElementById(tab + "Tab")
  .classList.remove("hidden");

 });

});

// TASKS

function claimTask(id, reward, btn){

 if(state.claimedTasks[id]) return;

 state.claimedTasks[id] = true;

 state.balance += reward;

 btn.innerText = "Claimed";

 btn.disabled = true;

 saveState();

 updateUI();

}

document
.getElementById("taskTelegram")
.addEventListener("click",
 function(){
  claimTask("tg",500,this);
 }
);

document
.getElementById("taskPartner")
.addEventListener("click",
 function(){
  claimTask("partner",2000,this);
 }
);

// AD TASK

document
.getElementById("watchAdBtn")
.addEventListener("click", () => {

 state.balance += 1000;

 saveState();

 updateUI();

});

// WALLET

document
.getElementById("connectWallet")
.addEventListener("click", () => {

 state.walletConnected = true;

 state.walletAddress =
 "EQB-" +
 Math.random()
 .toString(36)
 .substring(2,12);

 document
 .getElementById("walletAddress")
 .style.display =
 "block";

 document
 .getElementById("walletAddress")
 .innerText =
 state.walletAddress;

 saveState();

 updateUI();

});

// RECHARGE ADS

let adCount = 0;

document
.getElementById("rechargeBtn")
.addEventListener("click", () => {

 document
 .getElementById("rechargeModal")
 .style.display = "flex";

 adCount = 0;

 document
 .getElementById("adCounter")
 .innerText = "0 / 3";

});

document
.getElementById("watchRechargeAd")
.addEventListener("click", () => {

 adCount++;

 document
 .getElementById("adCounter")
 .innerText =
 adCount + " / 3";

 if(adCount >= 3){

  state.energy =
  state.maxEnergy;

  saveState();

  updateUI();

 }

});

document
.getElementById("closeRecharge")
.addEventListener("click", () => {

 document
 .getElementById("rechargeModal")
 .style.display = "none";

});

// INIT

updateUI();

</script>

</div>

</body>
</html>


func main() {

 token :=
 os.Getenv("TELEGRAM_BOT_TOKEN")

 if token == "" {

  log.Println(
   "Missing TELEGRAM_BOT_TOKEN"
  )

 }

 bot, err :=
 tgbotapi.NewBotAPI(token)

 if err != nil {
  log.Fatal(err)
 }

 go func() {

  u := tgbotapi.NewUpdate(0)

  u.Timeout = 60

  updates := bot.GetUpdatesChan(u)

  for update := range updates {

   if update.Message == nil {
    continue
   }

   if update.Message.Text == "/start" {

    msg :=
    tgbotapi.NewMessage(
     update.Message.Chat.ID,
     "ZX WebApp is live"
    )

    bot.Send(msg)

   }

  }

 }()

 http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

  w.Header().Set(
   "Content-Type",
   "text/html"
  )

  w.Write([]byte(webAppHTML))

 })

 port := os.Getenv("PORT")

 if port == "" {
  port = "8080"
 }

 log.Println("Running on :" + port)

 log.Fatal(
  http.ListenAndServe(
   ":" + port,
   nil,
  ),
 )

}
