package main

import (
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

/* BALANȚA CENTRATĂ DEASUPRA MONEDEI */
.balance-container {
  text-align: center;
  margin-bottom: 10px;
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

.btn:disabled{
 opacity:0.6;
 cursor:not-allowed;
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

 color:#ffffff;

 font-weight:900;
 font-size:24px;

 text-shadow: 0 0 15px rgba(0,255,135,0.9);

 pointer-events:none;

 animation:
 floatUp 1s ease-out forwards;
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
   translateY(-100px);
 }
}

/* Leaderboard styles (added) */
.leaderboardItem {
 display:flex;
 justify-content:space-between;
 padding:10px 0;
 border-bottom:1px solid rgba(255,255,255,0.06);
}
.leaderboardRank { width:30px; color:var(--green); font-weight:bold; }
.leaderboardName { flex:1; }
.leaderboardBalance { color:#fff; }
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

<div class="page">

<div
 id="generatorTab"
 class="section">

 <!-- Balanța centrată -->
 <div class="balance-container">
   <div class="balanceTitle">
     Total ZX Tokens
   </div>
   <div
    id="balanceDisplay"
    class="balanceValue">
    0
   </div>
 </div>

 <div class="coinArea">

  <!-- Moneda ZX cu literele ca traiectorii vectoriale -->
  <svg
   id="coin"
   class="coin"
   viewBox="0 0 500 500">

   <defs>
    <radialGradient id="coinGrad">
      <stop offset="0%" stop-color="#7affd8"/>
      <stop offset="60%" stop-color="#00ff87"/>
      <stop offset="100%" stop-color="#00c96c"/>
    </radialGradient>
    <filter id="glow" x="-20%" y="-20%" width="140%" height="140%">
      <feGaussianBlur stdDeviation="4" result="blur"/>
      <feMerge>
        <feMergeNode in="blur"/>
        <feMergeNode in="SourceGraphic"/>
      </feMerge>
    </filter>
   </defs>

   <!-- Inel exterior argintiu -->
   <circle
    cx="250" cy="250" r="220"
    fill="none" stroke="#b0c4de" stroke-width="10"/>

   <!-- Inel interior auriu -->
   <circle
    cx="250" cy="250" r="200"
    fill="none" stroke="url(#coinGrad)" stroke-width="6"/>

   <!-- Miez întunecat -->
   <circle
    cx="250" cy="250" r="180"
    fill="#05101c" stroke="#00ff87" stroke-width="2"/>

   <!-- Microcip central (spate) -->
   <rect x="140" y="150" width="220" height="200" rx="10" fill="none" stroke="rgba(0,255,135,0.2)" stroke-width="1.5"/>
   <path d="M160 160 L160 180 M180 160 L180 180 M200 160 L200 180 M220 160 L220 180 M240 160 L240 180 M260 160 L260 180 M280 160 L280 180 M300 160 L300 180 M320 160 L320 180" stroke="#00ff87" stroke-width="1" opacity="0.4"/>
   <path d="M160 340 L160 320 M180 340 L180 320 M200 340 L200 320 M220 340 L220 320 M240 340 L240 320 M260 340 L260 320 M280 340 L280 320 M300 340 L300 320 M320 340 L320 320" stroke="#00ff87" stroke-width="1" opacity="0.4"/>
   <rect x="230" y="230" width="40" height="40" fill="none" stroke="#00ff87" stroke-width="1.5"/>
   <circle cx="250" cy="250" r="12" fill="#00ff87" opacity="0.3"/>

   <!-- Literele Z și X ca traiectorii vectoriale (nu text) -->
   <g filter="url(#glow)">
     <!-- Z -->
     <path d="M 170 180 L 330 180 L 170 320 L 330 320"
           fill="none" stroke="#ffffff" stroke-width="22"
           stroke-linecap="round" stroke-linejoin="round"/>
     <!-- X -->
     <path d="M 170 180 L 330 320 M 330 180 L 170 320"
           fill="none" stroke="#ffffff" stroke-width="22"
           stroke-linecap="round" stroke-linejoin="round"/>
   </g>
  </svg>

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

let state = JSON.parse(
 localStorage.getItem(STORAGE_KEY) || "{}"
);

state = {
  balance: state.balance || 0,
  energy: state.energy || 500,
  maxEnergy: state.maxEnergy || 500,
  tapLevel: state.tapLevel || 0,
  energyLevel: state.energyLevel || 0,
  rechargeAds: state.rechargeAds || 0,
  walletConnected: state.walletConnected || false,
  walletAddress: state.walletAddress || "",
  claimedTasks: state.claimedTasks || {}
};

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

// ─── UPGRADE COST FUNCTIONS ───
function getTapUpgradeCost() {
  return 1000 * Math.pow(state.tapLevel + 1, 2);
}

function getEnergyUpgradeCost() {
  return 2500 * Math.pow(state.energyLevel + 1, 2);
}

// ─── LEADERBOARD ───
function generateFakePlayers() {
  var names = [
    "CryptoKing", "ZXWhale", "BlockchainLord", "TokenMaster", "NebulaHacker",
    "StarTrader", "QuantumMiner", "ApexGamer", "PhantomUser", "LuckyStrike"
  ];
  var players = [];
  for (var i = 0; i < names.length; i++) {
    players.push({
      name: names[i],
      balance: Math.floor(Math.random() * 500000) + 100000
    });
  }
  players.push({ name: "Tu", balance: state.balance, isUser: true });
  players.sort(function(a, b) { return b.balance - a.balance; });
  return players;
}

function updateLeaderboard() {
  var players = generateFakePlayers();
  var board = document.getElementById("leaderboard");
  if (!board) return;
  board.innerHTML = "";
  for (var i = 0; i < players.length; i++) {
    var p = players[i];
    var div = document.createElement("div");
    div.className = "leaderboardItem";
    div.innerHTML =
      '<span class="leaderboardRank">#' + (i + 1) + '</span>' +
      '<span class="leaderboardName" style="' + (p.isUser ? 'color:#00ff87; font-weight:bold' : '') + '">' + p.name + '</span>' +
      '<span class="leaderboardBalance">' + formatNumber(p.balance) + ' ZX</span>';
    board.appendChild(div);
  }
  var userRank = players.findIndex(function(p) { return p.isUser; }) + 1;
  document.getElementById("myRank").innerText = "#" + userRank;
  document.getElementById("myBalance").innerText = formatNumber(state.balance) + " ZX";
}

// ─── MAIN UI UPDATE ───
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

 // Upgrade display & button states
 const tapCost = getTapUpgradeCost();
 document.getElementById("tapLevel").innerText = "Nivel " + state.tapLevel;
 document.getElementById("tapCost").innerText = "Cost: " + formatNumber(tapCost) + " ZX";
 const tapBtn = document.getElementById("buyTapUpgrade");
 tapBtn.disabled = state.balance < tapCost;
 tapBtn.innerText = tapBtn.disabled ? "Fonduri insuficiente" : "Upgrade";

 const energyCost = getEnergyUpgradeCost();
 document.getElementById("energyLevel").innerText = "Max " + state.maxEnergy;
 document.getElementById("energyCost").innerText = "Cost: " + formatNumber(energyCost) + " ZX";
 const energyBtn = document.getElementById("buyEnergyUpgrade");
 energyBtn.disabled = state.balance < energyCost;
 energyBtn.innerText = energyBtn.disabled ? "Fonduri insuficiente" : "Upgrade";

 // Leaderboard
 updateLeaderboard();
}

// ─── TAP CU COORDONATE REALE ───
function gainTap(event) {
  if (state.energy <= 0) return;

  const gain = 1 + state.tapLevel; // valoare dinamică
  state.balance += gain;
  state.energy -= 1;

  // Coordonatele click-ului (relative la viewport)
  const x = event.clientX;
  const y = event.clientY;

  spawnFloat(gain, x, y);

  saveState();
  updateUI();
}

// ─── FLOATING TEXT LA POZIȚIA CLICK-ULUI ───
function spawnFloat(value, posX, posY) {
  const el = document.createElement("div");
  el.className = "floatGain";
  el.innerText = "+" + value;

  // Poziționare absolută la coordonatele click-ului
  el.style.left = posX + "px";
  el.style.top  = posY + "px";

  document.body.appendChild(el);

  // Elimină elementul după terminarea animației (1s)
  setTimeout(() => {
    el.remove();
  }, 1000);
}

// Evenimentul de click pe monedă
document.getElementById("coin").addEventListener("click", gainTap);

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

// ─── UPGRADE BUTTONS (added) ───
document.getElementById("buyTapUpgrade").addEventListener("click", () => {
  const cost = getTapUpgradeCost();
  if (state.balance < cost) return;
  state.balance -= cost;
  state.tapLevel += 1;
  saveState();
  updateUI();
});

document.getElementById("buyEnergyUpgrade").addEventListener("click", () => {
  const cost = getEnergyUpgradeCost();
  if (state.balance < cost) return;
  state.balance -= cost;
  state.energyLevel += 1;
  state.maxEnergy += 500;
  state.energy = state.maxEnergy;
  saveState();
  updateUI();
});

// INIT

updateUI();

</script>

</div>

</body>
</html>
`

func main() {

	// Token bot (direct în cod, așa cum ai cerut)
	token := "8744648391:AAHbsnd54wrv686PkLCbtj4ueBm4DqEB4vQ"

	bot, err := tgbotapi.NewBotAPI(token)
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
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "ZX WebApp is live")
				bot.Send(msg)
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(webAppHTML))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Running on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
