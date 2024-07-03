document.addEventListener("DOMContentLoaded", function () {
  const username = localStorage.getItem("username");

  if (!username) {
    window.location.href = "/";
  }

  let conn;
  let busy = false;

  const msg = document.getElementById("msg");
  const log = document.getElementById("log");
  const stopButton = document.getElementById("stop");
  const sendButton = document.getElementById("sendBtn");

  function appendLog(item) {
    const doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
    log.appendChild(item);
    if (doScroll) {
      log.scrollTop = log.scrollHeight - log.clientHeight;
    }
  }

  function updateButtonsState() {
    stopButton.disabled = !busy;
    sendButton.disabled = !busy;
    stopButton.classList.toggle("enabled", busy);
    stopButton.classList.toggle("disabled", !busy);
  }

  function handleStopButton(event) {
    event.preventDefault();
    if (conn) {
      conn.send("[STOP]");
      updateButtonsState();
    }

    const item = document.createElement("div");
    item.innerHTML = "<b>You have left the chat</b>";
    item.style.color = "#9e9e9e";
    item.style.marginTop = "20px";
    appendLog(item);
  }

  function handleSendButton(event) {
    event.preventDefault();
    if (!conn || !msg.value.trim()) return false;

    const item = document.createElement("div");
    item.innerText = msg.value;
    item.classList.add("sentMsg");
    appendLog(item);
    conn.send(msg.value);
    msg.value = "";
    return false;
  }

  function handleMessage(evt) {
    const message = evt.data;
    const item = document.createElement("div");

    if (message.startsWith("[NOTI1]:")) {
      busy = false;
      item.innerHTML = `<b>${message.substring(8)} has left the chat</b>`;
      item.style.color = "#9e9e9e";
      item.style.margin = "20px";
      item.style.display = "block";
      appendLog(item);
    } else if (message.startsWith("[CONNECT]:")) {
      busy = true;
      const partnerName = message.substring(10);
      item.innerHTML = `<b>You are now connected to ${partnerName}</b>`;
      item.style.color = "#9e9e9e";
      item.style.marginBottom = "20px";
      item.style.display = "block";
      appendLog(item);
    } else if (message.startsWith("[MSG]:")) {
      item.innerHTML = message.substring(6);
      item.classList.add("receivedMsg");
      appendLog(item);
    } else if (message.startsWith("[COUNT]:")) {
      const count = message.split(":")[1];
      document.getElementById("count").innerText = count;
    }

    updateButtonsState();
  }

  function initializeWebSocket() {
    conn = new WebSocket("ws://" + document.location.host + "/ws");

    conn.onopen = function () {
      conn.send(username);
    };

    conn.onclose = function () {
      localStorage.clear();
      const item = document.createElement("div");
      item.innerHTML = "<b>Connection closed.</b>";
      appendLog(item);
    };

    conn.onerror = function (error) {
      console.error("WebSocket error:", error);
      const item = document.createElement("div");
      item.innerHTML = "<b>Error connecting to the server.</b>";
      appendLog(item);
    };

    conn.onmessage = handleMessage;
  }

  stopButton.onclick = handleStopButton;
  document.getElementById("form").onsubmit = handleSendButton;

  if (window["WebSocket"]) {
    initializeWebSocket();
  } else {
    const item = document.createElement("div");
    item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
    appendLog(item);
  }
});

document
  .getElementById("welcome-form")
  .addEventListener("submit", function (event) {
    event.preventDefault();
    const username = document.getElementById("username").value;
    localStorage.setItem("username", username);
    window.location.href = "/chatbox.html";
  });
