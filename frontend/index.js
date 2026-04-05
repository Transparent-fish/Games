// ─── 状态 ───────────────────────────────────────
let ws = null;
let gameState = null;
let selectedCardId = null;
let myPlayerId = null;
let myName = null;
let currentRoomId = null;
let pendingBlackCardId = null; // 等待选色的黑牌 ID

// ─── DOM 元素 ────────────────────────────────────
const $ = (id) => document.getElementById(id);

const el = {
  // 大厅
  lobby: $('lobby'),
  nicknameInput: $('nicknameInput'),
  roomIdInput: $('roomIdInput'),
  joinBtn: $('joinBtn'),
  createRoomBtn: $('createRoomBtn'),
  roomList: $('roomList'),
  // 游戏区
  gameArea: $('gameArea'),
  roomTag: $('roomTag'),
  myNameTag: $('myNameTag'),
  leaveBtn: $('leaveBtn'),
  playersBar: $('playersBar'),
  waitingPanel: $('waitingPanel'),
  waitingText: $('waitingText'),
  startGameBtn: $('startGameBtn'),
  playingArea: $('playingArea'),
  topCardDisplay: $('topCardDisplay'),
  chosenColorDisplay: $('chosenColorDisplay'),
  actionLog: $('actionLog'),
  drawBtn: $('drawBtn'),
  playBtn: $('playBtn'),
  hand: $('hand'),
  // 弹窗
  colorModal: $('colorModal'),
  winOverlay: $('winOverlay'),
  winnerText: $('winnerText'),
  backToLobbyBtn: $('backToLobbyBtn'),
  toast: $('toast'),
};

// ─── Toast 提示 ──────────────────────────────────
let toastTimer = null;
function showToast(msg) {
  el.toast.textContent = msg;
  el.toast.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.toast.classList.remove('show'), 3000);
}

// ─── 视图切换 ────────────────────────────────────
function showLobby() {
  el.lobby.classList.add('active');
  el.gameArea.classList.remove('active');
  loadRoomList();
}

function showGame() {
  el.lobby.classList.remove('active');
  el.gameArea.classList.add('active');
}

// ─── 房间列表 ────────────────────────────────────
async function loadRoomList() {
  try {
    const resp = await fetch('/api/rooms');
    const rooms = await resp.json();
    renderRoomList(rooms);
  } catch {
    el.roomList.innerHTML = '<div class="empty-state">无法加载房间列表</div>';
  }
}

function renderRoomList(rooms) {
  if (!rooms || rooms.length === 0) {
    el.roomList.innerHTML = '<div class="empty-state">暂无房间，请创建一个</div>';
    return;
  }
  el.roomList.innerHTML = '';
  rooms.forEach(r => {
    const item = document.createElement('div');
    item.className = 'room-item';
    const statusMap = { waiting: '等待中', playing: '游戏中', finished: '已结束' };
    const badgeClass = `badge badge-${r.status}`;
    item.innerHTML = `
      <span class="room-id">${r.id}</span>
      <div class="room-meta">
        <span>👥 ${r.playerNum} 人</span>
        <span class="${badgeClass}">${statusMap[r.status] || r.status}</span>
      </div>
    `;
    item.addEventListener('click', () => {
      el.roomIdInput.value = r.id;
    });
    el.roomList.appendChild(item);
  });
}

// ─── 创建房间 ────────────────────────────────────
async function createRoom() {
  el.createRoomBtn.disabled = true;
  try {
    const resp = await fetch('/api/rooms', { method: 'POST' });
    const data = await resp.json();
    if (data && data.roomId) {
      el.roomIdInput.value = data.roomId;
      showToast(`房间 ${data.roomId} 创建成功！`);
      loadRoomList();
    }
  } catch {
    showToast('创建房间失败');
  } finally {
    el.createRoomBtn.disabled = false;
  }
}

