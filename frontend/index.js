const el = {
  roomId: document.getElementById("roomId"),
  createRoomBtn: document.getElementById("createRoomBtn"),
  playerId: document.getElementById("playerId"),
  connectBtn: document.getElementById("connectBtn"),
  drawBtn: document.getElementById("drawBtn"),
  playBtn: document.getElementById("playBtn"),
  status: document.getElementById("status"),
  players: document.getElementById("players"),
  topCard: document.getElementById("topCard"),
  hand: document.getElementById("hand"),
};

let ws = null;
let game = null;
let selectedCardId = null;

function setStatus(msg, type = "") {
  el.status.textContent = msg;
  el.status.className = `status ${type}`.trim();
}

function cardNode(card, selectable) {
  const div = document.createElement("div");
  div.className = `uno-card c-${card.color}`;
  if (selectable && selectedCardId === card.id) {
    div.classList.add("selected");
  }
  div.dataset.id = String(card.id);

  const idTag = document.createElement("small");
  idTag.textContent = `#${card.id}`;
  div.appendChild(idTag);

  const val = document.createElement("span");
  val.textContent = card.val;
  div.appendChild(val);
  return div;
}

function cardValid(card) {
  return !!(card && typeof card.id === "number" && card.id > 0 && card.color && card.val);
}

function myPlayer() {
  if (!game || !Array.isArray(game.players)) {
    return null;
  }
  return game.players.find((p) => p.ID === el.playerId.value) || null;
}

function renderTopCard() {
  el.topCard.innerHTML = "";
  if (!game || !cardValid(game.topCard)) {
    el.topCard.textContent = "等待发牌...";
    return;
  }
  const node = cardNode(game.topCard, false);
  node.classList.add("table-top-card");
  el.topCard.appendChild(node);
}

function renderPlayers() {
  el.players.innerHTML = "";
  if (!game || !Array.isArray(game.players)) {
    return;
  }

  game.players.forEach((p, idx) => {
    const row = document.createElement("div");
    row.className = "player";
    if (idx === game.nowID) {
      row.classList.add("now");
    }
    row.innerHTML = `<strong>${p.name} (${p.ID})</strong><span>手牌 ${p.cards.length}</span>`;
    el.players.appendChild(row);
  });
}

function renderHand() {
  el.hand.innerHTML = "";
  const me = myPlayer();
  if (!me) {
    el.hand.textContent = "未找到当前玩家，请检查 playerId。";
    return;
  }

  me.cards.forEach((c) => {
    const n = cardNode(c, true);
    n.addEventListener("click", () => {
      selectedCardId = c.id;
      renderHand();
    });
    el.hand.appendChild(n);
  });
}

function render() {
  renderPlayers();
  renderTopCard();
  renderHand();

  const me = myPlayer();
  const canAct = !!(game && me && game.players[game.nowID]?.ID === me.ID);
  el.playBtn.disabled = !canAct || selectedCardId === null;
  el.drawBtn.disabled = !canAct;
}

function sendAction(payload) {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    setStatus("连接未建立", "err");
    return;
  }
  ws.send(JSON.stringify(payload));
}

function connect() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const roomId = (el.roomId.value || "ROOM01").trim().toUpperCase();
  el.roomId.value = roomId;
  const url = `${protocol}//${window.location.host}/ws?roomId=${encodeURIComponent(roomId)}&playerId=${encodeURIComponent(el.playerId.value)}`;
  ws = new WebSocket(url);

  setStatus("连接中...");
  el.connectBtn.disabled = true;

  ws.onopen = () => {
    setStatus(`已连接房间 ${roomId} (${el.playerId.value})`, "ok");
    el.playerId.disabled = true;
    el.roomId.disabled = true;
    el.createRoomBtn.disabled = true;
    el.connectBtn.textContent = "已连接";
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (data && data.error) {
        setStatus(data.error, "err");
        return;
      }

      game = data;
      if (selectedCardId !== null) {
        const me = myPlayer();
        if (!me || !me.cards.some((c) => c.id === selectedCardId)) {
          selectedCardId = null;
        }
      }
      setStatus("状态已更新", "ok");
      render();
    } catch (err) {
      setStatus(`消息解析失败: ${String(err)}`, "err");
    }
  };

  ws.onclose = () => {
    setStatus("连接已断开", "err");
    el.connectBtn.disabled = false;
    el.connectBtn.textContent = "连接";
    el.playerId.disabled = false;
    el.roomId.disabled = false;
    el.createRoomBtn.disabled = false;
  };

  ws.onerror = () => {
    setStatus("网络错误，请检查后端是否启动", "err");
  };
}

async function createRoom() {
  el.createRoomBtn.disabled = true;
  try {
    const resp = await fetch("/api/rooms", { method: "POST" });
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}`);
    }
    const data = await resp.json();
    if (!data || !data.roomId) {
      throw new Error("返回格式错误");
    }
    el.roomId.value = String(data.roomId).toUpperCase();
    setStatus(`房间已创建: ${el.roomId.value}`, "ok");
  } catch (err) {
    setStatus(`开房失败: ${String(err)}`, "err");
  } finally {
    if (!el.roomId.disabled) {
      el.createRoomBtn.disabled = false;
    }
  }
}

el.connectBtn.addEventListener("click", connect);
el.createRoomBtn.addEventListener("click", createRoom);

el.playBtn.addEventListener("click", () => {
  if (selectedCardId === null) {
    setStatus("请先选择一张牌", "err");
    return;
  }
  sendAction({ type: "PLAY", cardId: selectedCardId });
});

el.drawBtn.addEventListener("click", () => {
  sendAction({ type: "DRAW" });
});

render();
