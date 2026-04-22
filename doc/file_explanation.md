# 项目文件说明文档

这份文档旨在解释本项目中各个文件的主要功能，帮助开发者快速理解游戏架构。

## 核心架构
本项目采用 **Node.js (后端)** + **Canvas/JavaScript (前端)** 的架构，通过 Socket.io 进行实时通信。

---

## 后端逻辑 (`/game`)

### 1. [app.js](file:///d:/baogame/app.js)
项目的入口文件。初始化 Express 服务器、配置 Socket.io 并启动 HTTP 服务。

### 2. [game.js](file:///d:/baogame/game/game.js)
核心游戏管理类。
- 管理所有玩家（Users）、物品（Items）和实体（Entities）。
- 包含主游戏循环（Update），处理全局状态更新和碰撞检测分发。
- 负责向所有客户端分发 Tick 数据。

### 3. [user.js](file:///d:/baogame/game/user.js)
玩家逻辑类。
- 定义玩家的状态（站立、坠落、虚化、死亡等）。
- 处理玩家的移动、技能、射击和死亡逻辑。

### 4. [client.js](file:///d:/baogame/game/client.js)
客户端连接管理。
- 处理 Socket 连接和断开。
- 负责控制包（Control Pack）的接收与解析，将其映射到对应的 `User` 对象。

### 5. [collide.js](file:///d:/baogame/game/collide.js)
碰撞检测系统。
- 定义玩家与玩家、玩家与物品之间的碰撞逻辑。

### 6. [map.js](file:///d:/baogame/game/map.js) 及 [/maps](file:///d:/baogame/game/maps)
地图加载与管理逻辑，存储具体关卡的布局数据。

---

## 前端逻辑 (`/static`)

### 1. [index.html](file:///d:/baogame/static/index.html)
游戏的主页面容器。

### 2. [js/game.js](file:///d:/baogame/static/js/game.js)
前端引擎逻辑。
- 负责 Canvas 渲染循环。
- 解析来自服务器的 Tick 数据并绘制玩家、物品和特效。
- 通过 Vue.js 管理简单的 UI 界面（如消息、玩家列表）。

### 3. [js/pcController.js](file:///d:/baogame/static/js/pcController.js)
按键监听器。捕获键盘输入（WASD, Q, Space, E）并将其打包发送给服务器。

### 4. [js/JPack.js](file:///d:/baogame/static/js/JPack.js)
二进制协议序列化工具。
- 定义了服务器与客户端通信时使用的数据包结构（Schema），以节省带宽并提高同步效率。

---

## 其他目录
- `/doc`: 项目相关文档。
- `/logs`: 服务器日志。
- `/static/imgs`: 游戏图片资源。
- `/game/ai`: NPC 的 AI 控制逻辑。