// ─── WebSocket 连接 ──────────────────────────────
function connectAndJoin() {
  const name = el.nicknameInput.value.trim();
  if (!name) {
    showToast('请输入昵称');
    return;
  }
  const roomId = (el.roomIdInput.value || 'ROOM01').trim().toUpperCase();
  el.roomIdInput.value = roomId;
  currentRoomId = roomId;
  myName = name;

  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const url = `${protocol}//${location.host}/ws?roomId=${encodeURIComponent(roomId)}`;

  if (ws) {
    ws.close();
  }

  ws = new WebSocket(url);
  el.joinBtn.disabled = true;

  ws.onopen = () => {
    // 连接后发送 JOIN
    sendMsg({ type: 'JOIN', name: name });
  };

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      handleMessage(msg);
    } catch (err) {
      showToast('消息解析失败');
    }
  };

  ws.onclose = () => {
    el.joinBtn.disabled = false;
    if (el.gameArea.classList.contains('active')) {
      showToast('连接已断开');
      showLobby();
    }
    ws = null;
    myPlayerId = null;
    gameState = null;
  };

  ws.onerror = () => {
    showToast('网络错误，请检查服务器是否启动');
  };
}

function sendMsg(payload) {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    showToast('未连接');
    return;
  }
  ws.send(JSON.stringify(payload));
}

// ─── 消息处理 ────────────────────────────────────
function handleMessage(msg) {
  switch (msg.type) {
    case 'JOINED':
      myPlayerId = msg.data.playerId;
      myName = msg.data.name;
      el.roomTag.textContent = `房间: ${currentRoomId}`;
      el.myNameTag.textContent = `${myName} (${myPlayerId})`;
      showGame();
      break;

    case 'ROOM_STATE':
      gameState = msg.data;
      selectedCardId = null;
      render();
      break;

    case 'ERROR':
      showToast(msg.msg || '未知错误');
      break;

    default:
      break;
  }
}

// ─── 渲染 ─────────────────────────────────────────
function render() {
  if (!gameState) return;

  renderPlayers();

  if (gameState.status === 'waiting') {
    el.waitingPanel.style.display = '';
    el.playingArea.style.display = 'none';
    el.hand.innerHTML = '';

    const count = gameState.players ? gameState.players.length : 0;
    el.waitingText.textContent = `当前 ${count} 人，至少需要2名玩家`;

    // 只有房主能看到开始按钮
    const me = gameState.players?.find(p => p.id === myPlayerId);
    el.startGameBtn.style.display = me && me.isHost ? '' : 'none';
    el.startGameBtn.disabled = count < 2;
  } else if (gameState.status === 'playing') {
    el.waitingPanel.style.display = 'none';
    el.playingArea.style.display = '';
    renderTopCard();
    renderHand();
    renderActions();
    el.actionLog.textContent = gameState.lastAction || '';
  } else if (gameState.status === 'finished') {
    el.waitingPanel.style.display = 'none';
    el.playingArea.style.display = '';
    renderTopCard();
    renderHand();

    // 找到获胜者名字
    const winner = gameState.players?.find(p => p.id === gameState.winner);
    el.winnerText.textContent = winner ? `${winner.name} 获得了胜利！` : '游戏结束';
    el.winOverlay.classList.add('active');
  }
}

function renderPlayers() {
  el.playersBar.innerHTML = '';
  if (!gameState || !gameState.players) return;

  gameState.players.forEach((p, idx) => {
    const div = document.createElement('div');
    div.className = 'player-card';
    if (idx === gameState.nowIdx && gameState.status === 'playing') {
      div.classList.add('current-turn');
    }
    if (p.id === myPlayerId) {
      div.classList.add('is-me');
    }

    let extra = '';
    if (p.isHost) extra += '<span class="p-host">👑房主</span>';
    if (!p.online) extra += '<span class="p-offline">离线</span>';

    const turnIcon = (idx === gameState.nowIdx && gameState.status === 'playing') ? '▶ ' : '';

    div.innerHTML = `
      <span class="p-name">${turnIcon}${p.name}${extra}</span>
      <span class="p-count">🃏 ${p.cardCount} 张</span>
    `;
    el.playersBar.appendChild(div);
  });
}

function renderTopCard() {
  el.topCardDisplay.innerHTML = '';
  if (!gameState || !gameState.topCard) return;

  const card = gameState.topCard;
  const node = createCardNode(card, false);
  node.classList.add('table-top-card');
  node.style.cursor = 'default';
  el.topCardDisplay.appendChild(node);

  // 显示选择的颜色
  if (gameState.chosenColor) {
    const colorNames = { Red: '红色', Yellow: '黄色', Blue: '蓝色', Green: '绿色' };
    el.chosenColorDisplay.textContent = `指定颜色: ${colorNames[gameState.chosenColor] || gameState.chosenColor}`;
    el.chosenColorDisplay.style.color = getColorCSS(gameState.chosenColor);
  } else {
    el.chosenColorDisplay.textContent = '';
  }
}

