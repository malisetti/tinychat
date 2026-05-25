(function () {
  const params = new URLSearchParams(location.search);
  let displayName = params.get("name") || "anon" + Math.floor(Math.random() * 1000);
  const baseTitle = "Tinychat";
  let unread = 0;
  let ws = null;
  let reconnectAttempt = 0;
  let reconnectTimer = null;
  let intentionalClose = false;

  const logEl = document.getElementById("log");
  const userListEl = document.getElementById("user-list");
  const form = document.getElementById("chat-form");
  const input = document.getElementById("input");
  const appTitle = document.getElementById("app-title");

  function wsURL() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    return proto + "//" + location.host + "/ws?name=" + encodeURIComponent(displayName);
  }

  function setTitle() {
    const badge = unread > 0 ? " (" + unread + ")" : "";
    document.title = baseTitle + badge;
    appTitle.textContent = baseTitle + badge;
  }

  function markUnread() {
    if (document.visibilityState === "visible") return;
    unread++;
    setTitle();
  }

  function clearUnread() {
    unread = 0;
    setTitle();
  }

  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible") clearUnread();
  });

  function updateUserList(users) {
    userListEl.replaceChildren();
    (users || []).slice().sort().forEach(function (name) {
      const li = document.createElement("li");
      li.textContent = name;
      li.setAttribute("role", "listitem");
      userListEl.appendChild(li);
    });
  }

  function appendLine(className, html) {
    const div = document.createElement("div");
    div.className = className;
    div.innerHTML = html;
    logEl.appendChild(div);
    logEl.scrollTop = logEl.scrollHeight;
  }

  function appendSystem(text) {
    appendLine("system", escapeHTML(text));
  }

  function appendChat(from, text) {
    appendLine("chat", '<span class="from">' + escapeHTML(from) + ":</span> " + escapeHTML(text));
    markUnread();
  }

  function escapeHTML(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function backoffMs(attempt) {
    const ms = Math.min(1000 * Math.pow(2, attempt), 30000);
    return ms;
  }

  function scheduleReconnect() {
    if (intentionalClose) return;
    const delay = backoffMs(reconnectAttempt);
    reconnectAttempt++;
    appendSystem("Disconnected. Reconnecting in " + delay / 1000 + "s…");
    reconnectTimer = setTimeout(connect, delay);
  }

  function connect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    ws = new WebSocket(wsURL());
    ws.onopen = function () {
      reconnectAttempt = 0;
      appendSystem("Connected as " + displayName);
    };
    ws.onmessage = function (ev) {
      let msg;
      try {
        msg = JSON.parse(ev.data);
      } catch (_) {
        return;
      }
      if (msg.type === "presence") {
        updateUserList(msg.users);
        return;
      }
      if (msg.type === "system") {
        appendSystem(msg.text);
        return;
      }
      if (msg.type === "chat") {
        appendChat(msg.from, msg.text);
      }
    };
    ws.onclose = function () {
      scheduleReconnect();
    };
    ws.onerror = function () {
      try {
        ws.close();
      } catch (_) {}
    };
  }

  function sendChat(text) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      appendSystem("Not connected.");
      return;
    }
    ws.send(JSON.stringify({ type: "chat", text: text }));
  }

  function sendRename(newName) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    displayName = newName;
    ws.send(JSON.stringify({ type: "rename", name: newName }));
    const url = new URL(location.href);
    url.searchParams.set("name", newName);
    history.replaceState(null, "", url);
  }

  function handleCommand(raw) {
    const parts = raw.trim().split(/\s+/);
    const cmd = parts[0].toLowerCase();
    if (cmd === "/who") {
      const names = Array.from(userListEl.querySelectorAll("li")).map(function (li) {
        return li.textContent;
      });
      appendSystem("Online: " + (names.length ? names.join(", ") : "(none)"));
      return true;
    }
    if (cmd === "/clear") {
      logEl.replaceChildren();
      return true;
    }
    if (cmd === "/name" && parts.length >= 2) {
      sendRename(parts.slice(1).join(" "));
      return true;
    }
    return false;
  }

  form.addEventListener("submit", function (e) {
    e.preventDefault();
    const text = input.value.trim();
    if (!text) return;
    input.value = "";
    if (text.startsWith("/")) {
      if (!handleCommand(text)) appendSystem("Unknown command. Try /who, /clear, /name <new>");
      return;
    }
    sendChat(text);
  });

  setTitle();
  connect();
})();
