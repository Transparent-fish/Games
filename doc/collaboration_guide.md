# 开发者协作指南 (Collaboration Guide)

欢迎加入 **baogame** 的开发协作！这份文档旨在为新加入的开发者提供技术概览、核心机制说明以及开发规范，帮助你快速上手。

---

## 1. 技术栈概览
- **服务端**: Node.js (Express + Socket.io)
- **前端**: HTML5 Canvas + Vue.js (仅限 UI)
- **协议层**: 自定义二进制序列化 (JPack)
- **运行环境**: 原生 JavaScript (ES6+)，尽量避免复杂的依赖。

---

## 2. 核心系统说明

### 2.1 同步机制 (Tick-based Sync)
游戏采用服务端权威架构。服务端每隔约 **17ms (60FPS)** 执行一次 `update` 循环：
1. **输入处理**: 接收客户端发送的按键包。
2. **状态更新**: 更新所有玩家、物品、实体的坐标和状态。
3. **碰撞检测**: 处理玩家间的推搡、道具拾取、伤害判定。
4. **状态广播**: 将当前 Tick 的所有数据打包发送给所有客户端。

### 2.2 网络协议 (JPack)
为了节省带宽，游戏并未使用纯 JSON 传输状态，而是通过 `static/js/JPack.js` 进行序列化：
- **Schema 定义**: 必须在 `JPack.js` 中两端一致定义 `userPack`、`controlPack` 等结构。
- **添加字段**: 
  1. 在 `JPack.js` 的 Schema 中添加字段。
  2. 在服务端的 `getData` 方法中确保返回该字段。
  3. 在客户端的渲染逻辑中读取该字段。

---

## 3. 技能系统开发指南

以新增的“虚化”技能为例，开发一个新功能通常涉及以下步骤：

### 第一步：按键捕获 (`pcController.js`)
监听键盘事件（如 `e.keyCode == 69`），将状态存储在全局的 `p1` 对象中。
```javascript
// keydown 时记录 Press 和 Down
p1.ePress = true;
p1.eDown = true;
```

### 第二步：更新协议 (`JPack.js`)
在 `controlPack` 中添加按键标志，在 `userPack` 中添加技能状态（如特效计时器）。

### 第三步：服务端逻辑 (`user.js`)
在 `User.prototype.update` 中根据接收到的按键执行逻辑：
- **计时器**: 减少技能持续时间或冷却时间。
- **状态切换**: 修改 `this.phasing` 或 `this.vx` 等属性。
- **伤害免疫**: 在 `killed` 函数中拦截死亡逻辑。

### 第四步：前端渲染 (`game.js`)
在 `drawUser` 函数中根据玩家状态添加视觉反馈：
```javascript
if (user.phasing > 0) {
    ctx.globalAlpha = 0.5; // 设置透明度
}
```

---

## 4. 代码风格与规范
- **坐标系**: 游戏原点在左下角，Canvas 渲染时通过 `P.h - y` 转换为屏幕坐标。
- **面向对象**: 逻辑类（User, Item, Entity）应保持职责单一，核心计算留在服务端。
- **性能**: 避免在 `update` 循环中进行复杂的对象创建，尽量复用对象。
- **中文注释/文档**: 请保持沟通使用中文，文档也优先提供中文版本。

---

## 5. 常用开发调试工具
- **/admin 界面**: 访问 `http://localhost:8030/admin`（需在 localStorage 设置管理员口令）。
- **控制台调试**: 通过 `app.logs` 在游戏内查看实时通知。

---

## 协作流程
1. **Fork 本仓库**。
2. **在分支上开发**。
3. **提交 PR 前进行验证**: 确保地图加载正常、多玩家连接不掉帧。

如有疑问，请查阅 `doc/file_explanation.md` 或直接询问项目负责人。