function renderHand() {
  el.hand.innerHTML = '';
  if (!gameState || !gameState.yourCards) {
    el.hand.innerHTML = '<div class="empty-state">暂无手牌</div>';
    return;
  }

  gameState.yourCards.forEach(c => {
    const node = createCardNode(c, true);
    if (selectedCardId === c.id) {
      node.classList.add('selected');
    }
    node.addEventListener('click', () => {
      selectedCardId = selectedCardId === c.id ? null : c.id;
      renderHand();
      renderActions();
    });
    el.hand.appendChild(node);
  });
}

function renderActions() {
  if (!gameState) return;
  const isMyTurn = gameState.yourIndex === gameState.nowIdx && gameState.status === 'playing';
  el.drawBtn.disabled = !isMyTurn;
  el.playBtn.disabled = !isMyTurn || selectedCardId === null;
}

// ─── 卡牌渲染 ────────────────────────────────────
function createCardNode(card, selectable) {
  const div = document.createElement('div');
  div.className = `uno-card c-${card.color}`;
  div.dataset.id = String(card.id);

  const idTag = document.createElement('small');
  idTag.textContent = `#${card.id}`;
  div.appendChild(idTag);

  const valMap = {
    'Skip': '🚫',
    'Reverse': '🔄',
    '+2': '+2',
    '+4': '+4',
    'Wild': '🌈',
  };
  const displayVal = valMap[card.val] || card.val;

  const span = document.createElement('span');
  span.className = 'card-text';
  span.textContent = displayVal;
  div.appendChild(span);

  return div;
}

function getColorCSS(color) {
  const map = { Red: '#ef4444', Yellow: '#fbbf24', Blue: '#3b82f6', Green: '#22c55e' };
  return map[color] || '#888';
}

// ─── 选色弹窗 ────────────────────────────────────
function showColorPicker(cardId) {
  pendingBlackCardId = cardId;
  el.colorModal.classList.add('active');
}

function hideColorPicker() {
  el.colorModal.classList.remove('active');
  pendingBlackCardId = null;
}

document.querySelectorAll('.color-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    if (pendingBlackCardId !== null) {
      sendMsg({ type: 'PLAY', cardId: pendingBlackCardId, chosenColor: btn.dataset.color });
      selectedCardId = null;
      hideColorPicker();
    }
  });
});

// ─── 事件绑定 ────────────────────────────────────
el.joinBtn.addEventListener('click', connectAndJoin);
el.createRoomBtn.addEventListener('click', createRoom);

el.nicknameInput.addEventListener('keydown', (e) => {
  if (e.key === 'Enter') connectAndJoin();
});

el.startGameBtn.addEventListener('click', () => {
  sendMsg({ type: 'START' });
});

el.playBtn.addEventListener('click', () => {
  if (selectedCardId === null) {
    showToast('请先选择一张手牌');
    return;
  }
  // 检查是否是黑牌，需要选色
  const card = gameState?.yourCards?.find(c => c.id === selectedCardId);
  if (card && card.color === 'Black') {
    showColorPicker(selectedCardId);
    return;
  }
  sendMsg({ type: 'PLAY', cardId: selectedCardId });
  selectedCardId = null;
});

el.drawBtn.addEventListener('click', () => {
  sendMsg({ type: 'DRAW' });
});


el.leaveBtn.addEventListener('click', () => {
  if (ws) ws.close();
  ws = null;
  myPlayerId = null;
  gameState = null;
  selectedCardId = null;
  showLobby();
});

el.backToLobbyBtn.addEventListener('click', () => {
  el.winOverlay.classList.remove('active');
  if (ws) ws.close();
  ws = null;
  myPlayerId = null;
  gameState = null;
  showLobby();
});

// ─── 初始化 ──────────────────────────────────────
showLobby();
// 定时刷新房间列表
setInterval(() => {
  if (el.lobby.classList.contains('active')) {
    loadRoomList();
  }
}, 5000);
